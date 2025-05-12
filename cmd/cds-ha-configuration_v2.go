package main

import (
	reloadlbv2 "cds-k8s-tools/pkg/haproxy/reload-lb-v2"
	"cds-k8s-tools/pkg/utils"
)

func main() {
	utils.SetLogAttribute("cds-ha-configuration")
	reloadlbv2.Run()
}
