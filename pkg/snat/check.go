package snat

import (
	"bytes"
	"cds-k8s-tools/pkg/client"
	"cds-k8s-tools/pkg/service"
	"cds-k8s-tools/pkg/utils"
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	CDS_OVERSEA = "CDS_OVERSEA"
)

type Monitor struct {
	Data        []int
	Index       int
	Sum         int
	Limit       int
	Error       bool
	RecoverStat int
	Step        int
	Host        string
}

var SMonitor *Monitor

func init() {
	SMonitor = &Monitor{
		Data:        make([]int, 10),
		Index:       0,
		Sum:         10,
		Limit:       6,
		Error:       false,
		RecoverStat: 0,
		Step:        60,
		Host:        "",
	}
	oversea := os.Getenv(CDS_OVERSEA)
	switch oversea {
	case "True":
		SMonitor.Host = "www.google.com"
	default:
		SMonitor.Host = "www.baidu.com"
	}
	// SMonitor.ChangeMonitor()
	log.Infof("%v", SMonitor)
}

func (m *Monitor) ChangeMonitor() {
	log.Infof("ChangeMonitor")
	// 检查的间隔
	step := conf.GetKeyInt("default.snat.check.step")
	if step != 0 {
		m.Step = step
	}
	// ping的地址
	host := conf.GetKeyString("default.snat.check.host")
	if host != "" {
		m.Host = host
	}
	// 检查的sum次内
	sum := conf.GetKeyInt("default.snat.check.sum")
	if sum != 0 {
		m.Sum = sum
		m.Data = make([]int, sum)
		m.Index = 0
	}
	// 出现了limit次不通
	limit := conf.GetKeyInt("default.snat.check.limit")
	if limit != 0 {
		m.Limit = limit
	}
}

func (m *Monitor) Put(data int) {
	m.Data[m.Index] = data
	m.Index++
	if m.Index >= m.Sum {
		m.Index = 0
	}
}

func (m *Monitor) RecoverUp() {
	m.RecoverStat += 1

}

func (m *Monitor) RecoverFail() {
	m.RecoverStat = 0
}

func (m *Monitor) Alarm() bool {
	n := 0
	for _, data := range m.Data {
		n += data
	}
	if n >= m.Limit {
		m.Error = true
	} else {
		m.Error = false
	}
	return m.Error
}

func (m *Monitor) Recover() bool {
	if m.RecoverStat >= 3 {
		m.Error = false
		m.Data = make([]int, m.Sum)
		m.RecoverStat = 0
		return true
	}
	return false
}

func CheckSNat(wg *sync.WaitGroup) {
	defer wg.Done()
	var seq int16 = 1
	for {
		time.Sleep(time.Duration(SMonitor.Step) * time.Second)
		if !SMonitor.Error {
			success, _ := utils.Ping(SMonitor.Host, seq)
			if success {
				SMonitor.Put(0)
			} else {
				log.Infof("ping %s error", SMonitor.Host)
				SMonitor.Put(1)
			}
			if SMonitor.Alarm() {
				// 进入告警状态，发送告警请求
				alarm(&service.AlarmMessage{
					NodeName: os.Getenv(NodeNameKey),
					Status:   "error",
					Msg:      "SNat出网异常",
				})
				go checkRecover()
			}
			seq++
		}
	}
}

func checkRecover() {
	var seq int16 = 1
	for {
		log.Infof("Recover Check")
		time.Sleep(30 * time.Second)
		success, _ := utils.Ping(SMonitor.Host, seq)
		if success {
			SMonitor.RecoverUp()
		} else {
			SMonitor.RecoverFail()
		}
		if SMonitor.Recover() {
			// 恢复, 发送回复请求
			alarm(&service.AlarmMessage{
				NodeName: os.Getenv(NodeNameKey),
				Status:   "recover",
				Msg:      "SNat出网能力恢复",
			})
			return
		}
		seq++
	}
}

func alarm(msg *service.AlarmMessage) {
	log.Infof("%v", msg)
	alarmService, err := client.Sa.CoreV1().Services("kube-system").
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Errorf("get cds-alarm-service error: %v", err)
		return
	}
	ip := ""
	port := 39989
	for _, v := range alarmService.Items {
		if v.Name == "cds-alarm-service" {
			ip = v.Spec.ClusterIP
			if len(v.Spec.Ports) > 0 {
				port = int(v.Spec.Ports[0].Port)
			}
			break
		}
	}
	if ip == "" {
		log.Errorf("don't have cds-alarm-service")
		return
	}
	log.Infof("cds-alarm-service ClusterIP: %v", ip)
	req := service.AlarmReq{
		Name:      "容器重要告警",
		SubObject: "容器",
		Type:      "paas",
		Group:     "cloud_native",
		Level:     "Alert",
		Message:   msg,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}
	body, err := json.Marshal(req)
	_, err = utils.DoRequest(http.MethodPost, fmt.Sprintf("http://%s:%d/alarm", ip, port), bytes.NewBuffer(body))
	if err != nil {
		return
	}
}
