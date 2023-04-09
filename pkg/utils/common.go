package utils

import (
	"cds-k8s-tools/pkg/oscmd"
	"fmt"
	log "github.com/sirupsen/logrus"
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

	dnsDeal()

	if os.Getenv("DEPLOY_TYPE") == "pre" {
		APIHost = preApiHost
		_, err := oscmd.Run("sh", "-c", "echo '127.0.0.1  cdsapi-gateway.gic.pre' >> /etc/hosts")
		if err != nil {
			log.Warnf("set /etc/hosts err: %v", err)
		}
	} else if os.Getenv("DEPLOY_TYPE") == "pro" {
		APIHost = defaultApiHost
	} else if os.Getenv("DEPLOY_TYPE") == "" {
		APIHost = defaultApiHost
	}
}

func dnsDeal() {
	dns := "nameserver 8.8.8.8"
	oversea := os.Getenv("CDS_OVERSEA")
	if oversea != "True" {
		dns = "nameserver 114.114.114.114"
	}
	_, err := oscmd.Run("sh", "-c", "cp /etc/resolv.conf /etc/resolv.conf.bak")
	if err != nil {
		log.Warnf("cp /etc/resolv.conf /etc/resolv.conf.bak err: %v", err)
		return
	}

	sh := fmt.Sprintf("sed '1i\\%s' /etc/resolv.conf.bak > /etc/resolv.conf", dns)
	_, err = oscmd.Run("sh", "-c", sh)
	if err != nil {
		log.Warnf("add nameserver err: %v", err)
		return
	}

}
