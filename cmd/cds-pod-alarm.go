package main

import (
	"cds-k8s-tools/pkg/consts"
	"cds-k8s-tools/pkg/podalarm"
	"cds-k8s-tools/pkg/utils"
)

func main() {
	utils.SetLogAttribute(consts.PodAlarmPodName)
	podalarm.Run()
}
