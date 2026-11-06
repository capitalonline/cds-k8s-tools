package podalarm

import (
	"cds-k8s-tools/pkg/config"
	"cds-k8s-tools/pkg/consts"
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

	// AutoDeleteEnabled 自动删除总开关，默认 no（dry-run，只打日志不调用删除接口）
	AutoDeleteEnabled = "pod.autodelete.enabled"
	// AutoDeleteDelayMinutes 在告警阈值基础上再等待的分钟数，超过 (告警阈值+该值) 才会真正处理
	AutoDeleteDelayMinutes = "pod.autodelete.delay.minutes"
	// AutoDeleteNamespaces 允许自动删除的命名空间，逗号分隔；必须显式指定，留空视为功能未生效
	AutoDeleteNamespaces = "pod.autodelete.namespaces"
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
		m = 20
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

// getAutoDeleteEnabled 自动删除总开关，未配置或配置非法值时一律视为关闭。
func getAutoDeleteEnabled() bool {
	return conf.GetKeyString(DefaultConfig, AutoDeleteEnabled) == consts.Yes
}

// getAutoDeleteDelayMinutes 告警阈值之后额外等待的分钟数。
// 实际自动删除阈值 = getTerminatingThresholdMinutes() + 该值，因此自动删除必然晚于告警。
func getAutoDeleteDelayMinutes() int {
	m := conf.GetKeyInt(DefaultConfig, AutoDeleteDelayMinutes)
	if m <= 0 {
		m = 5
	}
	return m
}

// getAutoDeleteNamespaces 允许自动删除的命名空间集合。
// 必须显式配置，留空返回空集合：调用方据此判断"功能未生效"，避免忘记配置就对全集群生效。
func getAutoDeleteNamespaces() map[string]bool {
	ns := conf.GetKeyString(DefaultConfig, AutoDeleteNamespaces)
	result := make(map[string]bool)
	if ns == "" {
		return result
	}
	for _, s := range strings.Split(ns, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			result[s] = true
		}
	}
	return result
}
