package main

import (
	"context"
	"encoding/json"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var kubeConfig = `
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUM1ekNDQWMrZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFWTVJNd0VRWURWUVFERXdwcmRXSmwKY201bGRHVnpNQjRYRFRJek1URXhOREF5TXpFeU1Wb1hEVE16TVRFeE1UQXlNekV5TVZvd0ZURVRNQkVHQTFVRQpBeE1LYTNWaVpYSnVaWFJsY3pDQ0FTSXdEUVlKS29aSWh2Y05BUUVCQlFBRGdnRVBBRENDQVFvQ2dnRUJBTWxiCloxeUszaUJqQXhSa3JkTlNCa2Njc0dLNkFhNnRkeGxwSWczWGNGNFQvQ3VuQ25SeDlOTjkwQXZSc3p3RXZySlkKSWVKV1lYcndLQVVLTnJFN3hTbjlCa2pVbk9ZUTVjWFNWRWxNR2JCaEp6Tk9MQWJnTmg5aDBMaXpDNk92SXZBWQpNamlGdS9kMWlwYlV0c2VkTm9LM3dCZWRQdGpmWUd6b2trVm1EbHJFQ2RUREdmODhqUTk3LzIyVk55bjk2dU1hCkJ2elJtaGd0UE5XZUJuTDVZbGdJSld4YTVSdlE3RTBubjFpYjFFWWRPSWlKdXZFZURTMmVaMlJSeWJUVjh0Z08KY2xjOWx3QjFVcFJJREZselpZaGZYYUQyeXJhd1RuUGlEdlNXZFcwOTFFcTFKUWZPUC9CL0ozd0t0MkpPYkVFZQpuMDMvYlpER21ibzJhSU1ScHhrQ0F3RUFBYU5DTUVBd0RnWURWUjBQQVFIL0JBUURBZ0trTUE4R0ExVWRFd0VCCi93UUZNQU1CQWY4d0hRWURWUjBPQkJZRUZIWnlmeklPSGFVbnU5YlJVNFBVUTJFUnR0ZVNNQTBHQ1NxR1NJYjMKRFFFQkN3VUFBNElCQVFBRDlrdWNlU3dqVWFhVVlJc0lLb0lITDdOdmV6SitLT2pFa1Mwd01RcGJLZ2tvM1NkYgpBMjVTdWR0MFhvSTJQaTU0em1vSytXazdWL2JGMFpjT2hLTTJtUnEzNVF5ZXpZQ2wxM2hyYnhaRWQ5d3A5bW02CnpubG9iL3BJZGJTZ3JXYmpMT0h0dmpHUXNscTdpYW13bDFqNjhCVVY1eG4xZzExQkRna1NhZUVKVnltVVhRMVgKMUIzNVZiRDFaRU9zYVZCWjU4QnVNYjM2K3dDcjNNVlpOa2RsRUJlbFR0Rzl6bitzSWRSV2lIS2ZMSVUrM3lycgpHSjFtbTcza1RxaGE2TU1JZDBaTm1CUE9ZQlhCUDZqVW83MCs2ZndJMll0Q1F1bTlNSzJpWC80NG5DdS9hdGJ1CjJBSE9lclhYSlBEcjNnVW96U1ZxSnFDMENkSW55NnRQUk42RwotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
    server: https://uetr5y.yun-paas.com:6443
  name: kubernetes
contexts:
- context:
    cluster: kubernetes
    user: kubernetes-admin
  name: kubernetes-admin@kubernetes
current-context: kubernetes-admin@kubernetes
kind: Config
preferences: {}
users:
- name: kubernetes-admin
  user:
    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURFekNDQWZ1Z0F3SUJBZ0lJT3ZwRkF6VVRtQXN3RFFZSktvWklodmNOQVFFTEJRQXdGVEVUTUJFR0ExVUUKQXhNS2EzVmlaWEp1WlhSbGN6QWVGdzB5TXpFeE1UUXdNak14TWpGYUZ3MHlOREV4TVRNd01qTXhNalJhTURReApGekFWQmdOVkJBb1REbk41YzNSbGJUcHRZWE4wWlhKek1Sa3dGd1lEVlFRREV4QnJkV0psY201bGRHVnpMV0ZrCmJXbHVNSUlCSWpBTkJna3Foa2lHOXcwQkFRRUZBQU9DQVE4QU1JSUJDZ0tDQVFFQTFLNzgzYjFnRUdaNzJ1TngKdlFEb1lDUk9FOFJYMitWVFBQTWpKYXhjQ1RuUGlBdkx3UU9KT1lvRjg0cmp0TzlMN28vcWZ4V2lxSG9URjVLZgpUUkUyMXF3UlNocCt2L05sSTh0TFllbDAxbTFqS3RBSUZkZ0JoTzJ3Z1ZWR25lc0gxOXVSWmZzSkVXWmFUQjViCmlmWnhTSFZJQ3pIQ0VYSko0ZXRjQ3ZXbDAzMndpTFNuVVpuV3RQWWEyL3Z0Uk1obG1JbXNrRW5TMTc2a2RtVEEKaVlNb3QwSWZrQzN0SC9pbEdPcG1MSURBTWlCSTlOME1hODVKVEx5MDhiVGM3YXBiTHB4RnNlUzJZYlphd1MxRQptcW5SaFhLYjJKQ2x4MzRCQ1g3RmhtaGNza091cDJna01SaWEzeDY2akpGT3BsNzJ3QWpSa2lINHZHNEI1RDFnClhYUEFOd0lEQVFBQm8wZ3dSakFPQmdOVkhROEJBZjhFQkFNQ0JhQXdFd1lEVlIwbEJBd3dDZ1lJS3dZQkJRVUgKQXdJd0h3WURWUjBqQkJnd0ZvQVVkbkovTWc0ZHBTZTcxdEZUZzlSRFlSRzIxNUl3RFFZSktvWklodmNOQVFFTApCUUFEZ2dFQkFNRUhUZSsybE1rK2pRV2xZc1I4VWVZUHoxcTIzendhU0dUdS9mWG92a2hpVHZnYmtDV3FrZVpDClJZVGhlQzNZbWlCNmRFdkF2S1FGbmFoMld6SDkvMnZ0MkxTZ2xoOWxlTWV0ZFE2MlpiYXkyNTQzRDJ0L0Q3VmQKb3Y1cUtBbFFPa1JDcUJOSm1yd0JzbGRhcVc4SVA4bTh1bUxkM1N2TGZjVFozOEFXVXBUdnRGZXJlemR5YUsweQorTUh0a1VzMDFycGV0ckN0OWc1UnNsRG13L0pQa3A3RFZyYk9QdnZWc1JUZVRzc1pkdnJnU1RPYXZBYkNtZmdOCmNlanFVSTg2a2NSRExUS3lhemRZYUhGU2hjUUNpcUtmMkZSbUVIdXY5Mkd5QU4wTCtCQ2ZkMldPSmo2bldBWVcKN3JTYnZzbCtrWjlsZ0I2bmhRei9oTHR6am5uNVlXST0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
    client-key-data: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb3dJQkFBS0NBUUVBMUs3ODNiMWdFR1o3MnVOeHZRRG9ZQ1JPRThSWDIrVlRQUE1qSmF4Y0NUblBpQXZMCndRT0pPWW9GODRyanRPOUw3by9xZnhXaXFIb1RGNUtmVFJFMjFxd1JTaHArdi9ObEk4dExZZWwwMW0xakt0QUkKRmRnQmhPMndnVlZHbmVzSDE5dVJaZnNKRVdaYVRCNWJpZlp4U0hWSUN6SENFWEpKNGV0Y0N2V2wwMzJ3aUxTbgpVWm5XdFBZYTIvdnRSTWhsbUltc2tFblMxNzZrZG1UQWlZTW90MElma0MzdEgvaWxHT3BtTElEQU1pQkk5TjBNCmE4NUpUTHkwOGJUYzdhcGJMcHhGc2VTMlliWmF3UzFFbXFuUmhYS2IySkNseDM0QkNYN0ZobWhjc2tPdXAyZ2sKTVJpYTN4NjZqSkZPcGw3MndBalJraUg0dkc0QjVEMWdYWFBBTndJREFRQUJBb0lCQUFNMXBORS8wMXhMLzZCOQpBREZtK3hyQVVZRlEzRGFRVG1KRUZRcXpnQ3dQZUVkSVRrbmFZTTdDZTNhQ2FROUk4UzluY3BWNllSc0R4SmY5CmVYUHpSNDJUeVNzQ0hWbU9OYitGaU55d1pqZjZMSjN2eDc5MHBBazZnUWhpbmc4eUJjdEhIL21YQVRzRy9XUGEKcld6MmtCMFUwQUtEUExlQXJ3YSt3NFBTMHk3TzFySWltMGRHT0F3Y1Eza2Y1QXdpRHZ6cXpocVc4YmFzS0JWegpJTTRGZWZPaGliL1NuaTdrVmwzb1dBNUxwazgvVDlzemZYNmxwNHlCRUIxODZzSk9OUXl1YTN2VGtOVHFZcnRGCjFVaEwwRGJRS1JldHF3U0dWK3B4akJSZzZKOEV6Q21BSytOM0ZpYWFseVpwczZXNTlMb2tTK2xpTUU2NmhnQksKMTNna2VnRUNnWUVBL3dTbTR3N0tqSmhUczNLM1NUbi82WUlkYW94b0Rld1c4ZXU4Ym9URU9XNGlDS0tpM3JhQQpzT3hiOXZsYzRPZndsdktmQ0JyUVpQb0pHMU9nZ2pYa1NPQzgyRmUzQ0E0OUk5OWRRVW9Cd0J4amxXdkd0M3NvCllMVWR2UVQzRXVTWXRCZjJlMTh1SVNEVEpEMEc2OERZQ2hnYVh2S00wYWpLN3k1M01Ua1Rzd0VDZ1lFQTFZQ2MKU05KWFhDVmF6NkJPOHo1RUxyTFIrcnJmNFBtcW01c3JPek9HUVEyNCtFcGYwOGZRSjF6RzMxKzg5QVkxR3JXSQpwTVFqNUVnd2RvZlBiK0xBU1BNbm1sUDBDK2Q2VjFPUEtFdjQ5SVFrQkNNVVRIZjhlQUVyVVFDS0kwRUJZdzYxCjVQZDBpbVBFQTgxZzlJUk5tZzd5VVE1WDRzTXQ0cHkwNXkvSFN6Y0NnWUVBNDBrWUhJSlFVQ0pyWnlJMDdSUysKV3pYV1ZlSXgwMGE1NUgrLy81aGc4dmFQYXJiWkJqb09WS0UwRGRpTnlQMnZWam1ETjh2K05DRU5BTWYxNUZkMwowT1JNSzExeUNjSDNDQVBKcjZqd0NuTEM1cWVhQW1uSHdQbHJPYzQxRHllaVdkQ0pvOGRlNjdPL3V5cVJBb2xyCmd3T2NiVWNyN3FqTHhZVGFRb3FtWGdFQ2dZQWF5Q2hTcGpnWk1nSmpPeXZNTFlwbUJUNTc0a2RGTkd4bldwNmcKcllUdzBpVGEySkdPd21qbEZ5bEhTZjRzNmo1dEhFcUl5S1hyOC9aSVdCNzRYUXhiMmt5a2VsV0p6TDYzQjU5VQpvYnNZQ1I5dmVXc0pjSisxK2dlU0FLeFRZY3NudnVlb1VqWkhTZDZEejVhUzlhbTZZcGVZL1dDZTdIYnNEMVpPCkRkdEZId0tCZ0JoZUM2Q2VqbVFSK2d5VnZrZTlBTUR4UFo5MlFtZllKUkN3R21oVWFTS2czaHZHNm44YUl1WUcKVmgvYkxDanFxT0pDQ2F1b3ZIa0FwOWhqQkVXSnVnODJKN3c4d3IwbEJ2RFJvRllyc2M0OGFzRGZGVmt5QWhTUwppTHRiakNCdk8xYWRPT2c5MmlnMG1PSDZDRHMwNDV2aXNoU3IwbXd5YmJ4MVF6N2hXQ2ZGCi0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==`

func KubeClient(kubeCfg string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeCfg))
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func GetPodByServiceName(serviceName, namespace string) *v1.PodList {
	cli, err := KubeClient(kubeConfig)
	if err != nil {
		fmt.Println(err)
	}

	service, err := cli.CoreV1().Services(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}
	// 获取Service的选择器标签
	selector := service.Spec.Selector
	podListOptions := metav1.ListOptions{
		LabelSelector: labels.Set(selector).String(),
	}

	podList, err := cli.CoreV1().Pods(namespace).List(context.TODO(), podListOptions)
	if err != nil {
		panic(err)
	}
	return podList
}

func GetAllWorkerNodeByLabel(name string) (*v1.NodeList, error) {

	cli, err := KubeClient(kubeConfig)
	if err != nil {
		fmt.Println(err)
	}

	options := metav1.ListOptions{
		LabelSelector: name,
	}
	nodeList, err := cli.CoreV1().Nodes().List(context.TODO(), options)
	if err != nil {
		panic(err)
	}
	return nodeList, nil
}

func GetConfigMapByName(name, namespace string) (*v1.ConfigMap, error) {
	cli, err := KubeClient(kubeConfig)
	if err != nil {
		fmt.Println(err)
	}

	configMap, err := cli.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return configMap, nil
}

type HaproxyConfigs struct {
	Instances []HaproxyConfig `json:"lb2port"`
}

type HaproxyConfig struct {
	LbTag        string `json:"lb_tag"`
	NameSpace    string `json:"namespace"`
	ServiceName  string `json:"service_name"`
	MaxConn      int    `json:"max_conn"`
	NodePort     int    `json:"node_port"`
	ListenerPort int    `json:"listener_port"`
}

func main1() {
	configMap, err := GetConfigMapByName("lb2pod", "kube-system")
	if err != nil {
		return
	}

	haproxyInstances := HaproxyConfigs{}

	for key, val := range configMap.Data {
		if key == "haproxy_instances" {
			if err := json.Unmarshal([]byte(val), &haproxyInstances); err == nil {
				fmt.Printf("Instance: %+v\n", haproxyInstances.Instances)
			} else {
				fmt.Println(err)
			}
		}
	}
}

func main2() {
	nodeList, err := GetAllWorkerNodeByLabel("node-role.kubernetes.io/compute")
	if err != nil {
		return
	}
	fmt.Printf("nodeList: %d", len(nodeList.Items))

	//var ipList []string
	for _, node := range nodeList.Items {
		for _, address := range node.Status.Addresses {
			if address.Type == v1.NodeInternalIP {
				fmt.Printf("Node: %s, IP: %s\n", node.Name, address.Address)
			}
		}
	}
}

func main() {
	podList := GetPodByServiceName("nginx-svc", "test-reload-lb")
	fmt.Printf("podList: %d", len(podList.Items))
	for _, node := range podList.Items {
		fmt.Println(node.Status.HostIP)
	}
}
