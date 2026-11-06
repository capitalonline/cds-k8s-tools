package podalarm

import (
	"cds-k8s-tools/pkg/client"
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// runAutoDelete 对一批已超过告警阈值的 Terminating Pod 执行自动删除逻辑。
//
// 只在日志里体现处理结果，不上报 cckservice，不影响原有告警协议/流程。
// pods 里是已超过告警阈值的 Terminating Pod；本函数在此基础上按 autoDeleteThreshold
// （告警阈值 + 额外延迟）二次过滤，只处理真正超过自动删除阈值的 Pod。
func runAutoDelete(pods []*v1.Pod, autoDeleteThreshold time.Duration, now time.Time) {
	if len(pods) == 0 {
		return
	}

	enabled := getAutoDeleteEnabled()

	for _, pod := range pods {
		if now.Sub(pod.DeletionTimestamp.Time) < autoDeleteThreshold {
			continue
		}

		eligible, needManual, reason := isAutoDeleteEligible(pod)
		key := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
		terminatingFor := now.Sub(terminatingSince(pod)).Round(time.Second)

		switch {
		case needManual:
			log.Warnf("[pod-autodelete] %s 已 Terminating %v，%s", key, terminatingFor, reason)
		case !eligible:
			log.Infof("[pod-autodelete] %s 跳过自动删除：%s", key, reason)
		case !enabled:
			log.Infof("[pod-autodelete] [dry-run] 将会强制删除 %s，已 Terminating %v", key, terminatingFor)
		default:
			deletePod(pod, terminatingFor)
		}
	}
}

func deletePod(pod *v1.Pod, terminatingFor time.Duration) {
	key := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
	gracePeriod := int64(0)
	err := client.Sa.CoreV1().Pods(pod.Namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	})
	if err != nil {
		log.Errorf("[pod-autodelete] 强制删除 %s 失败：%v", key, err)
		return
	}
	log.Warnf("[pod-autodelete] 已强制删除 %s，已 Terminating %v", key, terminatingFor)
}
