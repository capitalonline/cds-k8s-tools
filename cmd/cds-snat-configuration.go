package main

import (
	"cds-k8s-tools/pkg/snat"
	"cds-k8s-tools/pkg/utils"
	"os"
)

const (
	LogType = "LOG_TYPE"
)

func main() {
	// set log config
	logType := os.Getenv(LogType)
	utils.SetLogAttribute(logType, "cds-snat-configuration")
	snat.Run()
}
