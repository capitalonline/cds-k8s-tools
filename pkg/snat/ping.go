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
		pingInfo = fmt.Sprintf("ping【%s】: ", addr)
		ok       = true
		pingCmd  = fmt.Sprintf("ping %s -c %d", addr, cfg.CheckSum)
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
						left := num / 100
						right := float64(cfg.CheckLimit) / float64(cfg.CheckSum)
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
		Value:  ok,
		Addr:   addr,
		Msg:    pingInfo,
	}

	if netMonitor != nil && !ok {
		netMonitor.SendAlarm(alarmInfo)
	}
	return alarmInfo
}

func PingReview(info monitor.NetAlarmInfo, m *monitor.NetMonitor) {
	if m.Alarming {
		return
	}
	var (
		successSum = 0
		failSum    = 0
		callAlarm  = false
		maxFailSum = 3600 / (m.CheckStep / 2)
		result     monitor.NetAlarmInfo
	)
	if maxFailSum < 10 {
		maxFailSum = 10
	}
	m.Alarm()
	for {
		if !m.CheckAddrExist(info.Addr) {
			m.Recover()
			return
		}
		time.Sleep(time.Duration(m.CheckStep/2) * time.Second)
		result = m.CheckFunc(info.Addr, info.Metric, monitor.BaseMonitorConfig{
			CheckSum:     m.CheckSum,
			CheckLimit:   m.CheckLimit,
			CheckStep:    m.CheckStep,
			CheckTimeout: m.CheckTimeout,
		}, nil)
		if result.Value {
			successSum++
		} else {
			successSum = 0
			failSum++
		}
		if successSum >= m.RecoverSum {
			m.Recover()
			if callAlarm {
				// 恢复, 发送回复请求
				if err := alarm(&service.AlarmMessage{
					NodeName: os.Getenv(consts.NODE_NAME),
					Type:     consts.SNatRecoverAlarmType,
					Metric:   info.Metric,
					Value:    info.Addr,
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
				Value:    info.Addr,
				Msg:      result.Msg,
			})
			if err != nil {
				log.Errorf("call alarm fail: %v", err)
			} else {
				callAlarm = true
			}
		}
		if failSum > maxFailSum {
			m.Recover()
			return
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
