package snat

import (
	"bytes"
	"cds-k8s-tools/pkg/client"
	"cds-k8s-tools/pkg/consts"
	"cds-k8s-tools/pkg/monitor"
	"cds-k8s-tools/pkg/service"
	"cds-k8s-tools/pkg/utils"
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strings"
	"time"
)

func ChangeMonitor() {
	// 检查的间隔
	step := conf.GetKeyInt(consts.DefaultConfig, consts.CheckStep)

	// 检查的sum次内
	sum := conf.GetKeyInt(consts.DefaultConfig, consts.CheckSum)

	// 出现了limit次不通
	limit := conf.GetKeyInt(consts.DefaultConfig, consts.CheckLimit)

	// 检查出网能力recover次成功后恢复
	recoverSum := conf.GetKeyInt(consts.DefaultConfig, consts.CheckRecoverSum)

	// 网络连通性超时时间
	timeout := conf.GetKeyInt(consts.DefaultConfig, consts.CheckTimeout)

	cfg := monitor.BaseMonitorConfig{
		CheckSum:     sum,
		CheckLimit:   limit,
		CheckStep:    step,
		CheckTimeout: timeout,
		RecoverSum:   recoverSum,
	}

	TelnetMonitor.ResetCommon(cfg)
	NodePingMonitor.ResetCommon(cfg)
	PodPingMonitor.ResetCommon(cfg)

	pingDefault := conf.GetKeyString(consts.DefaultConfig, consts.CheckPodDefaultPing)
	PodPingMonitor.ResetDefault(pingDefault, consts.CheckPodDefaultPing)

	pingDns := conf.GetKeyString(consts.DefaultConfig, consts.CheckNodeDns)
	NodePingMonitor.ResetDefault(pingDns, consts.CheckNodeDns)

	// pod ping检查地址
	podPingExt := conf.GetKeyString(consts.DefaultConfig, consts.CheckPodPingExt)
	podCheckAddr := monitor.DealAddrList(podPingExt, "", false)
	PodPingMonitor.ResetAddr(podCheckAddr, nil)

	// node ping检查地址
	nodePingExt := conf.GetKeyString(consts.DefaultConfig, consts.CheckNodePingExt)
	nodePingExclude := conf.GetKeyString(consts.DefaultConfig, consts.CheckNodePingExclude)
	nodeCheckAddr := monitor.DealAddrList(nodePingExt, nodePingExclude, false)
	NodePingMonitor.ResetAddr(nodeCheckAddr, strings.Split(nodePingExclude, ","))

	// pod telnet检查地址
	telnetExt := conf.GetKeyString(consts.DefaultConfig, consts.CheckPodTelnetExt)
	telnetAddr := monitor.DealAddrList(telnetExt, "", true)
	TelnetMonitor.ResetAddr(telnetAddr, nil)

	log.Infof("change monitor success")
}

func CheckSNat() {
	go TelnetMonitor.StartMonitor(consts.CheckPodTelnetExt)
	go NodePingMonitor.StartMonitor(consts.CheckNodePingExt)
	go PodPingMonitor.StartMonitor(consts.CheckPodPingExt)
}

func alarm(msg *service.AlarmMessage) {
	alarmService, err := client.Sa.CoreV1().Services(consts.AlarmPodNamespace).
		List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Errorf("get cds-alarm-service error: %v", err)
		return
	}
	ip := ""
	port := consts.AlarmPodServiceDefaultPort
	for _, v := range alarmService.Items {
		if v.Name == consts.AlarmPodName {
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

	req := service.AlarmInstance{
		Type:        msg.Type,
		Key:         msg.Metric,
		Value:       msg.Value,
		Description: msg.Msg,
		Instance:    msg.NodeName,
		EventTime:   time.Now().Format("2006-01-02 15:04:05"),
	}
	body, err := json.Marshal(req)
	_, err = utils.DoRequest(http.MethodPost,
		fmt.Sprintf("http://%s:%d%s", ip, port, consts.AlarmServiceV2Route), bytes.NewBuffer(body))
	if err != nil {
		log.Errorf("request cds-alarm-service service err: %v", err)
		return
	}
}
