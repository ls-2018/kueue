package scheduler

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/cache"
	"sigs.k8s.io/kueue/pkg/features"
	"sigs.k8s.io/kueue/pkg/util/priority"
	"sigs.k8s.io/kueue/pkg/workload"
)

// fairSharingIterator 在启用 FairSharing 时，以“公平”的方式对候选项进行排序以供调度考虑。算法描述见 runTournament。
type fairSharingIterator struct {
	cqToEntry     map[*cache.ClusterQueueSnapshot]*entry
	entryComparer entryComparer
	log           logr.Logger
}

func makeFairSharingIterator(ctx context.Context, entries []entry, workloadOrdering workload.Ordering) *fairSharingIterator {
	f := fairSharingIterator{
		cqToEntry: make(map[*cache.ClusterQueueSnapshot]*entry, len(entries)),
		entryComparer: entryComparer{
			workloadOrdering: workloadOrdering,
		},
		log: ctrl.LoggerFrom(ctx),
	}
	for i := range entries {
		f.cqToEntry[entries[i].clusterQueueSnapshot] = &entries[i]
	}
	return &f
}

// getCq 返回一个有待调度工作负载的 CQ。该函数是非确定性的。我们不保证不同 Cohort 间工作负载的调度顺序。Cohort 内的工作负载调度顺序几乎是确定性的（仅当 DRS、优先级和时间都相等时才非确定性）。
func (f *fairSharingIterator) getCq() *cache.ClusterQueueSnapshot {
	for cq := range f.cqToEntry {
		return cq
	}
	return nil
}

func (f *fairSharingIterator) pop() *entry {
	cq := f.getCq()

	// CQ 没有 Cohort。我们直接返回其工作负载。
	if !cq.HasParent() {
		entry := f.cqToEntry[cq]
		f.log.V(3).Info("Returning workload from ClusterQueue without Cohort",
			"clusterQueue", klog.KRef("", string(cq.GetName())),
			"workload", klog.KObj(entry.Obj))
		delete(f.cqToEntry, cq)
		return entry
	}

	// CQ 属于某个 Cohort。我们运行锦标赛算法，在每一层选择最公平的工作负载。
	root := cq.Parent().Root()
	log := f.log.WithValues("rootCohort", klog.KRef("", string(root.GetName())))

	log.V(5).Info("Computing DominantResourceShare for tournament")
	f.entryComparer.computeDRS(root, f.cqToEntry)

	log.V(3).Info("Running tournament to decide next workload to consider in scheduling cycle")
	entry := runTournament(root, f.entryComparer, f.cqToEntry)

	log = log.WithValues(
		"cohort", klog.KRef("", string(entry.clusterQueueSnapshot.Parent().GetName())),
		"clusterQueue", klog.KRef("", string(entry.clusterQueueSnapshot.GetName())),
		"winningWorkload", klog.KObj(entry.Obj))

	log.V(3).Info("Determined tournament winner")
	f.entryComparer.logDrsValuesWhenVerbose(log)

	delete(f.cqToEntry, entry.clusterQueueSnapshot)
	return entry
}

// runTournament 是一个递归算法，为每个 Cohort 提名一个工作负载。它比较每个 Cohort 子节点（CQ 或 Cohort）的 DominantResourceShare（DRS）值，包含该子节点所提名的工作负载。DRS 最低的节点获胜，存在额外的平局判定（见 entryComparer.less）。
//
// 该过程会为每个节点逐级选出一个工作负载（如果 Cohort 本轮没有剩余工作负载则为零），直到根节点只剩下一个工作负载。
func runTournament(cohort *cache.CohortSnapshot, ec entryComparer, cqToEntry map[*cache.ClusterQueueSnapshot]*entry) *entry {
	candidates := make([]*entry, 0, cohort.ChildCount())

	// 对每个子 Cohort 递归运行算法。
	for _, childCohort := range cohort.ChildCohorts() {
		// 当子 Cohort 本轮没有剩余工作负载时，tournament 返回 0 节点。
		if candidate := runTournament(childCohort, ec, cqToEntry); candidate != nil {
			candidates = append(candidates, candidate)
		}
	}

	// 收集 CQ 的条目。如果条目已在上一次 pop 调用中返回，则不会在 cqToEntry 映射中。
	for _, childCq := range cohort.ChildCQs() {
		if candidate, ok := cqToEntry[childCq]; ok {
			candidates = append(candidates, candidate)
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// 比较每个子节点的工作负载的 DRS。
	best := candidates[0]
	for _, current := range candidates[1:] {
		if ec.less(current, best, cohort.GetName()) {
			best = current
		}
	}
	return best
}

type drsKey struct {
	parentCohort kueue.CohortReference
	workloadKey  string
}

type entryComparer struct {
	drsValues        map[drsKey]int
	workloadOrdering workload.Ordering
}

func (e *entryComparer) less(a, b *entry, parentCohort kueue.CohortReference) bool {
	aDrs := e.drsValues[drsKey{parentCohort: parentCohort, workloadKey: workload.Key(a.Obj)}]
	bDrs := e.drsValues[drsKey{parentCohort: parentCohort, workloadKey: workload.Key(b.Obj)}]
	// 1: DRF
	if aDrs != bDrs {
		return aDrs < bDrs
	}

	// 2: 优先级
	if features.Enabled(features.PrioritySortingWithinCohort) {
		p1 := priority.Priority(a.Obj)
		p2 := priority.Priority(b.Obj)
		if p1 != p2 {
			return p1 > p2
		}
	}

	// 3: FIFO
	aComparisonTimestamp := e.workloadOrdering.GetQueueOrderTimestamp(a.Obj)
	bComparisonTimestamp := e.workloadOrdering.GetQueueOrderTimestamp(b.Obj)
	return aComparisonTimestamp.Before(bComparisonTimestamp)
}

// computeDRS 计算从 CQ 到 root-1 路径上每个节点在工作负载被接纳后的 DominantResourceShare（DRS）。在锦标赛过程中，这些值用于比较 parentCohort 的所有子节点，以选择在接纳其提名工作负载后 DRS 最低的子节点。
func (e *entryComparer) computeDRS(rootCohort *cache.CohortSnapshot, cqToEntry map[*cache.ClusterQueueSnapshot]*entry) {
	e.drsValues = make(map[drsKey]int)
	for _, cq := range rootCohort.SubtreeClusterQueues() {
		entry, ok := cqToEntry[cq]
		if !ok {
			continue
		}
		// 将工作负载的用量加到 CQ 上，使后续 DRS 计算都包含该工作负载的接纳。
		revert := cq.SimulateUsageAddition(entry.assignmentUsage())

		// 计算 CQ 在包含工作负载后的 DRS。
		dominantResourceShare := cq.DominantResourceShare()

		// 计算路径上所有 Cohort 在包含工作负载后的 DRS。
		for ancestor := range cq.PathParentToRoot() {
			e.drsValues[drsKey{parentCohort: ancestor.GetName(), workloadKey: workload.Key(entry.Obj)}] = dominantResourceShare
			dominantResourceShare = ancestor.DominantResourceShare()
		}

		revert()
	}
}

func (e *entryComparer) logDrsValuesWhenVerbose(log logr.Logger) {
	if logV := log.V(5); logV.Enabled() {
		serializableDrs := make([]string, 0, len(e.drsValues))
		for k, v := range e.drsValues {
			serializableDrs = append(serializableDrs, fmt.Sprintf("{parentCohort: %s, workload %s, drs: %d}", k.parentCohort, k.workloadKey, v))
		}
		logV.Info("DominantResourceShare values used during tournament", "drsValues", serializableDrs)
	}
}
func (f *fairSharingIterator) hasNext() bool {
	return len(f.cqToEntry) > 0
}
