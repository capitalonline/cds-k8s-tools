package main

import (
	"cds-k8s-tools/pkg/service"
	"cds-k8s-tools/pkg/utils"
)

func main() {
	utils.SetLogAttribute("cds-alarm-service")
	service.Run()
}
