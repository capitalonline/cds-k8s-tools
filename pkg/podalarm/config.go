package podalarm

import (
	"cds-k8s-tools/pkg/config"
	"strings"
)

const (
	ConfigFileName              = "pod-alarm-config"
	ConfigDir                   = "/pod-alarm/"
	ConfigFmtType               = "ini"
	DefaultConfig               = "default."
	CheckStep                   = "pod.check.step"
	MonitorNamespaces           = "pod.monitor.namespaces"
	PendingThresholdMinutes     = "pod.pending.threshold.minutes"
	TerminatingThresholdMinutes = "pod.terminating.threshold.minutes"
)

var conf *config.Conf

func init() {
	conf = config.NewConf(ConfigFileName, ConfigFmtType, true, ConfigDir)
}

func getCheckStep() int {
	step := conf.GetKeyInt(DefaultConfig, CheckStep)
	if step <= 0 {
		step = 120
	}
	return step
}

func getNamespaces() []string {
	ns := conf.GetKeyString(DefaultConfig, MonitorNamespaces)
	if ns == "" {
		return nil
	}
	result := make([]string, 0)
	for _, s := range strings.Split(ns, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

func getPendingThresholdMinutes() int {
	m := conf.GetKeyInt(DefaultConfig, PendingThresholdMinutes)
	if m <= 0 {
		m = 15
	}
	return m
}

func getTerminatingThresholdMinutes() int {
	m := conf.GetKeyInt(DefaultConfig, TerminatingThresholdMinutes)
	if m <= 0 {
		m = 2
	}
	return m
}
