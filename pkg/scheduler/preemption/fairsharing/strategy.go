package fairsharing

// PreemptorNewShare 表示抢占者在加入新工作负载的使用量后，其主导资源份额。用于规则 S2-a 和 S2-b。
type PreemptorNewShare int

// TargetNewShare 表示被抢占者在其被抢占的工作负载使用量被移除后的主导资源份额。用于规则 S2-a。
type TargetNewShare int

// TargetOldShare 表示被抢占者在其工作负载被移除前的主导资源份额。用于规则 S2-b。
type TargetOldShare int

type Strategy func(PreemptorNewShare, TargetOldShare, TargetNewShare) bool

// LessThanOrEqualToFinalShare 实现了规则 S2-a，详见 https://sigs.k8s.io/kueue/keps/1714-fair-sharing#choosing-workloads-from-clusterqueues-for-preemption
func LessThanOrEqualToFinalShare(preemptorNewShare PreemptorNewShare, _ TargetOldShare, targetNewShare TargetNewShare) bool {
	return int(preemptorNewShare) <= int(targetNewShare)
}

// LessThanInitialShare 实现了规则 S2-b，详见 https://sigs.k8s.io/kueue/keps/1714-fair-sharing#choosing-workloads-from-clusterqueues-for-preemption
func LessThanInitialShare(preemptorNewShare PreemptorNewShare, targetOldShare TargetOldShare, _ TargetNewShare) bool {
	return int(preemptorNewShare) < int(targetOldShare)
}
