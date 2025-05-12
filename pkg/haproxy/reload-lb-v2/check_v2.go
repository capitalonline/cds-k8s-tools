package reload_lb_v2

import (
	"cds-k8s-tools/pkg/client"
	"cds-k8s-tools/pkg/haproxy/api"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"reflect"
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
	//InstanceIdPolicyMap   = new(sync.Map)
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
		serviceNameSpace := fmt.Sprintf("%s/%s", Config.NameSpace, Config.ServiceName)
		if verifyMap[serviceNameSpace] {
			return fmt.Errorf("service Name repeated in user haproxy-instance configmap")
		}
		verifyMap[serviceNameSpace] = true

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
		log.Infof("begin check haproxy instance by tag name %s", Config.LbTag)

		req := map[string]string{"TagName": Config.LbTag}
		Instances, err := api.DescribeHaproxyInstancesByTag(req)
		if err != nil {
			log.Errorf("DescribeHaproxyInstancesByTag failed: %s", err)
			return nil
		}

		if len(Instances) == 0 {
			log.Infof("not found haproxy instances by tag name %s", Config.LbTag)
			SvcNameInstanceIdsMap.Delete(Config.ServiceName)
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

		if len(SearchInstanceIds) == 0 {
			log.Infof("not found haproxy instances by tag name %s", config.LbTag)
			return nil
		}
		SvcNameInstanceIdsMap.Store(config.ServiceName, SearchInstanceIds)
	}

	Service := client.Sa.GetService(config.ServiceName, config.NameSpace)
	if Service == nil {
		return fmt.Errorf("not found service name %s in namespace %s", config.ServiceName, config.NameSpace)
	}

	if Service.Spec.ExternalTrafficPolicy == "Local" {
		// Count the number of Pods on each worker
		podsByServiceName := client.Sa.GetPodByServiceName(config.ServiceName, config.NameSpace)
		if len(podsByServiceName.Items) == 0 {
			return fmt.Errorf("not found pods by serviceName %s", config.ServiceName)
		}

		for _, pod := range podsByServiceName.Items {
			if pod.Status.HostIP != "" {
				if _, ok := IpPodNumMap[pod.Status.HostIP]; ok {
					IpPodNumMap[pod.Status.HostIP]++
				} else {
					workerIpList = append(workerIpList, pod.Status.HostIP)
					IpPodNumMap[pod.Status.HostIP] = 1
				}
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

	// verify customer params: tagName and ServiceName
	err = verifyUserConfigMap(haproxyConfigs)
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
