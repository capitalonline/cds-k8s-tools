package podalarm

import (
	"cds-k8s-tools/pkg/config"
	"cds-k8s-tools/pkg/consts"
	"strconv"
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
// 实际自动删除阈值 = getTerminatingThresholdMinutes() + 该值。
// 显式配置（含 0）按配置生效；只有 key 未配置/为空/非数字/负数时才回落到默认 5。
// 0 表示"不做额外延迟"，即删除阈值 = terminating 告警阈值。
func getAutoDeleteDelayMinutes() int {
	raw := strings.TrimSpace(conf.GetKeyString(DefaultConfig, AutoDeleteDelayMinutes))
	if raw == "" {
		return 5
	}
	m, err := strconv.Atoi(raw)
	if err != nil || m < 0 {
		return 5
	}
	return m
}
