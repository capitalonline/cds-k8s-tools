package main

import (
	"cds-k8s-tools/pkg/snat"
	"cds-k8s-tools/pkg/utils"
)

func main() {
	utils.SetLogAttribute("cds-snat-configuration")
	snat.Run()
}
