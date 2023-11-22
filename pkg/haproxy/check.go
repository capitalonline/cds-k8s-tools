package haproxy

import (
	"cds-k8s-tools/pkg/client"
	"cds-k8s-tools/pkg/haproxy/api"
	"encoding/json"
	"fmt"
	"github.com/gogf/gf/v2/util/gconv"
	log "github.com/sirupsen/logrus"
	"reflect"
	"strconv"
	"sync"
)

const (
	CustomerConfigMapName = "customer-haproxy-config"
	ConfigMapDataKey      = "haproxy-instances"
	DefaultNameSpace      = "kube-system"
	workerLabel           = "node-role.kubernetes.io/compute"
)

var (
	SvcNameInstanceIdsMap = new(sync.Map)
	InstanceIdPolicyMap   = new(sync.Map)
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

func GetCustomerHaproxyConfigs() (haproxyConfigs *HaConfigs, err error) {
	configMap := client.Sa.GetConfigMapByName(CustomerConfigMapName, DefaultNameSpace)
	if configMap == nil {
		return nil, fmt.Errorf("no found configmap name %s in namespace %s", CustomerConfigMapName, DefaultNameSpace)
	}

	dataMapStr, ok := configMap.Data[ConfigMapDataKey]
	if !ok {
		return nil, fmt.Errorf("not found data by key %s in comfigmap %s", ConfigMapDataKey, CustomerConfigMapName)
	}

	haproxyConfigs = &HaConfigs{}
	err = json.Unmarshal([]byte(dataMapStr), haproxyConfigs)
	if err != nil {
		return nil, fmt.Errorf("configmap parameter define error, err: %s", err)
	}
	return
}

func verifyUserConfigMap(CustomerHaConfigs *HaConfigs) error {
	verifyMap := make(map[string]bool)
	for _, Config := range CustomerHaConfigs.Instances {
		if verifyMap[Config.ServiceName] {
			return fmt.Errorf("service Name repeated in user haproxy-instance configmap")
		}
		verifyMap[Config.ServiceName] = true

		if verifyMap[Config.LbTag] {
			return fmt.Errorf("lb tag repeated in user haproxy-instance configmap")
		}
		verifyMap[Config.LbTag] = true
	}
	return nil
}

func UpdateHaproxyInstance() error {
	CustomerHaConfigs, err := GetCustomerHaproxyConfigs()
	if err != nil {
		log.Errorf("err: %s", err)
		return nil
	}

	// verify customer params: tagName and ServiceName
	err = verifyUserConfigMap(CustomerHaConfigs)
	if err != nil {
		log.Errorf("err: %s", err)
		return nil
	}

	for _, Config := range CustomerHaConfigs.Instances {
		log.Infof("begin check haproxy instance by serviceName %s", Config.ServiceName)

		req := map[string]string{"TagName": Config.LbTag}
		Instances, err := api.DescribeHaproxyInstancesByTag(req)
		if err != nil {
			log.Errorf("DescribeHaproxyInstancesByTag failed: %s", err)
			return nil
		}

		if len(Instances) == 0 {
			log.Infof("not found haproxy instances by tag name %s, serviceName %s", Config.LbTag, Config.ServiceName)
			continue
		}

		var NewSvcInstancesIds []string
		for _, item := range Instances {
			NewSvcInstancesIds = append(NewSvcInstancesIds, item.InstanceUuid)
		}

		// check old instanceIds cache exists
		if OldInstancesIds, ok := SvcNameInstanceIdsMap.Load(Config.ServiceName); ok {
			log.Infof("get cache in SvcNameInstanceIdsMap by ServiceName: %s", Config.ServiceName)

			if equal := reflect.DeepEqual(NewSvcInstancesIds, OldInstancesIds.([]string)); equal {
				log.Infof("Haproxy instances %+v no changed by ServiceName: %s", NewSvcInstancesIds, Config.ServiceName)
				continue
			}

			log.Infof("instances nums has changed, check or update backend server for haproxy policy")
			err := CheckClusterIpNodeByHaConfig(Config, NewSvcInstancesIds)
			if err != nil {
				log.Errorf("err: %s", err)
				return nil
			}

		}
		log.Infof("insert NewSvcInstancesIds into SvcNameInstanceIdsMap for serviceName %s", Config.ServiceName)
		SvcNameInstanceIdsMap.Store(Config.ServiceName, NewSvcInstancesIds)
	}

	return nil
}

func ModifyHaproxyConfigCLusterMode(instanceId string, haInfo HaConfigInfo, newNodeIpList []string) {
	var (
		InstancePolicy         *api.HaStrategyInfoData
		oldNodeIpList          []string
		newBackendServers      []api.BackendServer
		ChangeNewHttpListeners []api.HttpListener
	)

	log.Infof("cluster current node ip: %+v", newNodeIpList)

	if OldInstancePolicy, ok := InstanceIdPolicyMap.Load(instanceId); ok {
		log.Infof("get cache in InstanceIdPolicyMap by instanceId: %s", instanceId)
		InstancePolicy = OldInstancePolicy.(*api.HaStrategyInfoData)
	} else {
		req := map[string]string{"InstanceUuid": instanceId}
		HaInstancePolicy, err := api.DescribeLoadBalancerStrategy(req)
		if err != nil {
			log.Errorf("DescribeLoadBalancerStrategy failed: %s", err)
			return
		}
		InstancePolicy = HaInstancePolicy
	}

	// all user only uses http policies
	httpListeners := InstancePolicy.HttpListeners
	if len(httpListeners) == 0 {
		log.Errorf("haproxy instance don't have http type policy")
		return
	}

	oldBackendServers := httpListeners[0].BackendServer
	for _, backendServer := range oldBackendServers {
		oldNodeIpList = append(oldNodeIpList, backendServer.IP)
	}

	if equal := reflect.DeepEqual(oldNodeIpList, newNodeIpList); equal {
		InstanceIdPolicyMap.Store(instanceId, InstancePolicy)
		log.Infof("*** end *** haproxy backend server no changed, continue")
		return
	}

	for _, nodeIp := range newNodeIpList {
		newBackendServers = append(newBackendServers, api.BackendServer{
			IP:      nodeIp,
			Port:    haInfo.NodePort,
			Weight:  "1",
			MaxConn: haInfo.MaxConn,
		})
	}

	for _, policy := range InstancePolicy.HttpListeners {
		for _, port := range haInfo.ListenerPort {
			if policy.ListenerPort == port {
				policy.BackendServer = newBackendServers
			}
		}
		ChangeNewHttpListeners = append(ChangeNewHttpListeners, policy)
	}
	InstancePolicy.HttpListeners = ChangeNewHttpListeners

	// send to task for update haproxy policy
	modifyParams := api.ModifyHaStrategyReq{
		InstanceUuid:       instanceId,
		HaStrategyInfoData: InstancePolicy,
	}

	res, err := api.ModifyLoadBalancerStrategy(modifyParams)
	if err != nil {
		log.Errorf("send ModifyLoadBalancerStrategy task failed: %s", err)
		return
	}

	// cache can be updated only after the task is successfully
	InstanceIdPolicyMap.Store(instanceId, InstancePolicy)

	log.Infof("taskId: %s sent successfully", res.TaskId)
	return

}

func ModifyHaproxyConfigLocalMode(instanceId string, haInfo HaConfigInfo, newNodeIpList []string, NewNodeIpPodMap map[string]int64) {

	var (
		UseOldHaConfig         = true
		OldNodeIpPodMap        = make(map[string]int64)
		newBackendServers      []api.BackendServer
		ChangeNewHttpListeners []api.HttpListener
		InstancePolicy         *api.HaStrategyInfoData
	)

	log.Infof("cluster current node ip: %+v", newNodeIpList)

	if OldInstancePolicy, ok := InstanceIdPolicyMap.Load(instanceId); ok {
		log.Infof("get cache in InstanceIdPolicyMap by instanceId: %s", instanceId)
		InstancePolicy = OldInstancePolicy.(*api.HaStrategyInfoData)
	} else {
		req := map[string]string{"InstanceUuid": instanceId}
		HaInstancePolicy, err := api.DescribeLoadBalancerStrategy(req)
		if err != nil {
			log.Errorf("describeLoadBalancerStrategy failed: %s", err)
			return
		}
		InstancePolicy = HaInstancePolicy
	}

	// all user only uses http policies
	httpListeners := InstancePolicy.HttpListeners
	if len(httpListeners) == 0 {
		log.Errorf("haproxy instance don't have http type policy")
		return
	}

	// compute old backend for NodeIpMap
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
			if _, ok := OldNodeIpPodMap[newServerIp]; !ok {
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
		log.Infof("the number of backend servers is changed, send task to update Haproxy policy.")
	}

	if !UseOldHaConfig {
		// Generate the latest update haproxy params
		for _, policy := range InstancePolicy.HttpListeners {
			for _, port := range haInfo.ListenerPort {
				if policy.ListenerPort == port {
					policy.BackendServer = newBackendServers
				}
			}
			ChangeNewHttpListeners = append(ChangeNewHttpListeners, policy)
		}
		InstancePolicy.HttpListeners = ChangeNewHttpListeners
		log.Infof("***begin to update harproxy instance, latest policy params: %+v", InstancePolicy)

		// send to task for update haproxy policy
		modifyParams := api.ModifyHaStrategyReq{
			InstanceUuid:       instanceId,
			HaStrategyInfoData: InstancePolicy,
		}

		res, err := api.ModifyLoadBalancerStrategy(modifyParams)
		if err != nil {
			log.Errorf("send ModifyLoadBalancerStrategy task failed: %s", err)
			return
		}

		// cache can be updated only after the task is successfully
		InstanceIdPolicyMap.Store(instanceId, InstancePolicy)

		log.Infof("taskId: %s sent successfully", res.TaskId)
		return
	}

	InstanceIdPolicyMap.Store(instanceId, InstancePolicy)
	log.Infof("***no need to update the backend server by haInstanceId: %s", instanceId)
}

func CheckClusterIpNodeByHaConfig(config HaConfigInfo, NewSvcInstancesIds []string) error {
	log.Infof("*** begin *** search haproxy instances by tag: %s, NewSvcInstancesIds: %+v", config.LbTag, NewSvcInstancesIds)

	var (
		workerIpList      []string
		SearchInstanceIds []string
		IpPodNumMap       = make(map[string]int64)
	)

	// search haproxy instance describeInfo by tagName
	if OldInstancesIds, ok := SvcNameInstanceIdsMap.Load(config.ServiceName); ok {
		log.Infof("get cache in SvcNameInstanceIdsMap by ServiceName: %s", config.ServiceName)
		SearchInstanceIds = OldInstancesIds.([]string)

		// from haproxy check timer and update latest instanceIds
		if len(NewSvcInstancesIds) != 0 {
			SearchInstanceIds = NewSvcInstancesIds
		}

	} else {
		req := map[string]string{"TagName": config.LbTag}
		HaproxyInstances, err := api.DescribeHaproxyInstancesByTag(req)
		if err != nil {
			return fmt.Errorf("DescribeHaproxyInstancesByTag failed: %s", err)
		}

		for _, Instance := range HaproxyInstances {
			SearchInstanceIds = append(SearchInstanceIds, Instance.InstanceUuid)
		}

		if len(SearchInstanceIds) != 0 {
			SvcNameInstanceIdsMap.Store(config.ServiceName, SearchInstanceIds)
		}

	}

	Service := client.Sa.GetService(config.ServiceName, config.NameSpace)
	if Service == nil {
		return fmt.Errorf("not found service name %s in namespace %s", config.ServiceName, config.NameSpace)
	}

	if Service.Spec.ExternalTrafficPolicy == "local" {
		// Count the number of Pods on each worker
		podsByServiceName := client.Sa.GetPodByServiceName(config.ServiceName, config.NameSpace)
		if podsByServiceName == nil {
			return fmt.Errorf("not found pods by serviceName %s", config.ServiceName)
		}

		for _, pod := range podsByServiceName.Items {
			if _, ok := IpPodNumMap[pod.Status.HostIP]; ok {
				IpPodNumMap[pod.Status.HostIP]++
			} else {
				workerIpList = append(workerIpList, pod.Status.HostIP)
				IpPodNumMap[pod.Status.HostIP] = 1
			}
		}

		//  check or update backend server for haproxy policy
		for _, instanceId := range SearchInstanceIds {
			ModifyHaproxyConfigLocalMode(instanceId, config, workerIpList, IpPodNumMap)
		}

	} else {
		// ExternalTrafficPolicy == cluster and policy default weigh == 1
		workerIpList = client.Sa.GetWorkerNodeInternalIps(workerLabel)

		//  check or update backend server for haproxy policy
		for _, instanceId := range SearchInstanceIds {
			ModifyHaproxyConfigCLusterMode(instanceId, config, workerIpList)
		}
	}

	return nil
}

func UpdateNodePod() error {
	haproxyConfigs, err := GetCustomerHaproxyConfigs()
	if err != nil {
		log.Errorf("err: %s", err)
		return nil
	}

	for _, haConfig := range haproxyConfigs.Instances {
		err := CheckClusterIpNodeByHaConfig(haConfig, []string{})
		if err != nil {
			log.Errorf("err: %s", err)
			return nil
		}
	}
	return nil
}
