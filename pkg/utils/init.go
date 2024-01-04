package utils

import (
	"cds-k8s-tools/pkg/consts"
	"cds-k8s-tools/pkg/oscmd"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
)

var (
	APIHost         string
	AccessKeyID     string
	AccessKeySecret string
)

const (
	DescribeHaInstance           = "DescribeLoadBalancers"
	DescribeLoadBalancerStrategy = "DescribeLoadBalancerStrategys"
	ModifyLoadBalancerStrategy   = "ModifyLoadBalancerStrategys"
)

func IsAccessKeySet() bool {
	return AccessKeyID != "" && AccessKeySecret != ""
}

func init() {
	if AccessKeyID == "" {
		AccessKeyID = os.Getenv(consts.CDS_ACCESS_KEY_ID)
	}
	if AccessKeySecret == "" {
		AccessKeySecret = os.Getenv(consts.CDS_ACCESS_KEY_SECRET)
	}
	switch os.Getenv(consts.DEPLOY_TYPE) {
	case consts.PRE:
		APIHost = consts.PreOpenApiHost
		sh := fmt.Sprintf("echo '%s  cdsapi-gateway.gic.pre' >> /etc/hosts", os.Getenv(consts.OPENAPI_IP))
		_, err := oscmd.Run("sh", "-c", sh)
		if err != nil {
			log.Errorf("cmd [%s] fail: %v", sh, err)
		}
	case consts.PRO, consts.TEST, consts.DEV:
		APIHost = consts.DefaultApiHost
	default:
		log.Errorf("unknown evn %v=%v", consts.DEPLOY_TYPE, os.Getenv(consts.DEPLOY_TYPE))
		APIHost = consts.DefaultApiHost
	}
}
