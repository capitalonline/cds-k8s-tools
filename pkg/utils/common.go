package utils

import (
	"cds-k8s-tools/pkg/oscmd"
	"fmt"
	"os"
)

const (
	defaultApiHost = "http://cdsapi.capitalonline.net"
	// defaultApiHostOversea  = "http://cdsapi-us.capitalonline.net"
	preApiHost = "http://cdsapi-gateway.gic.pre/openapi"
	// apiHostLiteral         = "CDS_API_HOST"
	accessKeyIdLiteral     = "CDS_ACCESS_KEY_ID"
	accessKeySecretLiteral = "CDS_ACCESS_KEY_SECRET"
	cckProductType         = "cck"
	version                = "2019-08-08"
	signatureVersion       = "1.0"
	signatureMethod        = "HMAC-SHA1"
	timeStampFormat        = "2006-01-02T15:04:05Z"
)

const (
	SendAlarm = "SendAlarm"
)

var (
	APIHost         string
	AccessKeyID     string
	AccessKeySecret string
)

func IsAccessKeySet() bool {
	return AccessKeyID != "" && AccessKeySecret != ""
}

func init() {
	//if APIHost == "" {
	//	APIHost = devApiHost
	//}
	if AccessKeyID == "" {
		AccessKeyID = os.Getenv(accessKeyIdLiteral)
	}
	if AccessKeySecret == "" {
		AccessKeySecret = os.Getenv(accessKeySecretLiteral)
	}

	if os.Getenv("DEPLOY_TYPE") == "pre" {
		APIHost = preApiHost
		sh := fmt.Sprintf("echo '101.251.217.74 %s' >> hosts", preApiHost)
		_, _ = oscmd.CmdToNode(sh)
	} else if os.Getenv("DEPLOY_TYPE") == "pro" {
		APIHost = defaultApiHost
	}
}
