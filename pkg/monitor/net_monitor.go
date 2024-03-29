package monitor

import (
	"cds-k8s-tools/pkg/consts"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

type NetAlarmInfo struct {
	Metric string
	Ok     bool
	Value  string
	Addr   string
	Msg    string
}

type BaseMonitorConfig struct {
	CheckSum     int
	CheckLimit   int
	CheckStep    int
	CheckTimeout int
	RecoverSum   int
}

type NetMonitor struct {
	BaseMonitorConfig
	MonitorName string

	DefaultCheckMetric string
	AddrExt            []string
	AddrExclude        []string

	Alarming      bool
	AlarmInfoChan chan NetAlarmInfo

	CheckFunc        func(string, string, BaseMonitorConfig, *NetMonitor) NetAlarmInfo
	DefaultCheckAddr func() []string
	ReviewFunc       func(NetAlarmInfo, *NetMonitor)
}

func (m *NetMonitor) ResetCommon(cfg BaseMonitorConfig) {
	if cfg.CheckStep >= 30 {
		m.CheckStep = cfg.CheckStep
		if m.CheckStep > 3600 {
			m.CheckStep = 3600
		}
	}
	if cfg.CheckSum >= 3 {
		m.CheckSum = cfg.CheckSum
		if m.CheckSum > 30 {
			m.CheckSum = 30
		}
	}
	if cfg.CheckLimit >= 1 {
		m.CheckLimit = cfg.CheckLimit
		if m.CheckLimit > 30 {
			m.CheckLimit = 30
		}
		if m.CheckLimit > m.CheckSum {
			m.CheckLimit = m.CheckSum
		}
	}

	if cfg.CheckTimeout > 0 {
		if cfg.CheckLimit > 10 {
			m.CheckTimeout = 10
		} else {
			m.CheckTimeout = cfg.CheckTimeout
		}
	}
	if cfg.RecoverSum > 0 {
		m.RecoverSum = cfg.RecoverSum
		if m.RecoverSum > 30 {
			m.RecoverSum = 30
		}
	}

}

func (m *NetMonitor) ResetDefault(value, metric string) {
	switch value {
	case consts.Yes:
		m.DefaultCheckMetric = metric
	case consts.No:
		m.DefaultCheckMetric = consts.BlankMetric
	}
}

func (m *NetMonitor) ResetAddr(ext, exclude []string) {
	m.AddrExt = ext
	m.AddrExclude = exclude
}

func (m *NetMonitor) CheckAddrExist(addr string) bool {
	exist := false
	for _, v := range m.AddrExt {
		if v == addr {
			exist = true
		}
	}
	return exist
}

func (m *NetMonitor) Alarm() {
	m.Alarming = true
}

func (m *NetMonitor) Recover() {
	m.Alarming = false
}

func (m *NetMonitor) SendAlarm(alarmInfo NetAlarmInfo) {
	m.AlarmInfoChan <- alarmInfo
}

func (m *NetMonitor) StartMonitor(metric string) {
	wg := new(sync.WaitGroup)
	wg.Add(2)
	go m.check(metric, wg)
	go m.review(wg)
	wg.Wait()
}

func (m *NetMonitor) check(metric string, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		time.Sleep(time.Duration(m.CheckStep) * time.Second)
		cfg := BaseMonitorConfig{
			CheckSum:     m.CheckSum,
			CheckLimit:   m.CheckLimit,
			CheckStep:    m.CheckStep,
			CheckTimeout: m.CheckTimeout,
			RecoverSum:   m.RecoverSum,
		}
		if m.DefaultCheckMetric != consts.BlankMetric && m.DefaultCheckAddr != nil {
			l := ListFiltration(m.DefaultCheckAddr(), m.AddrExclude)
			for _, addr := range l {
				if addr == "" {
					continue
				}
				go m.CheckFunc(addr, m.DefaultCheckMetric, cfg, m)
			}
		}
		if len(m.AddrExt) == 0 {
			continue
		}
		addrList := make([]string, 0)
		addrList = append(addrList, m.AddrExt...)
		for _, addr := range addrList {
			if addr == "" || m.CheckFunc == nil {
				continue
			}
			go m.CheckFunc(addr, metric, cfg, m)
		}
	}
}

func (m *NetMonitor) review(wg *sync.WaitGroup) {
	defer wg.Done()
	for v := range m.AlarmInfoChan {
		if !v.Ok && m.ReviewFunc != nil {
			go m.ReviewFunc(v, m)
		}
	}
}

func NewNetMonitor(name string,
	checkFunc func(string, string, BaseMonitorConfig, *NetMonitor) NetAlarmInfo,
	reviewFunc func(NetAlarmInfo, *NetMonitor),
	defaultFunc func() []string) *NetMonitor {
	return &NetMonitor{
		BaseMonitorConfig: BaseMonitorConfig{
			CheckSum:     10,
			CheckLimit:   3,
			CheckStep:    60,
			CheckTimeout: 10,
			RecoverSum:   3,
		},
		MonitorName:        name,
		DefaultCheckMetric: consts.BlankMetric,
		AddrExt:            make([]string, 0),
		AddrExclude:        make([]string, 0),

		Alarming:         false,
		AlarmInfoChan:    make(chan NetAlarmInfo, 10),
		CheckFunc:        checkFunc,
		ReviewFunc:       reviewFunc,
		DefaultCheckAddr: defaultFunc,
	}
}

func DealAddrList(ext, exclude string, allowPort bool) (addrList []string) {
	var (
		extList    = make([]string, 0)
		excludeMap = make(map[string]bool)
	)
	addrList = make([]string, 0)
	if ext != "" {
		for _, addr := range strings.Split(ext, ",") {
			extList = append(extList, dealAddr(addr, allowPort)...)
		}
	}
	if exclude != "" {
		for _, v := range strings.Split(exclude, ",") {
			for _, addr := range dealAddr(v, allowPort) {
				excludeMap[addr] = true
			}
		}
		for _, addr := range extList {
			if _, ok := excludeMap[addr]; !ok {
				addrList = append(addrList, addr)
			}
		}
	} else {
		addrList = append(addrList, extList...)
	}
	return
}

func ListFiltration(ext, exclude []string) (result []string) {
	var (
		excludeMap = make(map[string]bool)
	)
	result = make([]string, 0)
	if len(exclude) == 0 {
		return ext
	}
	for _, v := range exclude {
		excludeMap[v] = true
	}
	for _, v := range ext {
		if _, ok := excludeMap[v]; !ok {
			result = append(result, v)
		}
	}
	return
}

func dealAddr(addr string, allowPort bool) (result []string) {
	result = make([]string, 0)
	if strings.Contains(addr, ":") {
		if !allowPort {
			return
		}
		l := strings.Split(addr, ":")
		host := l[0]
		for _, portStr := range l[1:] {
			port, err := strconv.Atoi(portStr)
			if err != nil || port == 0 {
				continue
			}
			result = append(result, fmt.Sprintf("%v:%v", host, port))
		}
	} else {
		result = []string{addr}
	}
	return
}
