package snat

import (
	"cds-k8s-tools/pkg/consts"
	"cds-k8s-tools/pkg/monitor"
	"cds-k8s-tools/pkg/service"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"sync"
	"time"
)

var (
	TelnetMonitor      = monitor.NewNetMonitor(consts.NetMonitorTelnet, TcpConn, TelnetReview, nil)
	TelnetAddrAlarmMap = new(sync.Map)
)

func TelnetReview(info monitor.NetAlarmInfo, m *monitor.NetMonitor) {
	if _, ok := TelnetAddrAlarmMap.Load(info.Addr); ok {
		return
	}
	var (
		successSum = 0
		failSum    = 0
		callAlarm  = false
		result     monitor.NetAlarmInfo
		maxFailSum = 3600 / (m.CheckStep / 2)
	)
	TelnetAddrAlarmMap.Store(info.Addr, 1)
	m.Alarm()
	for {
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
			TelnetAddrAlarmMap.Delete(info.Addr)
			if callAlarm {
				// 恢复, 发送回复请求
				alarm(&service.AlarmMessage{
					NodeName: os.Getenv(consts.NODE_NAME),
					Type:     consts.SNatRecoverAlarmType,
					Metric:   info.Metric,
					Value:    info.Addr,
					Msg:      result.Msg,
				})
			}
			return
		}
		if failSum >= m.RecoverSum && !callAlarm {
			alarm(&service.AlarmMessage{
				NodeName: os.Getenv(consts.NODE_NAME),
				Type:     consts.SNatErrorAlarmType,
				Metric:   info.Metric,
				Value:    info.Addr,
				Msg:      result.Msg,
			})
			callAlarm = true
		}
		if failSum > maxFailSum {
			m.Recover()
			TelnetAddrAlarmMap.Delete(info.Addr)
			return
		}
	}
}

func TcpConn(addr, key string, cfg monitor.BaseMonitorConfig, netMonitor *monitor.NetMonitor) monitor.NetAlarmInfo {
	var (
		failSum int
		ok      = true
		tcpInfo string
	)
	for i := 0; i < cfg.CheckSum; i++ {
		if !tcpDialer(addr, cfg.CheckTimeout) {
			failSum++
		}
		time.Sleep(1 * time.Second)
	}
	tcpInfo = fmt.Sprintf("地址【%v】tcp连接成功率%0.1f%%）",
		addr, (float64(cfg.CheckSum-failSum)/float64(cfg.CheckSum))*100)

	log.Infof("%s", tcpInfo)

	if failSum >= cfg.CheckLimit {
		ok = false
	}
	alarmInfo := monitor.NetAlarmInfo{
		Metric: key,
		Value:  ok,
		Addr:   addr,
		Msg:    tcpInfo,
	}
	if netMonitor != nil && !ok {
		netMonitor.SendAlarm(alarmInfo)
	}
	return alarmInfo
}

func tcpDialer(addr string, timeout int) bool {
	dialer := net.Dialer{
		Timeout: time.Duration(timeout) * time.Second,
	}
	conn, err := dialer.Dial(consts.Tcp, addr)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}
