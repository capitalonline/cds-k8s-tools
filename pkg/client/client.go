package client

import (
	"context"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func GetNode(name string) *v1.Node { return Sa.GetNode(name) }

func (sa *ServiceAccount) GetNode(name string) *v1.Node {
	nodeRef, err := sa.CoreV1().Nodes().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		log.Errorf("err: %s", err)
		return nil
	}
	return nodeRef
}

func (sa *ServiceAccount) GetAllWorkerNode(label string) *v1.NodeList {
	options := metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/compute",
	}
	nodeList, err := sa.CoreV1().Nodes().List(context.TODO(), options)
	if err != nil {
		log.Errorf("err: %s", err)
		return nil
	}
	return nodeList
}

func (sa *ServiceAccount) GetPodByServiceName(serviceName, namespace string) *v1.PodList {

	// get service
	service, err := sa.CoreV1().Services(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("err: %s", err)
		return nil
	}
	// 获取Service的选择器标签
	selector := service.Spec.Selector

	// 构建Pod查询选项
	podListOptions := metav1.ListOptions{
		LabelSelector: labels.Set(selector).String(),
	}

	// 查询匹配的Pod列表
	podList, err := sa.CoreV1().Pods(namespace).List(context.TODO(), podListOptions)
	if err != nil {
		log.Errorf("err: %s", err)
		return nil
	}

	return podList
}

func (sa *ServiceAccount) GetConfigMapByName(name, namespace string) *v1.ConfigMap {
	configMap, err := sa.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		log.Errorf("err: %s", err)
		return nil
	}
	return configMap
}
