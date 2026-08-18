package podalarm

// PodStatusAlarmReq 发送给 cckservice 的 Pod 状态告警请求体
type PodStatusAlarmReq struct {
	ClusterId                   string               `json:"cluster_id"`
	PendingPods                 []PendingPodInfo     `json:"pending_pods"`
	TerminatingPods             []TerminatingPodInfo `json:"terminating_pods"`
	PendingThresholdMinutes     int                  `json:"pending_threshold_minutes"`
	TerminatingThresholdMinutes int                  `json:"terminating_threshold_minutes"`
}

// PendingPodInfo Pending 状态的 Pod，line 为 kubectl get pod -o wide 的整行
type PendingPodInfo struct {
	Name string `json:"name"`
	Line string `json:"line"`
}

// TerminatingPodInfo Terminating 状态的 Pod
type TerminatingPodInfo struct {
	Namespace       string `json:"namespace"`
	Name            string `json:"name"`
	Node            string `json:"node"`
	NodeIP          string `json:"node_ip"`
	DurationSeconds int64  `json:"duration_seconds"`
}
