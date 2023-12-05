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

func AlarmCenter() {
	var (
		err       error
		alarmList = make([]AlarmInstance, 0, 100)
		timer     = time.NewTimer(3 * time.Second)
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
			timer.Reset(3 * time.Second)
		case <-timer.C:
			if len(alarmList) > 0 {
				if err = ReqAlarmOpenApi(CckNewAlarmReq{
					ClusterId:      os.Getenv(consts.CDS_CLUSTER_ID),
					AlarmInstances: alarmList,
				}); err != nil {
					log.Errorf("ReqAlarmOpenApi err: %s", err)
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
