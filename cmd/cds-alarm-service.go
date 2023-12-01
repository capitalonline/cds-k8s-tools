package main

import (
	"cds-k8s-tools/pkg/consts"
	"cds-k8s-tools/pkg/service"
	"cds-k8s-tools/pkg/utils"
)

func main() {
	utils.SetLogAttribute(consts.AlarmPodName)
	go service.AlarmCenter()
	service.Run()
}
