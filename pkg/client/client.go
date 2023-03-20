package client

import (
	"context"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetNode(name string) *v1.Node { return Sa.GetNode(name) }
func (sa *ServiceAccount) GetNode(name string) *v1.Node {
	nodeRef, err := sa.CoreV1().Nodes().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}
	return nodeRef
}
