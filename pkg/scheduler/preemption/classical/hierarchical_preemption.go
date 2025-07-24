package classical

import (
	"k8s.io/apimachinery/pkg/util/sets"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/cache"
	"sigs.k8s.io/kueue/pkg/resources"
	"sigs.k8s.io/kueue/pkg/util/priority"
	"sigs.k8s.io/kueue/pkg/workload"
)

// preemptionVariant 表示抢占的不同类型
// Never：不可被抢占
// WithinCQ：与抢占者在同一个 CQ 的候选
// HiearchicalReclaim：抢占者因 CQ 在 cohort 拓扑中的位置对资源有优先访问权
// ReclaimWithoutBorrowing：只有在抢占者 CQ（在所有抢占和新 workload 被接纳后）不借用配额时才能被抢占
// ReclaimWhileBorrowing：即使抢占者 CQ 仍在借用配额也可以被抢占
// 这些常量用于区分不同的抢占策略
type preemptionVariant int

const (
	Never                   preemptionVariant = iota // 不可被抢占
	WithinCQ                                         // 与抢占者在同一个 CQ 的候选
	HiearchicalReclaim                               // 抢占者因 CQ 在 cohort 拓扑中的位置对资源有优先访问权
	ReclaimWithoutBorrowing                          // 只有在抢占者 CQ 不借用配额时才能被抢占
	ReclaimWhileBorrowing                            // 即使抢占者 CQ 仍在借用配额也可以被抢占
)

// PreemptionReason 返回抢占原因的字符串表示
func (m preemptionVariant) PreemptionReason() string {
	switch m {
	case WithinCQ:
		return kueue.InClusterQueueReason // 在集群队列内抢占
	case HiearchicalReclaim:
		return kueue.InCohortReclamationReason // 在 cohort 层级回收
	case ReclaimWhileBorrowing:
		return kueue.InCohortReclaimWhileBorrowingReason // 在 cohort 借用时回收
	case ReclaimWithoutBorrowing:
		return kueue.InCohortReclamationReason // 在 cohort 层级回收
	}
	return "Unknown" // 未知原因
}

// HierarchicalPreemptionCtx 表示分层抢占的上下文信息
// Wl：待调度的 Workload
// Cq：当前的集群队列快照
// FrsNeedPreemption：需要抢占的资源集合
// Requests：资源请求量
// WorkloadOrdering：workload 排序方式
type HierarchicalPreemptionCtx struct {
	Wl                *kueue.Workload
	Cq                *cache.ClusterQueueSnapshot
	FrsNeedPreemption sets.Set[resources.FlavorResource]
	Requests          resources.FlavorResourceQuantities
	WorkloadOrdering  workload.Ordering
}

// IsBorrowingWithinCohortForbidden 判断 cohort 内部是否禁止借用资源，返回是否禁止和最大优先级阈值
func IsBorrowingWithinCohortForbidden(cq *cache.ClusterQueueSnapshot) (bool, *int32) {
	borrowWithinCohort := cq.Preemption.BorrowWithinCohort // 获取 cohort 内部借用策略
	if borrowWithinCohort == nil || borrowWithinCohort.Policy == kueue.BorrowWithinCohortPolicyNever {
		return true, nil // 禁止借用
	}
	return false, borrowWithinCohort.MaxPriorityThreshold // 允许借用，返回最大优先级阈值
}

// classifyPreemptionVariant 根据配置和优先级评估给定候选者的抢占类型
func classifyPreemptionVariant(ctx *HierarchicalPreemptionCtx, wl *workload.Info, haveHierarchicalAdvantage bool) preemptionVariant {
	if !WorkloadUsesResources(wl, ctx.FrsNeedPreemption) {
		return Never // 如果候选 workload 不使用需要抢占的资源，则不可被抢占
	}
	incomingPriority := priority.Priority(ctx.Wl)  // 获取新 workload 的优先级
	candidatePriority := priority.Priority(wl.Obj) // 获取候选 workload 的优先级
	if !satisfiesPreemptionPolicy(ctx, wl, incomingPriority, candidatePriority) {
		return Never // 不满足抢占策略，不可被抢占
	}
	if wl.ClusterQueue == ctx.Cq.Name {
		return WithinCQ // 在同一个 CQ 内
	}
	if haveHierarchicalAdvantage {
		return HiearchicalReclaim // 有层级优势
	}
	borrowWithinCohortForbidden, borrowWithinCohortThreshold := IsBorrowingWithinCohortForbidden(ctx.Cq)
	if borrowWithinCohortForbidden {
		return ReclaimWithoutBorrowing // 禁止借用时的回收
	}
	if isAboveBorrowingThreshold(candidatePriority, incomingPriority, borrowWithinCohortThreshold) {
		return ReclaimWithoutBorrowing // 超过借用阈值时的回收
	}
	return ReclaimWhileBorrowing // 允许借用时的回收
}

// satisfiesPreemptionPolicy 判断是否满足抢占策略
func satisfiesPreemptionPolicy(ctx *HierarchicalPreemptionCtx, wl *workload.Info, incomingPriority, candidatePriority int32) bool {
	var preemptionPolicy kueue.PreemptionPolicy // 抢占策略
	if wl.ClusterQueue == ctx.Cq.Name {
		preemptionPolicy = ctx.Cq.Preemption.WithinClusterQueue // 集群队列内策略
	} else {
		preemptionPolicy = ctx.Cq.Preemption.ReclaimWithinCohort // cohort 内回收策略
	}
	lowerPriority := incomingPriority > candidatePriority // 新 workload 优先级是否更高
	if preemptionPolicy == kueue.PreemptionPolicyLowerPriority {
		return lowerPriority // 只允许抢占优先级更低的
	}
	if preemptionPolicy == kueue.PreemptionPolicyLowerOrNewerEqualPriority {
		preemptorTS := ctx.WorkloadOrdering.GetQueueOrderTimestamp(ctx.Wl)                                                                       // 获取新 workload 的队列时间戳
		newerEqualPriority := (incomingPriority == candidatePriority) && preemptorTS.Before(ctx.WorkloadOrdering.GetQueueOrderTimestamp(wl.Obj)) // 优先级相同但新 workload 更早
		return (lowerPriority || newerEqualPriority)                                                                                             // 满足更低优先级或更早的同优先级
	}
	return preemptionPolicy == kueue.PreemptionPolicyAny // 允许任意抢占
}

// isAboveBorrowingThreshold 判断候选 workload 是否超过借用阈值
func isAboveBorrowingThreshold(candidatePriority, incomingPriority int32, borrowWithinCohortThreshold *int32) bool {
	if candidatePriority >= incomingPriority {
		return true // 候选优先级高于等于新 workload
	}
	if borrowWithinCohortThreshold == nil {
		return false // 没有阈值
	}
	return candidatePriority > *borrowWithinCohortThreshold // 超过阈值
}

// collectSameQueueCandidates 收集同一 CQ 内可被抢占的候选 workload
func collectSameQueueCandidates(ctx *HierarchicalPreemptionCtx) []*candidateElem {
	if ctx.Cq.Preemption.WithinClusterQueue == kueue.PreemptionPolicyNever {
		return []*candidateElem{} // 不允许抢占
	}
	return getCandidatesFromCQ(ctx.Cq, nil, ctx, false) // 获取候选
}

// getCandidatesFromCQ 从指定 CQ 获取可被抢占的候选 workload
func getCandidatesFromCQ(cq *cache.ClusterQueueSnapshot, lca *cache.CohortSnapshot, ctx *HierarchicalPreemptionCtx, hasHiearchicalAdvantage bool) []*candidateElem {
	candidates := []*candidateElem{} // 候选列表
	for _, candidateWl := range cq.Workloads {
		preemptionVariant := classifyPreemptionVariant(ctx, candidateWl, hasHiearchicalAdvantage) // 判断抢占类型
		if preemptionVariant == Never {
			continue // 不可被抢占
		}
		candidates = append(candidates,
			&candidateElem{
				wl:                candidateWl,       // 候选 workload
				lca:               lca,               // 最低公共祖先
				preemptionVariant: preemptionVariant, // 抢占类型
			})
	}
	return candidates // 返回候选列表
}

// collectCandidatesForHierarchicalReclaim 收集分层回收的候选 workload
func collectCandidatesForHierarchicalReclaim(ctx *HierarchicalPreemptionCtx) ([]*candidateElem, []*candidateElem) {
	hierarchyCandidates := []*candidateElem{} // 层级候选
	priorityCandidates := []*candidateElem{}  // 优先级候选
	if !ctx.Cq.HasParent() || ctx.Cq.Preemption.ReclaimWithinCohort == kueue.PreemptionPolicyNever {
		return hierarchyCandidates, priorityCandidates // 没有父节点或不允许回收
	}
	var previousSubtreeRoot *cache.CohortSnapshot                                                   // 上一个子树根
	var candidateList *[]*candidateElem                                                             // 当前候选列表
	var fits bool                                                                                   // 是否适配
	hasHierarchicalAdvantage, remainingRequests := cache.QuantitiesFitInQuota(ctx.Cq, ctx.Requests) // 判断是否有层级优势及剩余请求
	for currentSubtreeRoot := range ctx.Cq.PathParentToRoot() {
		if hasHierarchicalAdvantage {
			candidateList = &hierarchyCandidates // 有层级优势
		} else {
			candidateList = &priorityCandidates // 无层级优势
		}
		collectCandidatesInSubtree(ctx, currentSubtreeRoot, currentSubtreeRoot, previousSubtreeRoot, hasHierarchicalAdvantage, candidateList) // 收集子树候选
		fits, remainingRequests = cache.QuantitiesFitInQuota(currentSubtreeRoot, remainingRequests)                                           // 判断当前子树是否适配
		// 一旦找到适配请求的子树，将查找使用该子树配额的 workload。
		// 抢占者对所有这些 workload 都有层级优势，因为它属于该子树。
		// 因此 hasHierarchicalAdvantage 在后续循环中保持 true。
		hasHierarchicalAdvantage = hasHierarchicalAdvantage || fits
		previousSubtreeRoot = currentSubtreeRoot // 更新上一个子树根
	}
	return hierarchyCandidates, priorityCandidates // 返回候选
}

// collectCandidatesInSubtree 遍历层级节点，收集超出配额的 workload，跳过配额内和被跳过的子树
func collectCandidatesInSubtree(ctx *HierarchicalPreemptionCtx, currentCohort *cache.CohortSnapshot, subtreeRoot *cache.CohortSnapshot, skipSubtree *cache.CohortSnapshot, hasHierarchicalAdvantage bool, result *[]*candidateElem) {
	for _, childCohort := range currentCohort.ChildCohorts() {
		if childCohort == skipSubtree {
			continue // 已处理过该子树
		}
		if cache.IsWithinNominalInResources(childCohort, ctx.FrsNeedPreemption) {
			continue // 子树配额内，跳过
		}
		collectCandidatesInSubtree(ctx, childCohort, subtreeRoot, skipSubtree, hasHierarchicalAdvantage, result) // 递归收集
	}
	for _, childCq := range currentCohort.ChildCQs() {
		if childCq == ctx.Cq {
			continue // 跳过自身
		}
		if !cache.IsWithinNominalInResources(childCq, ctx.FrsNeedPreemption) {
			*result = append(*result, getCandidatesFromCQ(childCq, subtreeRoot, ctx, hasHierarchicalAdvantage)...) // 收集子 CQ 的候选
		}
	}
}

// FindHeightOfLowestSubtreeThatFits 返回一个最低子树的高度，该子树在 cohort 中适合额外的 val 资源 fr。
// 如果没有这样的子树，则返回整个 cohort 层次结构的高度。注意，高度为 0 的平凡子树只有一个节点。
// 它还返回该子树是否小于整个 cohort 树。
func FindHeightOfLowestSubtreeThatFits(c *cache.ClusterQueueSnapshot, fr resources.FlavorResource, val int64) (int, bool) {
	if !c.BorrowingWith(fr, val) || !c.HasParent() {
		return 0, c.HasParent() // 不需要借用或无父节点，返回 0
	}
	remaining := val - cache.LocalAvailable(c, fr) // 计算剩余需求
	for trackingNode := range c.PathParentToRoot() {
		if !trackingNode.BorrowingWith(fr, remaining) {
			return getNodeHeight(trackingNode), trackingNode.HasParent() // 找到合适子树，返回高度
		}
		remaining -= cache.LocalAvailable(trackingNode, fr) // 递减剩余需求
	}
	// 未找到合适子树，返回整个 cohort 树高度
	return getNodeHeight(c.Parent().Root()), false
}

// getNodeHeight 计算到最远叶子节点的距离
func getNodeHeight(node *cache.CohortSnapshot) int {
	maxHeight := min(node.ChildCount(), 1) // 取子节点数和 1 的最小值
	for _, childCohort := range node.ChildCohorts() {
		maxHeight = max(maxHeight, getNodeHeight(childCohort)+1) // 递归计算最大高度
	}
	return maxHeight // 返回最大高度
}
