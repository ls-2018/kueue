package classical

import (
	"k8s.io/apimachinery/pkg/util/sets"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/cache"
	"sigs.k8s.io/kueue/pkg/resources"
	"sigs.k8s.io/kueue/pkg/util/priority"
	"sigs.k8s.io/kueue/pkg/workload"
)

type preemptionVariant int

const (
	// 不能被抢占
	Never preemptionVariant = iota
	// 与抢占者在同一个 CQ 的候选项
	WithinCQ
	// 抢占者因其在 cohort 拓扑中的 CQ 位置，对需要抢占资源有优先访问权
	HiearchicalReclaim
	// 只有当抢占者 CQ（在所有抢占和新工作负载准入后）不再借用任何配额时，才可被抢占
	ReclaimWithoutBorrowing
	// 即使抢占者 CQ 仍在借用，也可以被抢占
	ReclaimWhileBorrowing
)

func (m preemptionVariant) PreemptionReason() string {
	switch m {
	case WithinCQ:
		return kueue.InClusterQueueReason
	case HiearchicalReclaim:
		return kueue.InCohortReclamationReason
	case ReclaimWhileBorrowing:
		return kueue.InCohortReclaimWhileBorrowingReason
	case ReclaimWithoutBorrowing:
		return kueue.InCohortReclamationReason
	}
	return "Unknown"
}

type HierarchicalPreemptionCtx struct {
	Wl                *kueue.Workload
	Cq                *cache.ClusterQueueSnapshot
	FrsNeedPreemption sets.Set[resources.FlavorResource]
	Requests          resources.FlavorResourceQuantities
	WorkloadOrdering  workload.Ordering
}

func IsBorrowingWithinCohortForbidden(cq *cache.ClusterQueueSnapshot) (bool, *int32) {
	borrowWithinCohort := cq.Preemption.BorrowWithinCohort
	if borrowWithinCohort == nil || borrowWithinCohort.Policy == kueue.BorrowWithinCohortPolicyNever {
		return true, nil
	}
	return false, borrowWithinCohort.MaxPriorityThreshold
}

// classifyPreemptionVariant 根据配置和优先级，评估给定候选项的抢占类型
func classifyPreemptionVariant(ctx *HierarchicalPreemptionCtx, wl *workload.Info, haveHierarchicalAdvantage bool) preemptionVariant {
	if !WorkloadUsesResources(wl, ctx.FrsNeedPreemption) {
		return Never
	}
	incomingPriority := priority.Priority(ctx.Wl)
	candidatePriority := priority.Priority(wl.Obj)
	if !satisfiesPreemptionPolicy(ctx, wl, incomingPriority, candidatePriority) {
		return Never
	}
	if wl.ClusterQueue == ctx.Cq.Name {
		return WithinCQ
	}
	if haveHierarchicalAdvantage {
		return HiearchicalReclaim
	}
	borrowWithinCohortForbidden, borrowWithinCohortThreshold := IsBorrowingWithinCohortForbidden(ctx.Cq)
	if borrowWithinCohortForbidden {
		return ReclaimWithoutBorrowing
	}
	if isAboveBorrowingThreshold(candidatePriority, incomingPriority, borrowWithinCohortThreshold) {
		return ReclaimWithoutBorrowing
	}
	return ReclaimWhileBorrowing
}

func satisfiesPreemptionPolicy(ctx *HierarchicalPreemptionCtx, wl *workload.Info, incomingPriority, candidatePriority int32) bool {
	var preemptionPolicy kueue.PreemptionPolicy
	if wl.ClusterQueue == ctx.Cq.Name {
		preemptionPolicy = ctx.Cq.Preemption.WithinClusterQueue
	} else {
		preemptionPolicy = ctx.Cq.Preemption.ReclaimWithinCohort
	}
	lowerPriority := incomingPriority > candidatePriority
	if preemptionPolicy == kueue.PreemptionPolicyLowerPriority {
		return lowerPriority
	}
	if preemptionPolicy == kueue.PreemptionPolicyLowerOrNewerEqualPriority {
		preemptorTS := ctx.WorkloadOrdering.GetQueueOrderTimestamp(ctx.Wl)
		newerEqualPriority := (incomingPriority == candidatePriority) && preemptorTS.Before(ctx.WorkloadOrdering.GetQueueOrderTimestamp(wl.Obj))
		return (lowerPriority || newerEqualPriority)
	}
	return preemptionPolicy == kueue.PreemptionPolicyAny
}

func isAboveBorrowingThreshold(candidatePriority, incomingPriority int32, borrowWithinCohortThreshold *int32) bool {
	if candidatePriority >= incomingPriority {
		return true
	}
	if borrowWithinCohortThreshold == nil {
		return false
	}
	return candidatePriority > *borrowWithinCohortThreshold
}

func collectSameQueueCandidates(ctx *HierarchicalPreemptionCtx) []*candidateElem {
	if ctx.Cq.Preemption.WithinClusterQueue == kueue.PreemptionPolicyNever {
		return []*candidateElem{}
	}
	return getCandidatesFromCQ(ctx.Cq, nil, ctx, false)
}

func getCandidatesFromCQ(cq *cache.ClusterQueueSnapshot, lca *cache.CohortSnapshot, ctx *HierarchicalPreemptionCtx, hasHiearchicalAdvantage bool) []*candidateElem {
	candidates := []*candidateElem{}
	for _, candidateWl := range cq.Workloads {
		preemptionVariant := classifyPreemptionVariant(ctx, candidateWl, hasHiearchicalAdvantage)
		if preemptionVariant == Never {
			continue
		}
		candidates = append(candidates,
			&candidateElem{
				wl:                candidateWl,
				lca:               lca,
				preemptionVariant: preemptionVariant,
			})
	}
	return candidates
}

func collectCandidatesForHierarchicalReclaim(ctx *HierarchicalPreemptionCtx) ([]*candidateElem, []*candidateElem) {
	hierarchyCandidates := []*candidateElem{}
	priorityCandidates := []*candidateElem{}
	if !ctx.Cq.HasParent() || ctx.Cq.Preemption.ReclaimWithinCohort == kueue.PreemptionPolicyNever {
		return hierarchyCandidates, priorityCandidates
	}
	var previousSubtreeRoot *cache.CohortSnapshot
	var candidateList *[]*candidateElem
	var fits bool
	hasHierarchicalAdvantage, remainingRequests := cache.QuantitiesFitInQuota(ctx.Cq, ctx.Requests)
	for currentSubtreeRoot := range ctx.Cq.PathParentToRoot() {
		if hasHierarchicalAdvantage {
			candidateList = &hierarchyCandidates
		} else {
			candidateList = &priorityCandidates
		}
		collectCandidatesInSubtree(ctx, currentSubtreeRoot, currentSubtreeRoot, previousSubtreeRoot, hasHierarchicalAdvantage, candidateList)
		fits, remainingRequests = cache.QuantitiesFitInQuota(currentSubtreeRoot, remainingRequests)
		// Once we find a subtree sT that fits the requests, we will look for workloads that use quota
		// of that subtree. The preemptor will have hierarchical advantage over all such workloads
		// because it belongs to subtree sT. For that reason variable hasHierarchicalAdvantage
		// remains true in subsequent iterations of the loop.
		hasHierarchicalAdvantage = hasHierarchicalAdvantage || fits
		previousSubtreeRoot = currentSubtreeRoot
	}
	return hierarchyCandidates, priorityCandidates
}

// visit the nodes in the hierarchy and collect the ones that exceed quota
// avoid subtrees that are within quota and the skipped subtree
// 遍历层级中的节点，收集超出配额的节点，跳过在配额内和被跳过的子树
func collectCandidatesInSubtree(ctx *HierarchicalPreemptionCtx, currentCohort *cache.CohortSnapshot, subtreeRoot *cache.CohortSnapshot, skipSubtree *cache.CohortSnapshot, hasHierarchicalAdvantage bool, result *[]*candidateElem) {
	for _, childCohort := range currentCohort.ChildCohorts() {
		// we already processed this subtree
		if childCohort == skipSubtree {
			continue
		}
		// don't look for candidates in subtrees that are not exceeding their quotas
		if cache.IsWithinNominalInResources(childCohort, ctx.FrsNeedPreemption) {
			continue
		}
		collectCandidatesInSubtree(ctx, childCohort, subtreeRoot, skipSubtree, hasHierarchicalAdvantage, result)
	}
	for _, childCq := range currentCohort.ChildCQs() {
		if childCq == ctx.Cq {
			continue
		}
		if !cache.IsWithinNominalInResources(childCq, ctx.FrsNeedPreemption) {
			*result = append(*result, getCandidatesFromCQ(childCq, subtreeRoot, ctx, hasHierarchicalAdvantage)...)
		}
	}
}

// getNodeHeight 计算到最远叶子的距离
func getNodeHeight(node *cache.CohortSnapshot) int {
	maxHeight := min(node.ChildCount(), 1)
	for _, childCohort := range node.ChildCohorts() {
		maxHeight = max(maxHeight, getNodeHeight(childCohort)+1)
	}
	return maxHeight
}

// FindHeightOfLowestSubtreeThatFits 返回 cohort 中能容纳额外资源 val 的最低子树高度。
// 如果不存在这样的子树，则返回整个 cohort 层级的高度。
// 注意，只有一个节点的平凡子树高度为 0。还会返回该子树是否小于整个 cohort 树。
func FindHeightOfLowestSubtreeThatFits(c *cache.ClusterQueueSnapshot, fr resources.FlavorResource, val int64) (int, bool) {
	if !c.BorrowingWith(fr, val) || !c.HasParent() {
		return 0, c.HasParent()
	}
	remaining := val - cache.LocalAvailable(c, fr)
	for trackingNode := range c.PathParentToRoot() {
		if !trackingNode.BorrowingWith(fr, remaining) {
			return getNodeHeight(trackingNode), trackingNode.HasParent()
		}
		remaining -= cache.LocalAvailable(trackingNode, fr)
	}
	// no fit found
	return getNodeHeight(c.Parent().Root()), false
}
