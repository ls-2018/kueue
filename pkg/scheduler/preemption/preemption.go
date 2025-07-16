package preemption

import (
	"context"
	"fmt"
	"sort"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	config "sigs.k8s.io/kueue/apis/config/v1beta1"
	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/cache"
	"sigs.k8s.io/kueue/pkg/controller/over_constants"
	"sigs.k8s.io/kueue/pkg/features"
	"sigs.k8s.io/kueue/pkg/metrics"
	"sigs.k8s.io/kueue/pkg/resources"
	"sigs.k8s.io/kueue/pkg/scheduler/flavorassigner"
	"sigs.k8s.io/kueue/pkg/scheduler/preemption/classical"
	"sigs.k8s.io/kueue/pkg/scheduler/preemption/fairsharing"
	"sigs.k8s.io/kueue/pkg/util/priority"
	"sigs.k8s.io/kueue/pkg/util/routine"
	"sigs.k8s.io/kueue/pkg/workload"
)

const parallelPreemptions = 8

type Preemptor struct {
	clock clock.Clock

	client   client.Client
	recorder record.EventRecorder

	workloadOrdering  workload.Ordering
	enableFairSharing bool
	fsStrategies      []fairsharing.Strategy

	// stubs
	applyPreemption func(ctx context.Context, w *kueue.Workload, reason, message string) error
}

type preemptionCtx struct {
	log               logr.Logger
	preemptor         workload.Info
	preemptorCQ       *cache.ClusterQueueSnapshot
	snapshot          *cache.Snapshot
	workloadUsage     workload.Usage
	tasRequests       cache.WorkloadTASRequests
	frsNeedPreemption sets.Set[resources.FlavorResource]
}

func New(
	cl client.Client,
	workloadOrdering workload.Ordering,
	recorder record.EventRecorder,
	fs config.FairSharing,
	clock clock.Clock,
) *Preemptor {
	p := &Preemptor{
		clock:             clock,
		client:            cl,
		recorder:          recorder,
		workloadOrdering:  workloadOrdering,
		enableFairSharing: fs.Enable,
		fsStrategies:      parseStrategies(fs.PreemptionStrategies),
	}
	p.applyPreemption = p.applyPreemptionWithSSA
	return p
}

func (p *Preemptor) OverrideApply(f func(context.Context, *kueue.Workload, string, string) error) {
	p.applyPreemption = f
}

type Target struct {
	WorkloadInfo *workload.Info
	Reason       string
}

// GetTargets 返回为给 wl 腾出空间需要驱逐的工作负载列表。
func (p *Preemptor) GetTargets(log logr.Logger, wl workload.Info, assignment flavorassigner.Assignment, snapshot *cache.Snapshot) []*Target {
	cq := snapshot.ClusterQueue(wl.ClusterQueue)
	tasRequests := assignment.WorkloadsTopologyRequests(&wl, cq)
	return p.getTargets(&preemptionCtx{
		log:               log,
		preemptor:         wl,
		preemptorCQ:       cq,
		snapshot:          snapshot,
		tasRequests:       tasRequests,
		frsNeedPreemption: flavorResourcesNeedPreemption(assignment),
		workloadUsage: workload.Usage{
			Quota: assignment.TotalRequestsFor(&wl),
			TAS:   wl.TASUsage(),
		},
	})
}

func (p *Preemptor) getTargets(preemptionCtx *preemptionCtx) []*Target {
	if p.enableFairSharing {
		return p.fairPreemptions(preemptionCtx, p.fsStrategies)
	}
	return p.classicalPreemptions(preemptionCtx)
}

var HumanReadablePreemptionReasons = map[string]string{
	kueue.InClusterQueueReason:                "优先级在集群队列中",
	kueue.InCohortReclamationReason:           "在 cohort 中回收",
	kueue.InCohortFairSharingReason:           "在 cohort 中公平共享",
	kueue.InCohortReclaimWhileBorrowingReason: "在 cohort 中借用时回收",
	"": "未知",
}

func preemptionMessage(preemptor *kueue.Workload, reason string) string {
	var wUID, jUID string
	if preemptor == nil || preemptor.UID == "" {
		wUID = "未知"
	} else {
		wUID = string(preemptor.UID)
	}
	uid, ok := preemptor.Labels[over_constants.JobUIDLabel]
	if !ok || uid == "" {
		jUID = "未知"
	} else {
		jUID = uid
	}

	return fmt.Sprintf("抢占以适应工作负载 (UID: %s, JobUID: %s) 由于 %s", wUID, jUID, HumanReadablePreemptionReasons[reason])
}

// IssuePreemptions 将目标工作负载标记为已驱逐。
func (p *Preemptor) IssuePreemptions(ctx context.Context, preemptor *workload.Info, targets []*Target) (int, error) {
	log := ctrl.LoggerFrom(ctx)
	errCh := routine.NewErrorChannel()
	ctx, cancel := context.WithCancel(ctx)
	var successfullyPreempted atomic.Int64
	defer cancel()
	workqueue.ParallelizeUntil(ctx, parallelPreemptions, len(targets), func(i int) {
		target := targets[i]
		if !meta.IsStatusConditionTrue(target.WorkloadInfo.Obj.Status.Conditions, kueue.WorkloadEvicted) {
			message := preemptionMessage(preemptor.Obj, target.Reason)
			err := p.applyPreemption(ctx, target.WorkloadInfo.Obj, target.Reason, message)
			if err != nil {
				errCh.SendErrorWithCancel(err, cancel)
				return
			}

			log.V(3).Info("抢占", "targetWorkload", klog.KObj(target.WorkloadInfo.Obj), "preemptingWorkload", klog.KObj(preemptor.Obj), "reason", target.Reason, "message", message, "targetClusterQueue", klog.KRef("", string(target.WorkloadInfo.ClusterQueue)))
			p.recorder.Eventf(target.WorkloadInfo.Obj, corev1.EventTypeNormal, "抢占", message)
			metrics.ReportPreemption(preemptor.ClusterQueue, target.Reason, target.WorkloadInfo.ClusterQueue)
		} else {
			log.V(3).Info("抢占中", "targetWorkload", klog.KObj(target.WorkloadInfo.Obj), "preemptingWorkload", klog.KObj(preemptor.Obj))
		}
		successfullyPreempted.Add(1)
	})
	return int(successfullyPreempted.Load()), errCh.ReceiveError()
}

func (p *Preemptor) applyPreemptionWithSSA(ctx context.Context, w *kueue.Workload, reason, message string) error {
	w = w.DeepCopy()
	workload.SetEvictedCondition(w, kueue.WorkloadEvictedByPreemption, message)
	workload.ResetChecksOnEviction(w, p.clock.Now())
	reportWorkloadEvictedOnce := workload.WorkloadEvictionStateInc(w, kueue.WorkloadEvictedByPreemption, "")
	workload.SetPreemptedCondition(w, reason, message)
	if reportWorkloadEvictedOnce {
		metrics.ReportEvictedWorkloadsOnce(w.Status.Admission.ClusterQueue, kueue.WorkloadEvictedByPreemption, "")
	}
	return workload.ApplyAdmissionStatus(ctx, p.client, w, true, p.clock)
}

type preemptionAttemptOpts struct {
	borrowing bool
}

// classicalPreemptions 实现了一种启发式方法，用于找到最小的需要抢占的工作负载集合。
// 该启发式方法首先按输入顺序移除候选项，只要它们的 ClusterQueue 仍在借用资源，且新工作负载无法适配配额。
// 一旦新工作负载适配，启发式方法会按相反顺序尝试将工作负载加回，只要新工作负载仍然适配。
func (p *Preemptor) classicalPreemptions(preemptionCtx *preemptionCtx) []*Target {
	hierarchicalReclaimCtx := &classical.HierarchicalPreemptionCtx{
		Wl:                preemptionCtx.preemptor.Obj,
		Cq:                preemptionCtx.preemptorCQ,
		FrsNeedPreemption: preemptionCtx.frsNeedPreemption,
		Requests:          preemptionCtx.workloadUsage.Quota,
		WorkloadOrdering:  p.workloadOrdering,
	}
	candidatesGenerator := classical.NewCandidateIterator(hierarchicalReclaimCtx, preemptionCtx.frsNeedPreemption, preemptionCtx.snapshot, p.clock, CandidatesOrdering)
	var attemptPossibleOpts []preemptionAttemptOpts
	borrowWithinCohortForbidden, _ := classical.IsBorrowingWithinCohortForbidden(preemptionCtx.preemptorCQ)
	// 我们有三类候选项：
	// 1. 层级候选项。新工作负载对其有层级优势（更接近候选项所用配额）。可以不考虑优先级直接抢占。
	// 2. 优先级候选项。对其没有层级优势，但是否可抢占取决于优先级。仅对这些候选项遵循 BorrowWithinCohort 配置。
	// 3. 同队列候选项。
	// 我们只能抢占优先级大于 MaxPriorityThreshold 的优先级候选项，
	// 如果目标 CQ 没有借用（根据 MaxPriorityThreshold 的定义）。
	// 有时需要同时考虑 allowBorrowing = true 和 false 两种情况（因为 false 时候选项更多但不能借用）。
	// 选项的考虑顺序是任意的，先尝试 allowBorrowing=false 是为了与旧版本兼容。
	switch {
	case candidatesGenerator.NoCandidateFromOtherQueues || (borrowWithinCohortForbidden && !queueUnderNominalInResourcesNeedingPreemption(preemptionCtx)):
		attemptPossibleOpts = []preemptionAttemptOpts{{true}}
	case borrowWithinCohortForbidden && candidatesGenerator.NoCandidateForHierarchicalReclaim:
		attemptPossibleOpts = []preemptionAttemptOpts{{false}, {true}}
	default:
		attemptPossibleOpts = []preemptionAttemptOpts{{true}, {false}}
	}

	for _, attemptOpts := range attemptPossibleOpts {
		var targets []*Target
		candidatesGenerator.Reset()
		for candidate, reason := candidatesGenerator.Next(attemptOpts.borrowing); candidate != nil; candidate, reason = candidatesGenerator.Next(attemptOpts.borrowing) {
			preemptionCtx.snapshot.RemoveWorkload(candidate)
			targets = append(targets, &Target{
				WorkloadInfo: candidate,
				Reason:       reason,
			})
			if workloadFits(preemptionCtx, attemptOpts.borrowing) {
				targets = fillBackWorkloads(preemptionCtx, targets, attemptOpts.borrowing)
				restoreSnapshot(preemptionCtx.snapshot, targets)
				return targets
			}
		}
		restoreSnapshot(preemptionCtx.snapshot, targets)
	}
	return nil
}

func fillBackWorkloads(preemptionCtx *preemptionCtx, targets []*Target, allowBorrowing bool) []*Target {
	// 以相反顺序检查是否可以将部分工作负载加回。
	for i := len(targets) - 2; i >= 0; i-- {
		preemptionCtx.snapshot.AddWorkload(targets[i].WorkloadInfo)
		if workloadFits(preemptionCtx, allowBorrowing) {
			// O(1) deletion: copy the last element into index i and reduce size.
			targets[i] = targets[len(targets)-1]
			targets = targets[:len(targets)-1]
		} else {
			preemptionCtx.snapshot.RemoveWorkload(targets[i].WorkloadInfo)
		}
	}
	return targets
}

func restoreSnapshot(snapshot *cache.Snapshot, targets []*Target) {
	for _, t := range targets {
		snapshot.AddWorkload(t.WorkloadInfo)
	}
}

// parseStrategies 将策略数组转换为算法使用的函数。
// 该函数利用了抢占算法的属性和策略，以减少返回的函数数量。
// 返回的函数数量可能与输入切片不匹配。
func parseStrategies(s []config.PreemptionStrategy) []fairsharing.Strategy {
	if len(s) == 0 {
		return []fairsharing.Strategy{fairsharing.LessThanOrEqualToFinalShare, fairsharing.LessThanInitialShare}
	}
	strategies := make([]fairsharing.Strategy, len(s))
	for i, strategy := range s {
		switch strategy {
		case config.LessThanOrEqualToFinalShare:
			strategies[i] = fairsharing.LessThanOrEqualToFinalShare
		case config.LessThanInitialShare:
			strategies[i] = fairsharing.LessThanInitialShare
		}
	}
	return strategies
}

// runFirstFsStrategy 运行第一个配置的 FairSharing 策略，
// 并返回 (fits, targets, retryCandidates) retryCandidates 可能
// 在配置了规则 S2-b 时使用。
func runFirstFsStrategy(preemptionCtx *preemptionCtx, candidates []*workload.Info, strategy fairsharing.Strategy) (bool, []*Target, []*workload.Info) {
	ordering := fairsharing.MakeClusterQueueOrdering(preemptionCtx.preemptorCQ, candidates)
	var targets []*Target
	var retryCandidates []*workload.Info
	for candCQ := range ordering.Iter() {
		if candCQ.InClusterQueuePreemption() {
			candWl := candCQ.PopWorkload()
			preemptionCtx.snapshot.RemoveWorkload(candWl)
			targets = append(targets, &Target{
				WorkloadInfo: candWl,
				Reason:       kueue.InClusterQueueReason,
			})
			if workloadFitsForFairSharing(preemptionCtx) {
				return true, targets, nil
			}
			continue
		}

		preemptorNewShare, targetOldShare := candCQ.ComputeShares()
		for candCQ.HasWorkload() {
			candWl := candCQ.PopWorkload()
			targetNewShare := candCQ.ComputeTargetShareAfterRemoval(candWl)
			if strategy(preemptorNewShare, targetOldShare, targetNewShare) {
				preemptionCtx.snapshot.RemoveWorkload(candWl)
				reason := kueue.InCohortFairSharingReason

				targets = append(targets, &Target{
					WorkloadInfo: candWl,
					Reason:       reason,
				})
				if workloadFitsForFairSharing(preemptionCtx) {
					return true, targets, nil
				}
				// 可能需要选择不同的 CQ 因为值发生变化。
				break
			} else {
				retryCandidates = append(retryCandidates, candWl)
			}
		}
	}
	return false, targets, retryCandidates
}

// runSecondFsStrategy 实现 Fair Sharing 规则 S2-b。它返回
// (fits, targets)。
func runSecondFsStrategy(retryCandidates []*workload.Info, preemptionCtx *preemptionCtx, targets []*Target) (bool, []*Target) {
	ordering := fairsharing.MakeClusterQueueOrdering(preemptionCtx.preemptorCQ, retryCandidates)
	for candCQ := range ordering.Iter() {
		preemptorNewShare, targetOldShare := candCQ.ComputeShares()
		// 由于 API 验证，我们只能到达这里，如果第二个策略是 LessThanInitialShare，
		// 在这种情况下，策略函数的最后一个参数无关紧要。
		if fairsharing.LessThanInitialShare(preemptorNewShare, targetOldShare, 0) {
			// 标准不依赖于被抢占的工作负载，所以只需抢占第一个候选者。
			candWl := candCQ.PopWorkload()
			preemptionCtx.snapshot.RemoveWorkload(candWl)
			targets = append(targets, &Target{
				WorkloadInfo: candWl,
				Reason:       kueue.InCohortFairSharingReason,
			})
			if workloadFitsForFairSharing(preemptionCtx) {
				return true, targets
			}
		}
		// 似乎没有场景可以多次应用规则 S2-b 在同一个 CQ 中。
		ordering.DropQueue(candCQ)
	}
	return false, targets
}

func (p *Preemptor) fairPreemptions(preemptionCtx *preemptionCtx, strategies []fairsharing.Strategy) []*Target {
	candidates := p.findCandidates(preemptionCtx.preemptor.Obj, preemptionCtx.preemptorCQ, preemptionCtx.frsNeedPreemption)
	if len(candidates) == 0 {
		return nil
	}
	sort.Slice(candidates, func(i, j int) bool {
		return CandidatesOrdering(candidates[i], candidates[j], preemptionCtx.preemptorCQ.Name, p.clock.Now())
	})
	if logV := preemptionCtx.log.V(5); logV.Enabled() {
		logV.Info("模拟公平抢占", "candidates", workload.References(candidates), "resourcesRequiringPreemption", preemptionCtx.frsNeedPreemption.UnsortedList(), "preemptingWorkload", klog.KObj(preemptionCtx.preemptor.Obj))
	}

	// DRS 值必须包含新工作负载。
	revertSimulation := preemptionCtx.preemptorCQ.SimulateUsageAddition(preemptionCtx.workloadUsage)

	fits, targets, retryCandidates := runFirstFsStrategy(preemptionCtx, candidates, strategies[0])
	if !fits && len(strategies) > 1 {
		fits, targets = runSecondFsStrategy(retryCandidates, preemptionCtx, targets)
	}

	revertSimulation()
	if !fits {
		restoreSnapshot(preemptionCtx.snapshot, targets)
		return nil
	}
	targets = fillBackWorkloads(preemptionCtx, targets, true)
	restoreSnapshot(preemptionCtx.snapshot, targets)
	return targets
}

func flavorResourcesNeedPreemption(assignment flavorassigner.Assignment) sets.Set[resources.FlavorResource] {
	resPerFlavor := sets.New[resources.FlavorResource]()
	for _, ps := range assignment.PodSets {
		for res, flvAssignment := range ps.Flavors {
			if flvAssignment.Mode == flavorassigner.Preempt {
				resPerFlavor.Insert(resources.FlavorResource{Flavor: flvAssignment.Name, Resource: res})
			}
		}
	}
	return resPerFlavor
}

// findCandidates 获取在 ClusterQueue 和 cohort 中需要抢占的工作负载，
// 并尊重抢占策略。
func (p *Preemptor) findCandidates(wl *kueue.Workload, cq *cache.ClusterQueueSnapshot, frsNeedPreemption sets.Set[resources.FlavorResource]) []*workload.Info {
	var candidates []*workload.Info
	wlPriority := priority.Priority(wl)

	if cq.Preemption.WithinClusterQueue != kueue.PreemptionPolicyNever {
		considerSamePrio := cq.Preemption.WithinClusterQueue == kueue.PreemptionPolicyLowerOrNewerEqualPriority
		preemptorTS := p.workloadOrdering.GetQueueOrderTimestamp(wl)

		for _, candidateWl := range cq.Workloads {
			candidatePriority := priority.Priority(candidateWl.Obj)
			if candidatePriority > wlPriority {
				continue
			}

			if candidatePriority == wlPriority && (!considerSamePrio || !preemptorTS.Before(p.workloadOrdering.GetQueueOrderTimestamp(candidateWl.Obj))) {
				continue
			}

			if !classical.WorkloadUsesResources(candidateWl, frsNeedPreemption) {
				continue
			}
			candidates = append(candidates, candidateWl)
		}
	}

	if cq.HasParent() && cq.Preemption.ReclaimWithinCohort != kueue.PreemptionPolicyNever {
		onlyLowerPriority := cq.Preemption.ReclaimWithinCohort != kueue.PreemptionPolicyAny
		for _, cohortCQ := range cq.Parent().Root().SubtreeClusterQueues() {
			if cq == cohortCQ || !cqIsBorrowing(cohortCQ, frsNeedPreemption) {
				// 不能从自身或不借用资源的 ClusterQueue 抢占配额。
				continue
			}
			for _, candidateWl := range cohortCQ.Workloads {
				if onlyLowerPriority && priority.Priority(candidateWl.Obj) >= priority.Priority(wl) {
					continue
				}
				if !classical.WorkloadUsesResources(candidateWl, frsNeedPreemption) {
					continue
				}
				candidates = append(candidates, candidateWl)
			}
		}
	}
	return candidates
}

func cqIsBorrowing(cq *cache.ClusterQueueSnapshot, frsNeedPreemption sets.Set[resources.FlavorResource]) bool {
	if !cq.HasParent() {
		return false
	}
	for fr := range frsNeedPreemption {
		if cq.Borrowing(fr) {
			return true
		}
	}
	return false
}

// workloadFits 确定工作负载请求是否适合给定资源和 ClusterQueue 及其 cohort 的模拟使用，
// 如果它属于一个 cohort。
func workloadFits(preemptionCtx *preemptionCtx, allowBorrowing bool) bool {
	for fr, v := range preemptionCtx.workloadUsage.Quota {
		if !allowBorrowing && preemptionCtx.preemptorCQ.BorrowingWith(fr, v) {
			return false
		}
		if v > preemptionCtx.preemptorCQ.Available(fr) {
			return false
		}
	}
	tasResult := preemptionCtx.preemptorCQ.FindTopologyAssignmentsForWorkload(preemptionCtx.tasRequests)
	return tasResult.Failure() == nil
}

// workloadFitsForFairSharing 是一个轻量级的 workloadFits 包装，
// 因为我们需要先移除，然后添加回，新工作负载的使用，
// 因为 FairSharing 在处理开始时添加了此使用，以准确计算 DominantResourceShare。
func workloadFitsForFairSharing(preemptionCtx *preemptionCtx) bool {
	revertSimulation := preemptionCtx.preemptorCQ.SimulateUsageRemoval(preemptionCtx.workloadUsage)
	res := workloadFits(preemptionCtx, true)
	revertSimulation()
	return res
}

func queueUnderNominalInResourcesNeedingPreemption(preemptionCtx *preemptionCtx) bool {
	for fr := range preemptionCtx.frsNeedPreemption {
		if preemptionCtx.preemptorCQ.ResourceNode.Usage[fr] >= preemptionCtx.preemptorCQ.QuotaFor(fr).Nominal {
			return false
		}
	}
	return true
}

func resourceUsagePreemptionEnabled(a, b *workload.Info) bool {
	// 如果两个工作负载在同一个 ClusterQueue 中，但属于不同的 LocalQueues，
	// 我们可以比较它们的 LocalQueue 使用情况。
	// 如果两个工作负载的 LocalQueueUsage 都不为 nil，则表示功能门已启用，
	// 并且 ClusterQueue 的 AdmissionScope 设置为 UsageBasedFairSharing。
	// 我们从快照初始化继承此信息。
	return a.ClusterQueue == b.ClusterQueue && a.Obj.Spec.QueueName != b.Obj.Spec.QueueName && a.LocalQueueFSUsage != nil && b.LocalQueueFSUsage != nil
}

// CandidatesOrdering 排序标准：
// 0. 首先标记为需要抢占的工作负载。
// 1. 在 cohort 中，其他 ClusterQueue 的工作负载优先于同一 ClusterQueue 的工作负载。
// 2. (AdmissionFairSharing 仅) 使用 LocalQueue 较少的优先级工作负载优先。
// 3. 优先级较低的工作负载优先。
// 4. 更早被准入的工作负载优先。
func CandidatesOrdering(a, b *workload.Info, cq kueue.ClusterQueueReference, now time.Time) bool {
	aEvicted := meta.IsStatusConditionTrue(a.Obj.Status.Conditions, kueue.WorkloadEvicted)
	bEvicted := meta.IsStatusConditionTrue(b.Obj.Status.Conditions, kueue.WorkloadEvicted)
	if aEvicted != bEvicted {
		return aEvicted
	}
	aInCQ := a.ClusterQueue == cq
	bInCQ := b.ClusterQueue == cq
	if aInCQ != bInCQ {
		return !aInCQ
	}

	if features.Enabled(features.AdmissionFairSharing) && resourceUsagePreemptionEnabled(a, b) {
		if a.LocalQueueFSUsage != b.LocalQueueFSUsage {
			return *a.LocalQueueFSUsage > *b.LocalQueueFSUsage
		}
	}
	pa := priority.Priority(a.Obj)
	pb := priority.Priority(b.Obj)
	if pa != pb {
		return pa < pb
	}
	timeA := quotaReservationTime(a.Obj, now)
	timeB := quotaReservationTime(b.Obj, now)
	if !timeA.Equal(timeB) {
		return timeA.After(timeB)
	}
	// 任意比较以确保排序确定性。
	return a.Obj.UID < b.Obj.UID
}

// quotaReservationTime 返回配额保留的时间，如果条件未填充则使用当前时间。
func quotaReservationTime(wl *kueue.Workload, now time.Time) time.Time {
	cond := meta.FindStatusCondition(wl.Status.Conditions, kueue.WorkloadQuotaReserved)
	if cond == nil || cond.Status != metav1.ConditionTrue {
		// 条件尚未填充，使用当前时间。
		return now
	}
	return cond.LastTransitionTime.Time
}
