package snat

import (
	"cds-k8s-tools/pkg/consts"
	"cds-k8s-tools/pkg/monitor"
	"cds-k8s-tools/pkg/oscmd"
	"cds-k8s-tools/pkg/service"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	NodePingMonitor = monitor.NewNetMonitor(consts.NetMonitorNodePing, PingConn, PingReview, GetDnsFromNode)
	PodPingMonitor  = monitor.NewNetMonitor(consts.NetMonitorPodPing, PingConn, PingReview, GetDefaultPingAddr)
)

func PingConn(addr, key string, cfg monitor.BaseMonitorConfig, netMonitor *monitor.NetMonitor) monitor.NetAlarmInfo {
	var (
		pingInfo         = fmt.Sprintf("ping【%s】: ", addr)
		ok               = true
		pingCmd          = fmt.Sprintf("ping %s -c %d", addr, cfg.CheckSum)
		left     float64 = 1.0
		right    float64 = float64(cfg.CheckLimit) / float64(cfg.CheckSum)
	)
	out, err := oscmd.CmdRun("sh", "-c", pingCmd)
	if err != nil {
		ok = false
		pingInfo += " 不通"
	} else {
		l := strings.Split(out, "\n")
		for _, data := range l {
			if strings.Contains(data, "packet loss") {
				pingInfo += data
				ll := strings.Split(data, ",")
				for _, lossData := range ll {
					if strings.Contains(lossData, "packet loss") {
						str := strings.ReplaceAll(strings.TrimSpace(lossData), "% packet loss", "")
						num, _ := strconv.ParseFloat(str, 64)
						left = num / 100
						if left >= right {
							ok = false
						} else {
							ok = true
						}
						// log.Infof("ping %s request: [%.2f/%.2f](%v)", addr, left, right, lossData)
					}
				}
				break
			}
		}
	}

	log.Infof("%v", pingInfo)

	alarmInfo := monitor.NetAlarmInfo{
		Metric: key,
		Ok:     ok,
		Addr:   addr,
		Value:  fmt.Sprintf("%0.2f", left),
		Msg:    pingInfo,
	}

	if netMonitor != nil && !ok {
		netMonitor.SendAlarm(alarmInfo)
	}
	return alarmInfo
}

// ReviewTimeout 单个地址进入复查后的最长持续时间。
// 原实现按 "3600/(CheckStep/2)" 次计数来近似 1 小时，但没有计入检测本身的耗时，
// 实测会拖到 1.5 小时以上，这里改为按时间控制。
const ReviewTimeout = 30 * time.Minute

func PingReview(info monitor.NetAlarmInfo, m *monitor.NetMonitor) {
	// 按地址去重：原先用全局 Alarming 标志，同一 monitor 下多个地址同时故障时
	// 只有第一个能进入复查，其余会被丢弃。
	alarmKey := fmt.Sprintf("%s|%s", info.Metric, info.Addr)
	if _, loaded := m.AlarmMap.LoadOrStore(alarmKey, struct{}{}); loaded {
		return
	}
	defer m.AlarmMap.Delete(alarmKey)

	var (
		successSum = 0
		failSum    = 0
		callAlarm  = false
		deadline   = time.Now().Add(ReviewTimeout)
		result     monitor.NetAlarmInfo
	)
	log.Infof("[%s]%s(%v)进入复查阶段", info.Metric, info.Addr, info.Value)
	for {
		if info.Metric != consts.CheckPodDefaultPing && info.Metric != consts.CheckNodeDns {
			if !m.CheckAddrExist(info.Addr) {
				return
			}
		}
		if time.Now().After(deadline) {
			log.Warnf("[%s]%s 复查超过 %v，退出", info.Metric, info.Addr, ReviewTimeout)
			return
		}
		time.Sleep(time.Duration(m.CheckStep/2) * time.Second)
		result = m.CheckWithLimit(info.Addr, info.Metric, monitor.BaseMonitorConfig{
			CheckSum:     m.CheckSum,
			CheckLimit:   m.CheckLimit,
			CheckStep:    m.CheckStep,
			CheckTimeout: m.CheckTimeout,
		})
		if result.Ok {
			successSum++
		} else {
			successSum = 0
			failSum++
		}
		if successSum >= m.RecoverSum {
			if callAlarm {
				// 恢复, 发送回复请求
				if err := alarm(&service.AlarmMessage{
					NodeName: os.Getenv(consts.NODE_NAME),
					Type:     consts.SNatRecoverAlarmType,
					Metric:   info.Metric,
					Value:    result.Value,
					Target:   info.Addr,
					Msg:      result.Msg,
				}); err != nil {
					log.Errorf("call alarm fail: %v", err)
				}
			}
			return
		}
		if failSum >= m.RecoverSum && !callAlarm {
			err := alarm(&service.AlarmMessage{
				NodeName: os.Getenv(consts.NODE_NAME),
				Type:     consts.SNatErrorAlarmType,
				Metric:   info.Metric,
				Value:    result.Value,
				Target:   info.Addr,
				Msg:      result.Msg,
			})
			if err != nil {
				log.Errorf("call alarm fail: %v", err)
			} else {
				callAlarm = true
			}
		}
	}
}

func GetDefaultPingAddr() []string {
	oversea := os.Getenv(consts.CDS_OVERSEA)
	switch oversea {
	case consts.IsOversea:
		return []string{consts.GoogleAddr}

	default:
		return []string{consts.BaiduAddr}
	}
}

func GetDnsFromNode() (dnsList []string) {
	result, err := oscmd.CmdToNode("grep 'nameserver' /etc/resolv.conf")
	if err != nil {
		return
	}

	list := strings.Split(strings.TrimSpace(result), "\n")
	for _, v := range list {
		v = strings.ReplaceAll(v, " ", "")
		dns := strings.ReplaceAll(v, "nameserver", "")
		addr := net.ParseIP(dns)
		if addr != nil {
			dnsList = append(dnsList, dns)
		}
	}
	return
}
