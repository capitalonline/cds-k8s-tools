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

	// pod.monitor.namespaces 同时决定告警和自动删除的范围，必须显式配置。
	// 留空会导致范围无法界定（旧版本"留空=监控全部"的语义容易误伤），直接启动失败更安全。
	if len(getNamespaces()) == 0 {
		log.Fatalf("%s is required, configure at least one namespace", MonitorNamespaces)
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
	// 热更新把命名空间清空时，本轮直接跳过；不能退化成"监控全部"，否则自动删除会覆盖全集群。
	if len(namespaces) == 0 {
		log.Errorf("%s is empty, skip this round", MonitorNamespaces)
		return
	}

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
	terminatingPods := make([]*v1.Pod, 0)

	for i := range pods.Items {
		pod := &pods.Items[i]
		if !nsSet[pod.Namespace] {
			continue
		}

		if pod.Status.Phase == v1.PodPending && now.Sub(pendingSince(pod)) > pendingThreshold {
			req.PendingPods = append(req.PendingPods, PendingPodInfo{
				Name: pod.Name,
				Line: buildPendingLine(pod, now),
			})
		}

		if pod.DeletionTimestamp != nil && now.Sub(pod.DeletionTimestamp.Time) > terminatingThreshold {
			req.TerminatingPods = append(req.TerminatingPods, buildTerminatingInfo(pod, now))
			terminatingPods = append(terminatingPods, pod)
		}
	}

	// 自动删除是告警之外的独立旁路：告警走原有协议上报 cckservice，
	// 自动删除只打本地日志，两者共用同一份已采集的 Terminating Pod 列表，互不影响。
	autoDeleteThreshold := terminatingThreshold + time.Duration(getAutoDeleteDelayMinutes())*time.Minute
	runAutoDelete(terminatingPods, autoDeleteThreshold, now)

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

// pendingSince 返回 Pod 进入当前 Pending 状态的近似时间。
// 注意：容器被 kill/crash 后（CrashLoopBackOff、ContainerCreating 重拉镜像等）Pod Phase 会回退为 Pending，
// 但对象不会重建，CreationTimestamp 仍是创建时间；此时应取 Ready condition 最近一次翻转为 False 的时间，
// 否则存量 Pod 会在启动扫描时立即误报。新扩容 Pod 无 Ready condition（未调度）时回退到创建时间。
func pendingSince(pod *v1.Pod) time.Time {
	since := pod.CreationTimestamp.Time
	for _, c := range pod.Status.Conditions {
		if c.Type == v1.PodReady && c.Status != v1.ConditionTrue && c.LastTransitionTime.After(since) {
			since = c.LastTransitionTime.Time
		}
	}
	return since
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

// terminatingSince 返回 Pod 真正开始 Terminating 的时间。
// DeletionTimestamp 存的是"删除截止时间"(删除时刻 + gracePeriod)，不是删除开始时间，
// 直接用它算时长会少算一个优雅退出窗口（默认 30s），需要把 gracePeriod 减回去。
// 带 finalizer 的 Pod 同样满足该关系（实测 deletionTimestamp 仍是删除时刻 + grace）。
func terminatingSince(pod *v1.Pod) time.Time {
	start := pod.DeletionTimestamp.Time
	if pod.DeletionGracePeriodSeconds != nil {
		start = start.Add(-time.Duration(*pod.DeletionGracePeriodSeconds) * time.Second)
	}
	return start
}

// buildTerminatingInfo 采集 terminating Pod 的信息（对齐 shell 脚本：namespace/name/spec.nodeName/status.hostIP）
func buildTerminatingInfo(pod *v1.Pod, now time.Time) TerminatingPodInfo {
	return TerminatingPodInfo{
		Namespace:       pod.Namespace,
		Name:            pod.Name,
		Node:            pod.Spec.NodeName,
		NodeIP:          pod.Status.HostIP,
		DurationSeconds: int64(now.Sub(terminatingSince(pod)).Seconds()),
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
