package service

import (
	"bytes"
	"cds-k8s-tools/pkg/consts"
	"cds-k8s-tools/pkg/utils"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var AlarmChan = make(chan AlarmInstance, 100)

const alarmFlushInterval = 3 * time.Second

// resetAlarmTimer 重置批量发送计时器。
// Go 1.15+ 要求 Reset 前先 Stop 并排空通道：如果上一次触发发生在
// ReqAlarmOpenApi 阻塞期间，通道里会残留一个已过期的值，
// 直接 Reset 会让下一轮 select 立刻命中，导致同一批告警被重复发送。
func resetAlarmTimer(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(alarmFlushInterval)
}

func AlarmCenter() {
	var (
		err       error
		alarmList = make([]AlarmInstance, 0, 100)
		timer     = time.NewTimer(alarmFlushInterval)
	)

	defer timer.Stop()

	for {
		select {
		case alarmInfo := <-AlarmChan:
			alarmList = append(alarmList, alarmInfo)
			if len(alarmList) >= 100 {
				if err = ReqAlarmOpenApi(CckNewAlarmReq{
					ClusterId:      os.Getenv(consts.CDS_CLUSTER_ID),
					AlarmInstances: alarmList,
				}); err != nil {
					log.Errorf("ReqAlarmOpenApi err: %s", err)
				} else {
					alarmList = make([]AlarmInstance, 0, 100)
				}
			}
			resetAlarmTimer(timer)
		case <-timer.C:
			if len(alarmList) > 0 {
				if err = ReqAlarmOpenApi(CckNewAlarmReq{
					ClusterId:      os.Getenv(consts.CDS_CLUSTER_ID),
					AlarmInstances: alarmList,
				}); err != nil {
					log.Errorf("ReqAlarmOpenApi err: %s", err)
					// 发送失败：保留批次并重置计时器，3 秒后重试。
					// 之前这里漏了 Reset，告警流一旦停止，积压批次就会永久滞留。
					resetAlarmTimer(timer)
				} else {
					alarmList = make([]AlarmInstance, 0, 100)
				}
			}
		}
	}
}

func ReqAlarmOpenApi(request CckNewAlarmReq) error {
	b, _ := json.Marshal(request)
	log.Infof("req alarm openapi body %s", string(b))
	// return nil
	req, _ := utils.NewCCKRequest(consts.SendSNatAlarmInfoV2, http.MethodPost, nil, bytes.NewReader(b))
	response, err := utils.DoOpenApiRequest(req)
	if err != nil {
		return fmt.Errorf("action[%s], DoOpenApiRequest err: %v", consts.SendSNatAlarmInfoV2, err)
	}
	content, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("action[%s] read resp body err: %v", consts.SendSNatAlarmInfoV2, err)
	}
	if response.StatusCode >= 400 || !strings.Contains(string(content), consts.SuccessCode) {
		return fmt.Errorf("action[%s] fail, status=%s, resp=%s", consts.SendSNatAlarmInfoV2, response.Status, string(content))
	}

	return nil
}
