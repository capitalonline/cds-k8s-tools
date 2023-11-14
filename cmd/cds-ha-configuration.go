package main

import (
	"cds-k8s-tools/pkg/haproxy"
	"cds-k8s-tools/pkg/utils"
)

func main() {
	utils.SetLogAttribute("cds-ha-configuration")
	haproxy.Run()
}
