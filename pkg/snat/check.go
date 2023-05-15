package snat

import (
	"bytes"
	"cds-k8s-tools/pkg/client"
	"cds-k8s-tools/pkg/oscmd"
	"cds-k8s-tools/pkg/service"
	"cds-k8s-tools/pkg/utils"
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	CDS_OVERSEA = "CDS_OVERSEA"
)

type PingInfo struct {
	Success bool
	Ip      string
}

var (
	CheckWorkerChan = make(chan *PingInfo, 5)
	CheckPodChan    = make(chan *PingInfo, 5)
)

type Monitor struct {
	Sum         int
	Limit       int
	WorkerError bool
	PodError    bool
	RecoverSum  int
	Step        int
	Host        string
}

var SMonitor *Monitor

func init() {
	SMonitor = &Monitor{
		Sum:         10,
		Limit:       3,
		WorkerError: false,
		PodError:    false,
		RecoverSum:  3,
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
		if step < 30 {
			m.Step = 30
		} else {
			m.Step = step
		}
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
	}
	// 出现了limit次不通
	limit := conf.GetKeyInt("default.snat.check.limit")
	if limit != 0 {
		m.Limit = limit
	}

	// 检查出网能力recover次成功后恢复
	r := conf.GetKeyInt("default.snat.check.recover")
	if r != 0 {
		m.RecoverSum = r
	}
}

func (m *Monitor) Alarm(isPod bool) string {
	info := fmt.Sprintf("集群%s：", os.Getenv(NodeNameKey))
	if isPod {
		m.PodError = true
		info += "Pod出网异常"
	} else {
		m.WorkerError = true
		info += "节点出网异常"
	}
	return info
}

func (m *Monitor) Recover(isPod bool) string {
	info := fmt.Sprintf("集群%s：", os.Getenv(NodeNameKey))
	if isPod {
		m.PodError = false
		info += "Pod出网恢复"
	} else {
		m.WorkerError = false
		info += "节点出网恢复"
	}
	return info
}

func CheckWorkerResult(wg *sync.WaitGroup) {
	defer wg.Done()
	for v := range CheckWorkerChan {
		if !v.Success {
			if !SMonitor.WorkerError {
				// 进入异常状态
				go checkRecover(v.Ip, false)
			}
		}
	}
}

func CheckPodResult(wg *sync.WaitGroup) {
	defer wg.Done()
	for v := range CheckPodChan {
		if !v.Success {
			if !SMonitor.PodError {
				// 进入异常状态
				go checkRecover(v.Ip, true)
			}
		}
	}
}

func CheckSNat(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		time.Sleep(time.Duration(SMonitor.Step) * time.Second)
		if !SMonitor.WorkerError {
			for _, dns := range getDnsFromNode() {
				go ping(dns, SMonitor.Sum, SMonitor.Limit, false, true)
			}
		}
		if !SMonitor.PodError {
			go ping(SMonitor.Host, SMonitor.Sum, SMonitor.Limit, true, true)
		}
	}
}

func checkRecover(host string, isPod bool) {
	var (
		ok      = 0
		fail    = 0
		isAlarm = false
	)
	for {
		log.Infof("Recover Check")
		time.Sleep(20 * time.Second)
		success, pingInfo := ping(host, SMonitor.Sum, SMonitor.Limit, isPod, false)
		if success {
			ok++
		} else {
			ok = 0
			fail++
		}
		if ok >= SMonitor.RecoverSum {
			if isAlarm {
				info := SMonitor.Recover(isPod)
				// 恢复, 发送回复请求
				alarm(&service.AlarmMessage{
					NodeName: os.Getenv(NodeNameKey),
					Status:   "recover",
					Msg:      info,
				})
			}
			return
		}
		if fail >= 4 && !isAlarm {
			info := SMonitor.Alarm(isPod)
			alarm(&service.AlarmMessage{
				NodeName: os.Getenv(NodeNameKey),
				Status:   "error",
				Msg:      fmt.Sprintf("%s(ping %s 丢包: %s)", info, host, pingInfo),
			})
			isAlarm = true
		}
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

func ping(host string, sum, limit int, isPod, sendChan bool) (bool, string) {
	pingInfo := ""
	ok := true
	pingCmd := fmt.Sprintf("ping %s -c %d", host, sum)
	out, err := oscmd.Run("sh", "-c", pingCmd)
	if err != nil {
		ok = false
		log.Errorf("ping %v cmd err: %v", host, err)
	} else {
		l := strings.Split(out, "\n")
		for _, data := range l {
			if strings.Contains(data, "packet loss") {
				pingInfo = data
				ll := strings.Split(data, ",")
				for _, lossData := range ll {
					if strings.Contains(lossData, "packet loss") {
						str := strings.ReplaceAll(strings.TrimSpace(lossData), "% packet loss", "")
						num, _ := strconv.ParseFloat(str, 64)
						if num/100 >= float64(limit)/float64(sum) {
							ok = false
						} else {
							ok = true
						}
						log.Infof("ping %s request: [%.2f/%.2f](%v)", host, num/100, float64(limit)/float64(sum), lossData)
					}
				}
				break
			}
		}
	}
	if !ok && sendChan {
		if isPod {
			CheckPodChan <- &PingInfo{
				Success: ok,
				Ip:      host,
			}
		} else {
			CheckWorkerChan <- &PingInfo{
				Success: ok,
				Ip:      host,
			}
		}
	}
	return ok, pingInfo
}

func getDnsFromNode() (dnsList []string) {
	result, err := oscmd.CmdToNode("grep 'nameserver' /etc/resolv.conf")
	if err != nil {
		return
	}
	log.Infof("%s", result)
	list := strings.Split(strings.TrimSpace(result), "\n")
	for _, v := range list {
		v = strings.ReplaceAll(v, " ", "")
		dns := strings.ReplaceAll(v, "nameserver", "")
		addr := net.ParseIP(dns)
		if addr != nil {
			dnsList = append(dnsList, dns)
		}
	}
	log.Infof("%v", dnsList)
	return
}
