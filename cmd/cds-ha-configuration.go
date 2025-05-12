package main

import (
	reloadlbv1 "cds-k8s-tools/pkg/haproxy/reload-lb-v1"
	"cds-k8s-tools/pkg/utils"
)

func main() {
	utils.SetLogAttribute("cds-ha-configuration")
	reloadlbv1.Run()
}
