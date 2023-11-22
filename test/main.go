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

var kubeConfig = `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUM1ekNDQWMrZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFWTVJNd0VRWURWUVFERXdwcmRXSmwKY201bGRHVnpNQjRYRFRJek1URXhOekE0TVRjek5Wb1hEVE16TVRFeE5EQTRNVGN6TlZvd0ZURVRNQkVHQTFVRQpBeE1LYTNWaVpYSnVaWFJsY3pDQ0FTSXdEUVlKS29aSWh2Y05BUUVCQlFBRGdnRVBBRENDQVFvQ2dnRUJBTFVKClFGTGE0Qng2ZjQxUDhsem95WmRMcEhLSzdIcEpSOW10dHFoMlgrelYwU1A1R2xwMm5LM3o5YzBhSUdsVTRiQ0gKUmVpM0FoSGt0bmVRNndFbUNsZHFQU3dGeTZFdmxobXozNmlWTnFqNmYzTzhTTUxTV3pLaFU0aXlKdGo2bjZRZAprc2pMM2p6NkRCWjBtWWlPK3VHTlB4R2xKa1h0Z0thZmdTelp4Y3FWcjR6MktiL3JiYmdkWGpuYUtXMWlpbG96CnZBNjY4eVVQN0I5L2FENWZ0MnpxVm13NkV3ZndPaXhnRWNFQi9hRGRFcEoycHlDam90c1Nhb0F2VjFkR3ZJbysKek1LbjV3Y0V6dGQ4c2dnK1c5bC9DM1ZiTEJ5S2tlU0Q2UWhBYnU3VW5jOHhNa1g2b3U2MWx0T25TQXR4ZzJJRgpGTzhyNlliR2l6bWNuazRiYXdVQ0F3RUFBYU5DTUVBd0RnWURWUjBQQVFIL0JBUURBZ0trTUE4R0ExVWRFd0VCCi93UUZNQU1CQWY4d0hRWURWUjBPQkJZRUZOYno3cVQ2MktXb05namxIUzZCQks2UU1NY2RNQTBHQ1NxR1NJYjMKRFFFQkN3VUFBNElCQVFBRGdRVzhvN1JBQUxLVmh5VDZXVDVOQktSYmtyay9DcnhzT1htQVJBbm44dEtERG1VTgpENER5bnUyWjJmQjRsTm4wTWx5SlZYNUxnaGlhQUxYcTYzM0krSEFZc2g5M3lCbVpGcmt4T254THQ1OWRTSTFhCkx3MkhaRDZtaExWWXhMTHN3S1RXZnBoNWo0eFBIaGVBRUN2cEZ6Q2RwNUxPZzh2cmxsYnNieGVSUTl4b1pXK3cKY0paWjlGVktLaWpmcHJocG5CY0dIZ1YxYmxoQUY2L1FzUGRIN2w3MkhqU09VVmNlTWEzZmhDY1VKRDFBV2ZEZQpaYkpsRW1oSDlDT2ZZN2E1U2VhREhXRUJqYW1OQVFQYVpxai9VMWZjZjZCMXc5cy8va1VuMDRxczNSZUEvR1VHCjVNVzUxMzVIN0dNOVRoUGZrNVd4bXE0NjU4aGJwWHo4VTRJVAotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
    server: https://2h8kpb.yun-paas.com:6443
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
    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURFekNDQWZ1Z0F3SUJBZ0lJTFplY2Q2ckl4RVF3RFFZSktvWklodmNOQVFFTEJRQXdGVEVUTUJFR0ExVUUKQXhNS2EzVmlaWEp1WlhSbGN6QWVGdzB5TXpFeE1UY3dPREUzTXpWYUZ3MHlOREV4TVRZd09ERTNNemRhTURReApGekFWQmdOVkJBb1REbk41YzNSbGJUcHRZWE4wWlhKek1Sa3dGd1lEVlFRREV4QnJkV0psY201bGRHVnpMV0ZrCmJXbHVNSUlCSWpBTkJna3Foa2lHOXcwQkFRRUZBQU9DQVE4QU1JSUJDZ0tDQVFFQXpGZWFaRkhwbUFtNGs2a0gKM1BRTk5pNDFTdmVTSUdHWG81ZGsrUEg5ZGJSWVJCYW1DMzFQbWJRS1RhVUR3RWhqTTNkZGRjVVB5VTdGOTRGbgpZdncrZFFRVFc5cFg3NC9OQ25ZT0lOemZBS2NESGRKSDFhZXpzQkE5T2RRcXFtT3Vxd1J4WjVMaFRFa1A5V3VoCjJ4UjBEU0g2OWEvWTViNnV4ODZLem8vdXpxZ3RrOVRIZFM4UjU2WGJwRGR5Q1ZUV3A5TlhnQzF5eFlYOUtPVVYKTG9VSC9oOUFYWDEybG5YaVJnckVYUFNUQkVtaU12ZldGZW5FUzJyaXc1bzhpdjNKbFNLeGw3b3pRNm81VkNiQQo2WmtpSURaUWdBUklvaXBieElLRTVlY3NpdURNdlc5WEJTWUx2MCtkcktMaTdZajcwT3Q4NmkzWjNNb2lzU0dwCnRZa1NPUUlEQVFBQm8wZ3dSakFPQmdOVkhROEJBZjhFQkFNQ0JhQXdFd1lEVlIwbEJBd3dDZ1lJS3dZQkJRVUgKQXdJd0h3WURWUjBqQkJnd0ZvQVUxdlB1cFByWXBhZzJDT1VkTG9FRXJwQXd4eDB3RFFZSktvWklodmNOQVFFTApCUUFEZ2dFQkFCdEdCM0g4MDJUVFBmcnZjczlsaUdjTUF4a0Y4aGhqUENYQ01pNWs2anhuMnB6ZktvUDFrWTFRCjcxb1pDSThnOVVlNks2cmJoa285MWQyaTRRN2YxTDNSTGlVZ3pHaHFiVkdhQmdVV0NFbmlSMTVFNHBTeFMwdHYKTmJhWks5WnZ4N2EyTVd0T25TZXZXY3ZES2N3RDYwakMrZk9kL2lMbkphdlZ0R054ZnltQlE2Kys2b0dnRzJHMwpVdWcrUy9yL08xK0ZxTE1ySFo3cHA1bXF4TjI0dGdWc2NQclBQSmRtYnNlcW1wM29aYmRnSXRyUk5rVXZjS2twClZEQ3lHY2srUTBRZWRBZFlaSXRabDdydU92L3BEY3QxNXpBWUJlbEtkTEZzQkFWODlOTGhab2RBRi9vcG1TUU4KUUJiVG4vTjJsRTdtN3BDZnpTb2t6ZktkU09veEJYaz0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
    client-key-data: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb3dJQkFBS0NBUUVBekZlYVpGSHBtQW00azZrSDNQUU5OaTQxU3ZlU0lHR1hvNWRrK1BIOWRiUllSQmFtCkMzMVBtYlFLVGFVRHdFaGpNM2RkZGNVUHlVN0Y5NEZuWXZ3K2RRUVRXOXBYNzQvTkNuWU9JTnpmQUtjREhkSkgKMWFlenNCQTlPZFFxcW1PdXF3UnhaNUxoVEVrUDlXdWgyeFIwRFNINjlhL1k1YjZ1eDg2S3pvL3V6cWd0azlUSApkUzhSNTZYYnBEZHlDVlRXcDlOWGdDMXl4WVg5S09VVkxvVUgvaDlBWFgxMmxuWGlSZ3JFWFBTVEJFbWlNdmZXCkZlbkVTMnJpdzVvOGl2M0psU0t4bDdvelE2bzVWQ2JBNlpraUlEWlFnQVJJb2lwYnhJS0U1ZWNzaXVETXZXOVgKQlNZTHYwK2RyS0xpN1lqNzBPdDg2aTNaM01vaXNTR3B0WWtTT1FJREFRQUJBb0lCQUJrb3JBeXYvU2ZJQXA3RAprQUZIVngwVm9XQWlqUzVKZGNjaGk4QU80MXNMb2xaM3gyZmd2TjA2eW8zMnhEaDNjU2RVQ2dESEM0T0luRjAxCjVJbk9iczR2ZTBheTRtTFBmTHBPQUwxUkZHL2JJRW9hcXRlR2QxdzFFNlM4RjZpMDd6dUZKNFZPRTBrMk1hM1EKMjdQQ2wrdEtCTUVkTG9KUzhPZ082UTQzc0hwbkNDWlY5aWltSzZGOGJWQTJSamFUSmJGMVdsUndDM1A2aWUySQp6SUoyc202UUhBMi8rRitXWWZNdk5xc0lVa283TXFndWNKbHplYWwyUkJJZDVCblM0dXdWRTFzMWlTVm5GVG01CjFJOFJ4cllKd3RVdE15NDRaUU0xYkwxZm8rbnhUVERXOUJPbnZNV1c1bFBWYllaRlFUcXFyR0RZNENYWFZnM08KV0VFRGtuVUNnWUVBNnVBSUZRMmYxdWVrb1dlRHgxRW1MM0E5TnJYMnhIdk1TNUlUaWVJcjJQTjJkcUtyZk4xYwowL1pNenp3aXBTNm9qODlYVEk5dDd2TnFJSHRBaGd5cVhjbkkyQ0FtNldkZkRkSzJJbXZpdE8zU3BZMVhFK1NmCkwrY2liZkY1ekc3Mm1qdlExV1lRS29hN3kyT1E2RzdWNzRONDBkUUNqanhGV1BnVlRjampXZXNDZ1lFQTNyaU8KRmNpcDF3cUpQcWlWbytjd2MvM1h6dkZKNFBSSldvZVpPbnVkTlZqUTN0MHQyZUpCTkZWYU51dTkvcUhGMktLWgowWFBUaFgyUnUwSkt1eXFSbE1FS08rL0tPakNOTkV6RnUxMS83WGp2QzdjZ0MwcmR4NGJnRzI5Q2VPYUxWTXhNClpySzY0QUZKN3ZGYXgzVHZSOUxjUUpDUHRhSS9rSHZtSUgxVE4yc0NnWUF3UFJycjJBU0FDc3RSS3dWeHBrVUYKY1RQaFRMWUYzTGMwdmlldEpmcURjRjFnT0VDb1FINlVPZjNFZ2tGTFU2M2krMTZlcHNhWlVQejI5dGxscnF3KwozdmFWRE9WeEFuNFBSTHVMamtUZGpBcTdYYkFJc2VmUDJ0VERaOWp3RjhvbUd5cms2VFZneHBORFRvdXdjVE1YCkloVnFZdlN6YWNXRVpFOWJ6bXFEU1FLQmdFb1VESDVHWGVjK2crT3BZd3cvQ3lpcFY2eG5LUEYvang1alY1M1MKRzdud3JwaFI3THc1dXdKVEdVeUhJSXllOWhWV0Q5OVFyUndMWmZ0bzB6NXByRDVUN3JsOHlrQ01nWXJSdGpyWgpvSUUxNWh4NWJsa1RMNno3dVhLbWtPOXhqd3BIWVdvUExJVHhLTXdtenREa25lbS93cTVlNXMyOUIzTmhJbXZRCkEydTlBb0dCQUliUXZXU3kyVWVxSWl0Z010UUlWWHVjYlpieVdMOEhPaHdHcmEzTnhIdVFMd0ZIWkZYZmRlNEcKSmNCbVhMOXZlTXVuVG4vK1VaR091NFVQRlZzaGF2ZFZVMGJUSG9XMFN4WHNKMjVnUWZVZGlrdEpXRWprU011NAorb1FjdEFsMFdaODRQU1lFNWNlLys4YmJmQU1HWUY1dzVHM0J1MjJEMk1FQ2U5bE5uamlkCi0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==`

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
	fmt.Println(service)
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

func GetAllWorkerNodeByLabel(name string) (ips []string) {

	cli, err := KubeClient(kubeConfig)
	if err != nil {
		fmt.Println(err)
	}

	options := metav1.ListOptions{
		LabelSelector: name,
	}

	nodeList, err := cli.CoreV1().Nodes().List(context.TODO(), options)
	if err != nil {
		return nil
	}

	for _, node := range nodeList.Items {
		for _, address := range node.Status.Addresses {
			if address.Type == v1.NodeInternalIP {
				ips = append(ips, address.Address)
			}
		}
	}
	return
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

func GetService(serviceName, nameSpace string) *v1.Service {
	cli, err := KubeClient(kubeConfig)
	if err != nil {
		fmt.Println(err)
	}
	service, err := cli.CoreV1().Services(nameSpace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if err != nil {
		return nil
	}
	return service
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
	ListenerPort []int  `json:"listener_ports"`
}

func main2() {
	// generate configmap from json file: kubectl create configmap my-cm  --from-file=haproxy_instances --namespace=kube-system
	configMap, err := GetConfigMapByName("my-cm", "kube-system")
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

func main1() {
	ips := GetService("game1-svc", "ads")

	fmt.Printf("ips: %+v", ips)

}

func main() {
	podList := GetPodByServiceName("game1-svc", "ads")
	fmt.Printf("podList: %d", len(podList.Items))
	for _, node := range podList.Items {
		fmt.Println(node.Status.HostIP)
	}
}
