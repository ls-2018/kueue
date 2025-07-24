package classical

import (
	"sort"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/clock"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/cache"
	"sigs.k8s.io/kueue/pkg/resources"
	"sigs.k8s.io/kueue/pkg/workload"
)

// candidateIterator 用于遍历抢占候选 workload 的迭代器
// candidates：候选元素列表
// runIndex：当前遍历索引
// frsNeedPreemption：需要抢占的资源集合
// snapshot：集群快照
// NoCandidateFromOtherQueues：是否没有来自其他队列的候选
// NoCandidateForHierarchicalReclaim：是否没有分层回收候选
// hierarchicalReclaimCtx：分层抢占上下文
type candidateIterator struct {
	candidates                        []*candidateElem                   // 候选 workload 元素列表
	runIndex                          int                                // 当前遍历到的索引
	frsNeedPreemption                 sets.Set[resources.FlavorResource] // 需要抢占的资源集合
	snapshot                          *cache.Snapshot                    // 集群快照
	NoCandidateFromOtherQueues        bool                               // 是否没有来自其他队列的候选
	NoCandidateForHierarchicalReclaim bool                               // 是否没有分层回收候选
	hierarchicalReclaimCtx            *HierarchicalPreemptionCtx         // 分层抢占上下文
}

// candidateElem 表示一个候选 workload 及其抢占变体
// wl：workload 信息
// lca：当前 workload 和候选队列（cq）的最近公共祖先（LCA）快照
// preemptionVariant：抢占变体，表示抢占方式（如抢占、不抢占）
type candidateElem struct {
	wl                *workload.Info        // 候选 workload 信息
	lca               *cache.CohortSnapshot // 当前队列和目标队列的最近公共祖先
	preemptionVariant preemptionVariant     // 抢占类型
}

// candidateElemsOrdering 用于对候选 workload 进行排序
// 在将来，我们将修改排序以考虑在 cohort 层级树中的距离
func candidateElemsOrdering(candidates []*candidateElem, cq kueue.ClusterQueueReference, now time.Time, ordering func([]*workload.Info, kueue.ClusterQueueReference, time.Time) func(int, int) bool) func(int, int) bool {
	// 适配 candidatesOrdering 函数以与 candidateElem 一起工作
	adaptedOrdering := func(i, j int) bool {
		a := candidates[i].wl                                  // 第 i 个候选 workload
		b := candidates[j].wl                                  // 第 j 个候选 workload
		return ordering([]*workload.Info{a, b}, cq, now)(0, 1) // 调用排序函数
	}
	return adaptedOrdering // 返回适配后的排序函数
}

// splitEvicted 将 workload 列表按 WorkloadEvicted 条件拆分为已驱逐和未驱逐的 workload
func splitEvicted(workloads []*candidateElem) ([]*candidateElem, []*candidateElem) {
	firstFalse := sort.Search(len(workloads), func(i int) bool {
		return !meta.IsStatusConditionTrue(workloads[i].wl.Obj.Status.Conditions, kueue.WorkloadEvicted) // 判断 workload 是否被驱逐
	})
	return workloads[:firstFalse], workloads[firstFalse:] // 返回已驱逐和未驱逐的 workload 列表
}

// NewCandidateIterator 创建一个新的迭代器，用于生成抢占候选 workload
// 该迭代器可以用于执行两个独立的运行：
// 1. 不借用资源
// 2. 借用资源
// 这两个运行是独立的，这意味着相同的候选 workload 可能会在两个运行中返回，
// 但请注意，借用资源的候选 workload 是未借用资源的候选 workload 的子集。
func NewCandidateIterator(hierarchicalReclaimCtx *HierarchicalPreemptionCtx, frsNeedPreemption sets.Set[resources.FlavorResource], snapshot *cache.Snapshot, clock clock.Clock, ordering func([]*workload.Info, kueue.ClusterQueueReference, time.Time) func(int, int) bool) *candidateIterator {
	sameQueueCandidates := collectSameQueueCandidates(hierarchicalReclaimCtx)                                                           // 收集同队列候选
	hierarchyCandidates, priorityCandidates := collectCandidatesForHierarchicalReclaim(hierarchicalReclaimCtx)                          // 收集分层回收候选
	sort.Slice(sameQueueCandidates, candidateElemsOrdering(sameQueueCandidates, hierarchicalReclaimCtx.Cq.Name, clock.Now(), ordering)) // 对同队列候选排序
	sort.Slice(priorityCandidates, candidateElemsOrdering(priorityCandidates, hierarchicalReclaimCtx.Cq.Name, clock.Now(), ordering))   // 对优先级候选排序
	sort.Slice(hierarchyCandidates, candidateElemsOrdering(hierarchyCandidates, hierarchicalReclaimCtx.Cq.Name, clock.Now(), ordering)) // 对分层候选排序
	evictedHierarchicalReclaimCandidates, nonEvictedHierarchicalReclaimCandidates := splitEvicted(hierarchyCandidates)                  // 拆分分层回收候选
	evictedSTCandidates, nonEvictedSTCandidates := splitEvicted(priorityCandidates)                                                     // 拆分优先级候选
	evictedSameQueueCandidates, nonEvictedSameQueueCandidates := splitEvicted(sameQueueCandidates)                                      // 拆分同队列候选
	allCandidates := make([]*candidateElem, 0, len(hierarchyCandidates)+len(priorityCandidates)+len(sameQueueCandidates))               // 初始化所有候选列表
	allCandidates = append(allCandidates, evictedHierarchicalReclaimCandidates...)                                                      // 添加已驱逐分层回收候选
	allCandidates = append(allCandidates, evictedSTCandidates...)                                                                       // 添加已驱逐优先级候选
	allCandidates = append(allCandidates, evictedSameQueueCandidates...)                                                                // 添加已驱逐同队列候选
	allCandidates = append(allCandidates, nonEvictedHierarchicalReclaimCandidates...)                                                   // 添加未驱逐分层回收候选
	allCandidates = append(allCandidates, nonEvictedSTCandidates...)                                                                    // 添加未驱逐优先级候选
	allCandidates = append(allCandidates, nonEvictedSameQueueCandidates...)                                                             // 添加未驱逐同队列候选
	return &candidateIterator{
		runIndex:                          0,                                                             // 初始化索引
		frsNeedPreemption:                 frsNeedPreemption,                                             // 需要抢占的资源
		snapshot:                          snapshot,                                                      // 集群快照
		candidates:                        allCandidates,                                                 // 所有候选
		NoCandidateFromOtherQueues:        len(hierarchyCandidates) == 0 && len(priorityCandidates) == 0, // 是否没有其他队列候选
		NoCandidateForHierarchicalReclaim: len(hierarchyCandidates) == 0,                                 // 是否没有分层回收候选
		hierarchicalReclaimCtx:            hierarchicalReclaimCtx,                                        // 分层抢占上下文
	}
}

// Next 允许迭代遍历排序后的候选 workload，并返回候选 workload 的驱逐原因
func (c *candidateIterator) Next(borrow bool) (*workload.Info, string) {
	if c.runIndex >= len(c.candidates) { // 如果遍历完所有候选
		return nil, "" // 返回空
	}
	candidate := c.candidates[c.runIndex]       // 获取当前候选
	c.runIndex++                                // 索引加一
	if !c.candidateIsValid(candidate, borrow) { // 检查候选是否有效
		return c.Next(borrow) // 无效则递归查找下一个
	}
	return candidate.wl, candidate.preemptionVariant.PreemptionReason() // 返回候选 workload 及其抢占原因
}

// candidateIsValid 检查候选 workload 是否有效
// 例如，某些候选 workload 只能在不借用资源的情况下考虑
// 抢占候选 workload 可能会使其他候选 workload 失效
func (c *candidateIterator) candidateIsValid(candidate *candidateElem, borrow bool) bool {
	if c.hierarchicalReclaimCtx.Cq.Name == candidate.wl.ClusterQueue { // 如果候选和当前队列相同
		return true // 有效
	}
	if borrow && candidate.preemptionVariant == ReclaimWithoutBorrowing { // 借用资源时不能抢占不允许借用的候选
		return false // 无效
	}
	cq := c.snapshot.ClusterQueue(candidate.wl.ClusterQueue)       // 获取候选所在的集群队列
	if cache.IsWithinNominalInResources(cq, c.frsNeedPreemption) { // 如果资源在配额内
		return false // 无效
	}
	// 我们不遍历到根节点，只遍历到 lca 节点
	for node := range cq.PathParentToRoot() { // 遍历从当前队列到根的路径
		if node == candidate.lca { // 到达最近公共祖先
			break // 停止遍历
		}
		if cache.IsWithinNominalInResources(node, c.frsNeedPreemption) { // 如果节点资源在配额内
			return false // 无效
		}
	}
	return true // 有效
}

// Reset 将候选迭代器重置回起始位置
// 在每次运行之前都需要重置迭代器
func (c *candidateIterator) Reset() {
	c.runIndex = 0 // 重置索引
}

// WorkloadUsesResources 检查 workload 是否使用了需要抢占的资源
func WorkloadUsesResources(wl *workload.Info, frsNeedPreemption sets.Set[resources.FlavorResource]) bool {
	for _, ps := range wl.TotalRequests { // 遍历 workload 的所有资源请求
		for res, flv := range ps.Flavors { // 遍历每个资源和其 flavor
			if frsNeedPreemption.Has(resources.FlavorResource{Flavor: flv, Resource: res}) { // 判断是否需要抢占
				return true // 需要抢占
			}
		}
	}
	return false // 不需要抢占
}
