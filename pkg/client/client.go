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

func (sa *ServiceAccount) GetWorkerNodeInternalIps(label string) (ips []string) {
	options := metav1.ListOptions{
		LabelSelector: label,
	}

	nodeList, err := sa.CoreV1().Nodes().List(context.TODO(), options)
	if err != nil {
		log.Errorf("err: %s", err)
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

func (sa *ServiceAccount) GetService(serviceName, nameSpace string) *v1.Service {
	log.Infof("service name: %s, namespace: %s", serviceName, nameSpace)
	service, err := sa.CoreV1().Services(nameSpace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("err: %s", err)
		return nil
	}
	return service
}

func (sa *ServiceAccount) GetPodByService(service *v1.Service) *v1.PodList {
	selector := service.Spec.Selector
	podListOptions := metav1.ListOptions{
		LabelSelector: labels.Set(selector).String(),
	}
	podList, err := sa.CoreV1().Pods(service.ObjectMeta.Namespace).List(context.TODO(), podListOptions)
	if err != nil {
		log.Errorf("err: %s", err)
		return nil
	}
	return podList
}

func (sa *ServiceAccount) GetPodByServiceName(serviceName, nameSpace string) *v1.PodList {

	service, err := sa.CoreV1().Services(nameSpace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("err: %s", err)
		return nil
	}

	// Service Selector
	selector := service.Spec.Selector

	podListOptions := metav1.ListOptions{
		LabelSelector: labels.Set(selector).String(),
	}

	podList, err := sa.CoreV1().Pods(nameSpace).List(context.TODO(), podListOptions)
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
