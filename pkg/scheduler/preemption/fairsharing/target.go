package fairsharing

import (
	"sigs.k8s.io/kueue/pkg/cache"
	"sigs.k8s.io/kueue/pkg/workload"
)

// TargetClusterQueue 是一个 ClusterQueue，用于为抢占操作提供候选工作负载。
type TargetClusterQueue struct {
	ordering *TargetClusterQueueOrdering
	targetCq *cache.ClusterQueueSnapshot // 将被抢占的
}

// InClusterQueuePreemption 表示 TargetClusterQueue 是抢占者 ClusterQueue；
// 即抢占者 ClusterQueue 正在考虑其自身的工作负载以进行基于优先级的抢占。
func (t *TargetClusterQueue) InClusterQueuePreemption() bool {
	return t.targetCq == t.ordering.preemptorCq
}

func (t *TargetClusterQueue) PopWorkload() *workload.Info {
	cqt := t.ordering.clusterQueueToTarget
	cqName := t.targetCq.GetName()

	head := cqt[cqName][0]
	cqt[cqName] = cqt[cqName][1:]
	return head
}

// ComputeShares 计算抢占者（preemptor）和目标 ClusterQueue 的
// AlmostLeastCommonAncestors 的 DominantResourceShares（主导资源份额）。
// 这些份额的计算不依赖于当前被考虑用于抢占的工作负载是否被移除。
func (t *TargetClusterQueue) ComputeShares() (PreemptorNewShare, TargetOldShare) {
	preemptorAlmostLCA, targetAlmostLCA := getAlmostLCAs(t) // ✅
	// 生命周期评估LCA
	return PreemptorNewShare(preemptorAlmostLCA.DominantResourceShare()), TargetOldShare(targetAlmostLCA.DominantResourceShare())
}

// ComputeTargetShareAfterRemoval 返回目标 ClusterQueue 的 AlmostLeastCommonAncestor 的 DominantResourceShare，
// 在移除指定工作负载后。
//
// 这种模拟是必须的，以便在每个 ClusterQueue 的父 Cohort 中计入新的使用量。
// 我们不能仅仅在 almostLCA 上进行这个操作，因为 almostLCA 上存储的使用量会依赖于子节点的 LendingLimits。
// 详见 cache.resource_node.go。
func (t *TargetClusterQueue) ComputeTargetShareAfterRemoval(wl *workload.Info) TargetNewShare {
	revertSimulation := t.targetCq.SimulateUsageRemoval(wl.Usage())
	defer revertSimulation()
	_, almostLCA := getAlmostLCAs(t)
	return TargetNewShare(almostLCA.DominantResourceShare())
}

// HasWorkload 判断目标 ClusterQueue 是否有工作负载。
func (t *TargetClusterQueue) HasWorkload() bool {
	return t.ordering.hasWorkload(t.targetCq)
}
