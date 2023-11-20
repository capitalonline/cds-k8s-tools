package haproxy

import (
	"cds-k8s-tools/pkg/client"
	"cds-k8s-tools/pkg/haproxy/api"
	"encoding/json"
	"fmt"
	"github.com/gogf/gf/v2/util/gconv"
	log "github.com/sirupsen/logrus"
	"strconv"
	"time"
)

const (
	CustomerConfigMapName = "customer-haproxy-config"
	ConfigMapDataKey      = "haproxy-instances"
	DefaultNameSpace      = "kube-system"
	ComputeNodeLabel      = "node-role.kubernetes.io/compute"
)

type HaConfigs struct {
	Instances []HaConfigInfo `json:"lb2port"`
}

type HaConfigInfo struct {
	LbTag        string `json:"lb_tag"`
	NameSpace    string `json:"namespace"`
	ServiceName  string `json:"service_name"`
	MaxConn      int    `json:"max_conn"`
	NodePort     int    `json:"node_port"`
	ListenerPort []int  `json:"listener_ports"`
}

func UpdateHaproxyInstance() (err error) {
	return
}

func ModifyHaproxyConfig(instanceId string, haInfo HaConfigInfo, newNodeIpList []string, NewNodeIpPodMap map[string]int64) {

	var (
		UseOldHaConfig        = true
		newBackendServers     []api.BackendServer
		ChangeNewTcpListeners []api.HttpListener
	)

	log.Infof("cluster current node ip: %+v", newNodeIpList)

	req := map[string]string{"InstanceUuid": instanceId}
	HaInstancePolicy, err := api.DescribeLoadBalancerStrategy(req)
	if err != nil {
		log.Errorf("describeLoadBalancerStrategy failed: %s", err)
		return
	}

	// all user only uses http policies
	httpListeners := HaInstancePolicy.HttpListeners
	if len(httpListeners) == 0 {
		log.Errorf("haproxy instance don't have http type policy")
		return
	}

	// compute old backend for NodeIpMap
	OldNodeIpPodMap := make(map[string]int64)
	oldBackendServers := httpListeners[0].BackendServer
	for _, backendServer := range oldBackendServers {
		num, _ := strconv.ParseInt(backendServer.Weight, 10, 64)
		OldNodeIpPodMap[backendServer.IP] = num
	}

	log.Infof("old nodeIp podNum map: %+v", OldNodeIpPodMap)
	log.Infof("new nodeIp podNum map: %+v", NewNodeIpPodMap)

	for _, nodeIp := range newNodeIpList {
		newBackendServers = append(newBackendServers, api.BackendServer{
			IP:   nodeIp,
			Port: haInfo.NodePort,
			// Weight default value: 1
			Weight:  gconv.String(NewNodeIpPodMap[nodeIp]),
			MaxConn: haInfo.MaxConn,
		})
	}
	log.Infof("old haproxy backend instance servers: %+v", oldBackendServers)
	log.Infof("new haproxy backend instance servers: %+v", newBackendServers)

	if len(oldBackendServers) == len(newBackendServers) {
		for _, newServerIp := range newNodeIpList {
			// Check whether new backend servers exists
			_, ok := OldNodeIpPodMap[newServerIp]
			if !ok {
				UseOldHaConfig = false
				log.Infof("new backend server exists, send task to update Haproxy policy.")
				break
			}

			// Check whether the weight of the backend server changes
			if NewNodeIpPodMap[newServerIp] != OldNodeIpPodMap[newServerIp] {
				UseOldHaConfig = false
				log.Infof("backend server weight changed, send task to update Haproxy policy.")
				break
			}
		}
	} else {
		UseOldHaConfig = false
		log.Infof("the number of backend servers is inconsistent, send task to update Haproxy policy.")
	}

	if !UseOldHaConfig {
		// Generate the latest update haproxy params
		for _, policy := range HaInstancePolicy.HttpListeners {
			for _, port := range haInfo.ListenerPort {
				if policy.ListenerPort == port {
					policy.BackendServer = newBackendServers
				}
			}
			ChangeNewTcpListeners = append(ChangeNewTcpListeners, policy)
		}
		HaInstancePolicy.HttpListeners = ChangeNewTcpListeners
		log.Infof("***begin to update harproxy instance, latest policy params: %+v", HaInstancePolicy)

		// send to task for update haproxy policy
		modifyParams := api.ModifyHaStrategyReq{
			InstanceUuid:       instanceId,
			HaStrategyInfoData: HaInstancePolicy,
		}
		res, err := api.ModifyLoadBalancerStrategy(modifyParams)
		if err != nil {
			log.Errorf("send ModifyLoadBalancerStrategy task failed: %s", err)
			return
		}
		log.Infof("taskId: %d sent successfully", res.TaskId)
		time.Sleep(2 * time.Second)
	}
	log.Infof("***no need to update the backend server by haInstanceId: %s", instanceId)
}

func CheckClusterIpNodeByHaConfig(config HaConfigInfo) error {
	log.Infof("begin to search haproxy instances by tag: %s", config.LbTag)

	// Count the number of existing workers
	// var workerIpList []string
	// workers := client.Sa.GetAllWorkerNode(ComputeNodeLabel)
	// for _, node := range workers.Items {
	// 	for _, address := range node.Status.Addresses {
	// 		if address.Type == v1.NodeInternalIP {
	// 			workerIpList = append(workerIpList, address.Address)
	// 		}
	// 	}
	// }

	// Count the number of Pods on each worker
	var workerIpList []string
	IpPodNumMap := make(map[string]int64)

	podsByServiceName := client.Sa.GetPodByServiceName(config.ServiceName, config.NameSpace)
	if podsByServiceName == nil {
		return nil
	}
	for _, pod := range podsByServiceName.Items {
		nodeIp := pod.Status.HostIP
		workerIpList = append(workerIpList, nodeIp)
		if _, ok := IpPodNumMap[nodeIp]; ok {
			IpPodNumMap[nodeIp]++
		} else {
			IpPodNumMap[nodeIp] = 1
		}
	}

	// search haproxy instance describe info by ha tag
	req := map[string]string{"TagName": config.LbTag}
	haproxyInstances, err := api.DescribeHaproxyInstancesByTag(req)
	if err != nil {
		return err
	}

	// send modify haproxy task
	for _, instance := range haproxyInstances {
		ModifyHaproxyConfig(instance.InstanceUuid, config, workerIpList, IpPodNumMap)
	}
	log.Infof("no search haproxy instance by tagName: %s", config.LbTag)
	return nil
}

func UpdateNodePod() error {
	// 1. get haproxy instance by configmap
	configMap := client.Sa.GetConfigMapByName(CustomerConfigMapName, DefaultNameSpace)
	if configMap == nil {
		log.Infof("there is no configmap by name %s in namespace %s", CustomerConfigMapName, DefaultNameSpace)
		return nil
	}
	dataMapStr, ok := configMap.Data[ConfigMapDataKey]
	if !ok {
		return fmt.Errorf("failed to get configmap")
	}
	haproxyInstances := HaConfigs{}
	err := json.Unmarshal([]byte(dataMapStr), &haproxyInstances)
	if err != nil {
		return fmt.Errorf("failed to get configmap")
	}
	for _, haConfig := range haproxyInstances.Instances {
		err := CheckClusterIpNodeByHaConfig(haConfig)
		if err != nil {
			return fmt.Errorf("failed to get configmap")
		}
	}
	return nil
}
