package podalarm

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
)

// 自动删除的硬编码保护命名空间：即使这些命名空间被写进 pod.autodelete.namespaces
// 也不会被处理。系统组件卡在 Terminating 往往有更深层原因，误删影响面巨大。
var hardExcludedNamespaces = map[string]bool{
	"kube-system":     true,
	"kube-public":     true,
	"kube-node-lease": true,
}

// SkipAutoDeleteAnnotation 业务方给 Pod 打上该标注且值为 "true" 时，永不被自动删除。
const SkipAutoDeleteAnnotation = "cds.cck.net/skip-auto-delete"

// autoDeleteOwnerKinds 允许自动删除的 Pod 所属工作负载类型。
// 只放无状态类型：StatefulSet 有身份/存储语义，DaemonSet 与节点强绑定，裸 Pod 无人重建。
var autoDeleteOwnerKinds = map[string]bool{
	"ReplicaSet": true, // Deployment 的下级
	"Job":        true,
}

// isAutoDeleteEligible 判断一个 Terminating Pod 是否允许被自动强制删除。
//
// eligible=false 且 reason 非空表示"跳过"，reason 会打进日志便于排查。
// 返回 needManual=true 表示该 Pod 需要人工处理（存在 finalizer 的情况），
// 调用方应单独按 warn 级别输出，而不是当作普通跳过。
// 返回 inScope=true 表示该 Pod 所属命名空间在自动删除范围内，即"确实被考虑过"；
// 这类跳过原因对运维排查有价值，调用方应按 info 级别输出，
// 而 inScope=false（命名空间未配置/不在范围内）属于全集群范围的正常噪音，只按 debug 输出。
func isAutoDeleteEligible(pod *v1.Pod, targetNS map[string]bool) (eligible bool, needManual bool, inScope bool, reason string) {
	// 1. 命名空间必须显式配置。targetNS 为空表示功能未生效。
	if len(targetNS) == 0 {
		return false, false, false, "未配置 pod.autodelete.namespaces，功能未生效"
	}
	if !targetNS[pod.Namespace] {
		return false, false, false, fmt.Sprintf("命名空间 %s 不在自动删除范围内", pod.Namespace)
	}

	// 2. 硬编码的系统命名空间保护，优先级高于配置。
	if hardExcludedNamespaces[pod.Namespace] {
		return false, false, true, fmt.Sprintf("命名空间 %s 属于系统保留命名空间，禁止自动删除", pod.Namespace)
	}

	// 3. 业务方显式豁免。
	if strings.EqualFold(pod.Annotations[SkipAutoDeleteAnnotation], "true") {
		return false, false, true, fmt.Sprintf("已通过标注 %s=true 豁免", SkipAutoDeleteAnnotation)
	}

	// 4. 只处理无状态工作负载拥有的 Pod。
	ownerKind := primaryOwnerKind(pod)
	if ownerKind == "" {
		return false, false, true, "没有 ownerReferences（裸 Pod），不会自动重建，跳过"
	}
	if !autoDeleteOwnerKinds[ownerKind] {
		return false, false, true, fmt.Sprintf("owner 类型为 %s，不属于无状态工作负载，跳过", ownerKind)
	}

	// 5. 挂了 PVC 的一律跳过：节点 NotReady 时强删可能导致同一块盘被两个进程同时写入。
	for _, v := range pod.Spec.Volumes {
		if v.PersistentVolumeClaim != nil {
			return false, false, true, fmt.Sprintf("挂载了 PVC(%s)，跳过", v.PersistentVolumeClaim.ClaimName)
		}
	}

	// 6. 存在 finalizer：强删对象也不会消失，需要人工介入。
	if len(pod.Finalizers) > 0 {
		return false, true, true, fmt.Sprintf("存在 finalizer [%s]，需人工处理", strings.Join(pod.Finalizers, ", "))
	}

	return true, false, true, ""
}

// primaryOwnerKind 返回 Pod 的第一个 owner 类型，没有则返回空串。
func primaryOwnerKind(pod *v1.Pod) string {
	if len(pod.OwnerReferences) == 0 {
		return ""
	}
	return pod.OwnerReferences[0].Kind
}
