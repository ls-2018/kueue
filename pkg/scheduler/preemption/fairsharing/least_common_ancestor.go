package fairsharing

import "sigs.k8s.io/kueue/pkg/cache"

// almostLCA 定义在两个 ClusterQueue 上，表示最低共享节点（即最小公共祖先，LCA）之前的两个节点。LCA 总是 Cohort，而 almostLCA 可能是 ClusterQueue 或 Cohort。
type almostLCA interface {
	DominantResourceShare() int
}

// getLCA 从 ClusterQueue 向根 Cohort 遍历，
// 返回第一个在其子树中包含抢占者 ClusterQueue 的 Cohort。
func getLCA(t *TargetClusterQueue) *cache.CohortSnapshot {
	for ancestor := range t.targetCq.PathParentToRoot() {
		if t.ordering.onPathFromRootToPreemptorCQ(ancestor) {
			return ancestor
		}
	}
	// 让编译器满意
	panic("严重错误：未找到最小公共祖先（LeastCommonAncestor）")
}

// getAlmostLCA 从 ClusterQueue 向根遍历，
// 返回第一个其父节点为 LeastCommonAncestor 的 Cohort 或 ClusterQueue。
func getAlmostLCA(cq *cache.ClusterQueueSnapshot, lca *cache.CohortSnapshot) almostLCA {
	var aLca almostLCA = cq
	for ancestor := range cq.PathParentToRoot() {
		if ancestor == lca {
			return aLca
		}
		aLca = ancestor
	}
	// 让编译器满意
	panic("严重错误：未找到 AlmostLeastCommonAncestor")
}

func getAlmostLCAs(t *TargetClusterQueue) (almostLCA, almostLCA) {
	lca := getLCA(t)
	return getAlmostLCA(t.ordering.preemptorCq, lca), getAlmostLCA(t.targetCq, lca)
}
