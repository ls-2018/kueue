// 版权所有 Kubernetes 作者。
//
// 根据 Apache 许可证 2.0 版（以下简称“许可证”）授权；
// 除非符合许可证，否则您不得使用此文件。
// 您可以在以下网址获取许可证副本：
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// 除非适用法律要求或书面同意，
// 根据许可证分发的软件按“原样”分发，
// 不附带任何明示或暗示的担保或条件。
// 有关许可证下权限和限制的具体语言，请参阅许可证。

package fairsharing

import (
	"iter"

	"k8s.io/apimachinery/pkg/util/sets"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/cache"
	"sigs.k8s.io/kueue/pkg/workload"
)

// TargetClusterQueueOrdering 定义了在抢占期间考虑的集群队列排序。
// 这是通过从根 cohort 遍历来完成的，选择满足 3 个条件的子 cohort：
// 1. 有候选工作负载
// 2. DRS >= 0
// 3. 具有通过过滤条件 1 和 2 传递的节点最高 DominantResourceShare。
//
// 相同的 TargetClusterQueue 可能会在多行中返回，如果其 AlmostLeastCommonAncestor 的 DominantResourceShare 仍然最高。
//
// 为了保证进度，必须在每次返回条目之间调用 TargetClusterQueue.PopWorkload 或 TargetClusterQueueOrdering.DropQueue。
type TargetClusterQueueOrdering struct {
	preemptorCq          *cache.ClusterQueueSnapshot
	preemptorAncestors   sets.Set[*cache.CohortSnapshot] // 预抢占集群队列的祖先 cohort。
	clusterQueueToTarget map[kueue.ClusterQueueReference][]*workload.Info

	// 已剪枝的集群队列是那些我们确定永远不会
	// 产生更多抢占目标候选者的集群队列。我们使用此集合来
	// 确定我们的停止条件：一旦根 cohort 在 prunedCohorts 列表中，
	// 我们将不再找到任何抢占目标候选者。
	prunedClusterQueues sets.Set[*cache.ClusterQueueSnapshot]
	prunedCohorts       sets.Set[*cache.CohortSnapshot]
}

// DropQueue 表示我们不应再
// 考虑此队列的工作负载。
func (t *TargetClusterQueueOrdering) DropQueue(cq *TargetClusterQueue) {
	t.prunedClusterQueues.Insert(cq.targetCq)
}

// nextTarget 是一个递归算法，用于查找下一个
// TargetClusterQueue。它找到具有最高 DRS 的子 cohort，
// 如果它是集群队列，则返回它；如果是 cohort，则进入递归
// 调用。返回 nil 并不意味着没有更多候选集群队列；
// 迭代可能只修剪了树中的节点。
func (t *TargetClusterQueueOrdering) nextTarget(cohort *cache.CohortSnapshot) *TargetClusterQueue {
	var highestCq *cache.ClusterQueueSnapshot = nil
	highestCqDrs := -1
	for _, cq := range cohort.ChildCQs() {
		if t.prunedClusterQueues.Has(cq) {
			continue
		}

		drs := cq.DominantResourceShare()
		// 我们不能剪枝预抢占集群队列本身，
		// 直到它耗尽候选者。
		if (drs == 0 && cq != t.preemptorCq) || !t.hasWorkload(cq) {
			t.prunedClusterQueues.Insert(cq)
		} else if drs >= highestCqDrs {
			highestCqDrs = drs
			highestCq = cq
		}
	}

	var highestCohort *cache.CohortSnapshot = nil
	highestCohortDrs := -1
	for _, cohort := range cohort.ChildCohorts() {
		if t.prunedCohorts.Has(cohort) {
			continue
		}

		drs := cohort.DominantResourceShare()

		// 当它不再借用时（DRS=0），我们剪枝 cohort。
		// 即使不借用，我们也不能从预抢占集群队列到
		// 根的路径上剪枝，因为某些子树可能存在不平衡，
		// 或者预抢占集群队列本身可能存在抢占。
		// 我们只会在所有子 cohort 都被剪枝时才剪枝这样的 cohort。
		if drs == 0 && !t.onPathFromRootToPreemptorCQ(cohort) {
			t.prunedCohorts.Insert(cohort)
		} else if drs >= highestCohortDrs {
			highestCohortDrs = drs
			highestCohort = cohort
		}
	}

	// 没有有效的候选者（即所有子 cohort 都被剪枝），
	// 因此此 cohort 被剪枝。
	if highestCohort == nil && highestCq == nil {
		t.prunedCohorts.Insert(cohort)
		return nil
	}

	// 我们使用 >= 因为，作为平局，选择 cohort 似乎
	// 稍微更公平一些，因为我们可以在该 cohort 中选择最不公平的节点。
	if highestCohortDrs >= highestCqDrs {
		return t.nextTarget(highestCohort)
	}
	return &TargetClusterQueue{
		ordering: t,
		targetCq: highestCq,
	}
}
func MakeClusterQueueOrdering(cq *cache.ClusterQueueSnapshot, candidates []*workload.Info) TargetClusterQueueOrdering {
	t := TargetClusterQueueOrdering{
		preemptorCq:        cq,
		preemptorAncestors: sets.New[*cache.CohortSnapshot](),

		clusterQueueToTarget: make(map[kueue.ClusterQueueReference][]*workload.Info),

		prunedClusterQueues: sets.New[*cache.ClusterQueueSnapshot](),
		prunedCohorts:       sets.New[*cache.CohortSnapshot](),
	}

	for ancestor := range cq.PathParentToRoot() {
		t.preemptorAncestors.Insert(ancestor)
	}

	for _, candidate := range candidates {
		t.clusterQueueToTarget[candidate.ClusterQueue] = append(t.clusterQueueToTarget[candidate.ClusterQueue], candidate)
	}

	return t
}

func (t *TargetClusterQueueOrdering) Iter() iter.Seq[*TargetClusterQueue] {
	return func(yield func(v *TargetClusterQueue) bool) {
		// 处理没有 cohort 的集群队列情况。
		if !t.preemptorCq.HasParent() {
			targetCq := &TargetClusterQueue{
				ordering: t,
				targetCq: t.preemptorCq,
			}

			for targetCq.HasWorkload() {
				if !yield(targetCq) {
					return
				}
			}
		}

		root := t.preemptorCq.Parent().Root()
		// 一旦我们标记根为已剪枝，就停止。
		for !t.prunedCohorts.Has(root) {
			targetCq := t.nextTarget(root)

			// 一个只剪枝了一些节点（或节点）的迭代。
			if targetCq == nil {
				continue
			}
			if !yield(targetCq) {
				return
			}
		}
	}
}

func (t *TargetClusterQueueOrdering) hasWorkload(cq *cache.ClusterQueueSnapshot) bool {
	return len(t.clusterQueueToTarget[cq.GetName()]) > 0
}
func (t *TargetClusterQueueOrdering) onPathFromRootToPreemptorCQ(cohort *cache.CohortSnapshot) bool {
	return t.preemptorAncestors.Has(cohort)
}
