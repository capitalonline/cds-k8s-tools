package podalarm

import (
	"bytes"
	"cds-k8s-tools/pkg/consts"
	"cds-k8s-tools/pkg/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// sendPodStatusAlarm 调用 cckservice 内部 OpenAPI，上报 Pod 状态告警
func sendPodStatusAlarm(req PodStatusAlarmReq) error {
	b, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal req err: %v", err)
	}

	request, err := utils.NewCCKRequest(consts.SendPodStatusAlarm, http.MethodPost, nil, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("new request err: %v", err)
	}

	response, err := utils.DoOpenApiRequest(request)
	if err != nil {
		return fmt.Errorf("action[%s], DoOpenApiRequest err: %v", consts.SendPodStatusAlarm, err)
	}
	defer response.Body.Close()

	content, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("action[%s] read resp body err: %v", consts.SendPodStatusAlarm, err)
	}

	if response.StatusCode >= 400 || !strings.Contains(string(content), consts.SuccessCode) {
		return fmt.Errorf("action[%s] fail, status=%s, resp=%s", consts.SendPodStatusAlarm, response.Status, string(content))
	}

	return nil
}
