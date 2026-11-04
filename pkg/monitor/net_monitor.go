package monitor

import (
	"cds-k8s-tools/pkg/consts"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// DefaultCheckConcurrency 单个 monitor 允许同时运行的检测数量上限。
// 地址较多时（如 200 个），若不限制并发，每轮会同时拉起等量的 ping/telnet 子进程，
// 且上一轮未结束时下一轮还会继续叠加，最终导致内存无上界增长。
const DefaultCheckConcurrency = 50

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

	// Alarming 已废弃：原先用全局标志对复查去重，导致同一 monitor 下多个地址同时故障时
	// 只有第一个能进入复查并上报。现已改为 AlarmMap 按地址去重，该字段仅为兼容保留。
	Alarming      bool
	AlarmInfoChan chan NetAlarmInfo

	// AlarmMap 按 "指标|地址" 去重，保证多个地址可以并发进入复查、各自上报。
	AlarmMap *sync.Map

	semMu    sync.RWMutex
	checkSem chan struct{}

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

// ResetConcurrency 设置同时运行的检测数量上限，非法值回落到默认值。
// 仅在容量变化时重建信号量，已有 goroutine 持有的是旧信号量，不会互相影响。
func (m *NetMonitor) ResetConcurrency(n int) {
	if n <= 0 || n > 500 {
		n = DefaultCheckConcurrency
	}
	m.semMu.Lock()
	defer m.semMu.Unlock()
	if cap(m.checkSem) == n {
		return
	}
	m.checkSem = make(chan struct{}, n)
}

func (m *NetMonitor) sem() chan struct{} {
	m.semMu.RLock()
	defer m.semMu.RUnlock()
	return m.checkSem
}

// dispatch 异步执行一次检测。并发槽位已满时阻塞等待，而不是跳过：
// 跳过会导致列表尾部的地址永远抢不到槽位（每轮派发顺序相同），相当于漏监控。
// 代价是一轮耗时超过 CheckStep 时实际检查间隔会被拉长，check() 里会打印告警日志。
func (m *NetMonitor) dispatch(addr, metric string, cfg BaseMonitorConfig) {
	sem := m.sem()
	sem <- struct{}{}
	go func() {
		defer func() { <-sem }()
		m.CheckFunc(addr, metric, cfg, m)
	}()
}

// CheckWithLimit 同步执行一次检测，受并发上限约束。复查使用该方法，保证不会被跳过。
func (m *NetMonitor) CheckWithLimit(addr, metric string, cfg BaseMonitorConfig) NetAlarmInfo {
	sem := m.sem()
	sem <- struct{}{}
	defer func() { <-sem }()
	return m.CheckFunc(addr, metric, cfg, nil)
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
		roundStart := time.Now()
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
				m.dispatch(addr, m.DefaultCheckMetric, cfg)
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
			m.dispatch(addr, metric, cfg)
		}
		if elapsed := time.Since(roundStart); elapsed > time.Duration(m.CheckStep)*time.Second {
			log.Warnf("[%s] 本轮检测派发耗时 %v，超过检查频率 %ds，实际间隔会被拉长；请调大 snat.check.step 或 snat.check.concurrency",
				m.MonitorName, elapsed.Round(time.Second), m.CheckStep)
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
		AlarmMap:         new(sync.Map),
		checkSem:         make(chan struct{}, DefaultCheckConcurrency),
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
