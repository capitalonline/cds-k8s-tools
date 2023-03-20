package client

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	"time"
)

const (
	Host               = "KUBERNETES_SERVICE_HOST"
	Port               = "KUBERNETES_SERVICE_PORT_HTTPS"
	ServiceAccountPath = "/var/run/secrets/kubernetes.io/serviceaccount"
)

type ServiceAccount struct {
	Host  string
	Port  string
	Ca    []byte
	Token string
	*kubernetes.Clientset
}

var Sa *ServiceAccount

func init() {
	Sa = NewServiceAccount()
}

func NewServiceAccount() *ServiceAccount {
	host := os.Getenv(Host)
	portHttps := os.Getenv(Port)
	tokenFile := fmt.Sprintf("%s/token", ServiceAccountPath)
	caFile := fmt.Sprintf("%s/ca.crt", ServiceAccountPath)

	if host == "" || portHttps == "" {
		panic(fmt.Errorf("env args host:(%s) port:(%s) for serviceaccount blanked", host, portHttps))
	}
	sa := new(ServiceAccount)
	sa.Host = host
	sa.Port = portHttps
	if content, err := os.ReadFile(tokenFile); err != nil {
		panic(err)
	} else {
		sa.Token = string(content)
	}
	if content, err := os.ReadFile(caFile); err != nil {
		panic(err)
	} else {
		sa.Ca = content
	}

	tlsClientConfig := rest.TLSClientConfig{
		CAData: sa.Ca,
	}

	config := rest.Config{
		Host:            fmt.Sprintf("https://%s:%s", sa.Host, sa.Port),
		BearerToken:     sa.Token,
		TLSClientConfig: tlsClientConfig,
		Timeout:         20 * time.Second,
	}
	if clientSet, err := kubernetes.NewForConfig(&config); err != nil {
		panic(err)
	} else {
		sa.Clientset = clientSet
	}
	return sa
}
