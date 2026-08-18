package podalarm

import (
	"cds-k8s-tools/pkg/client"
	"cds-k8s-tools/pkg/consts"
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"time"
)

// Run 启动 Pod 状态监控主循环
func Run() {
	clusterId := os.Getenv(consts.CDS_CLUSTER_ID)
	if clusterId == "" {
		log.Fatalf("env %s is empty", consts.CDS_CLUSTER_ID)
	}

	// 启用配置热更新
	conf.WatchConf()

	log.Infof("cds-pod-alarm start, cluster=%s, check step=%ds", clusterId, getCheckStep())

	for {
		checkAndSend(clusterId)
		time.Sleep(time.Duration(getCheckStep()) * time.Second)
	}
}

// checkAndSend 检测 pending/terminating Pod 并上报
func checkAndSend(clusterId string) {
	namespaces := getNamespaces()
	pendingThresholdMinutes := getPendingThresholdMinutes()
	terminatingThresholdMinutes := getTerminatingThresholdMinutes()
	pendingThreshold := time.Duration(pendingThresholdMinutes) * time.Minute
	terminatingThreshold := time.Duration(terminatingThresholdMinutes) * time.Minute

	pods, err := client.Sa.CoreV1().Pods(metav1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Errorf("list pods err: %v", err)
		return
	}

	nsSet := make(map[string]bool, len(namespaces))
	for _, ns := range namespaces {
		nsSet[ns] = true
	}

	req := PodStatusAlarmReq{
		ClusterId:                   clusterId,
		PendingThresholdMinutes:     pendingThresholdMinutes,
		TerminatingThresholdMinutes: terminatingThresholdMinutes,
	}
	now := time.Now()

	for i := range pods.Items {
		pod := &pods.Items[i]
		if len(nsSet) > 0 && !nsSet[pod.Namespace] {
			continue
		}

		if pod.Status.Phase == v1.PodPending && now.Sub(pod.CreationTimestamp.Time) > pendingThreshold {
			req.PendingPods = append(req.PendingPods, PendingPodInfo{
				Name: pod.Name,
				Line: buildPendingLine(pod, now),
			})
		}

		if pod.DeletionTimestamp != nil && now.Sub(pod.DeletionTimestamp.Time) > terminatingThreshold {
			req.TerminatingPods = append(req.TerminatingPods, buildTerminatingInfo(pod, now))
		}
	}

	if len(req.PendingPods) == 0 && len(req.TerminatingPods) == 0 {
		log.Debugf("no abnormal pod, cluster=%s", clusterId)
		return
	}

	log.Infof("detect %d pending pods, %d terminating pods, cluster=%s",
		len(req.PendingPods), len(req.TerminatingPods), clusterId)

	if err := sendPodStatusAlarm(req); err != nil {
		log.Errorf("send pod status alarm err: %v", err)
	}
}

// buildPendingLine 拼装 kubectl get pod -o wide 的整行（NAME/READY/STATUS/RESTARTS/AGE/IP/NODE/NOMINATED NODE/READINESS GATES）
func buildPendingLine(pod *v1.Pod, now time.Time) string {
	name := pod.Name
	ready := fmt.Sprintf("%d/%d", readyContainers(pod), len(pod.Spec.Containers))
	status := string(pod.Status.Phase)
	restarts := restartCount(pod)
	age := humanizeAge(now.Sub(pod.CreationTimestamp.Time))

	ip := pod.Status.PodIP
	if ip == "" {
		ip = "<none>"
	}
	node := pod.Spec.NodeName
	if node == "" {
		node = "<none>"
	}
	nominated := pod.Status.NominatedNodeName
	if nominated == "" {
		nominated = "<none>"
	}
	gates := readinessGates(pod)

	return fmt.Sprintf("%s\t%s\t%s\t%d\t%s\t%s\t%s\t%s\t%s",
		name, ready, status, restarts, age, ip, node, nominated, gates)
}

// buildTerminatingInfo 采集 terminating Pod 的信息（对齐 shell 脚本：namespace/name/spec.nodeName/status.hostIP）
func buildTerminatingInfo(pod *v1.Pod, now time.Time) TerminatingPodInfo {
	return TerminatingPodInfo{
		Namespace:       pod.Namespace,
		Name:            pod.Name,
		Node:            pod.Spec.NodeName,
		NodeIP:          pod.Status.HostIP,
		DurationSeconds: int64(now.Sub(pod.DeletionTimestamp.Time).Seconds()),
	}
}

func readyContainers(pod *v1.Pod) int {
	ready := 0
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Ready {
			ready++
		}
	}
	return ready
}

func restartCount(pod *v1.Pod) int32 {
	var restarts int32
	for _, cs := range pod.Status.ContainerStatuses {
		restarts += cs.RestartCount
	}
	for _, cs := range pod.Status.InitContainerStatuses {
		restarts += cs.RestartCount
	}
	return restarts
}

// humanizeAge 复刻 kubectl 的 AGE 显示（最多两个时间单位）
func humanizeAge(d time.Duration) string {
	seconds := int(d.Seconds())
	if seconds < 0 {
		return "<invalid>"
	}
	if seconds == 0 {
		return "0s"
	}
	minutes := seconds / 60
	if minutes == 0 {
		return fmt.Sprintf("%ds", seconds)
	}
	hours := minutes / 60
	if hours == 0 {
		remainingSeconds := seconds % 60
		if remainingSeconds == 0 {
			return fmt.Sprintf("%dm", minutes)
		}
		return fmt.Sprintf("%dm%ds", minutes, remainingSeconds)
	}
	days := hours / 24
	if days == 0 {
		remainingMinutes := minutes % 60
		if remainingMinutes == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dh%dm", hours, remainingMinutes)
	}
	remainingHours := hours % 24
	if remainingHours == 0 {
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dd%dh", days, remainingHours)
}

func readinessGates(pod *v1.Pod) string {
	total := len(pod.Spec.ReadinessGates)
	if total == 0 {
		return "<none>"
	}
	ready := 0
	for _, gate := range pod.Spec.ReadinessGates {
		for _, cond := range pod.Status.Conditions {
			if cond.Type == gate.ConditionType && cond.Status == v1.ConditionTrue {
				ready++
				break
			}
		}
	}
	return fmt.Sprintf("%d/%d", ready, total)
}
