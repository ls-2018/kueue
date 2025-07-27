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
	"sigs.k8s.io/kueue/pkg/controller/constants"
	"sigs.k8s.io/kueue/pkg/metrics"
	"sigs.k8s.io/kueue/pkg/resources"
	"sigs.k8s.io/kueue/pkg/scheduler/flavorassigner"
	"sigs.k8s.io/kueue/pkg/scheduler/preemption/classical"
	"sigs.k8s.io/kueue/pkg/scheduler/preemption/fairsharing"
	"sigs.k8s.io/kueue/pkg/util/priority"
	"sigs.k8s.io/kueue/pkg/util/routine"
	"sigs.k8s.io/kueue/pkg/workload"
)

const parallelPreemptions = 8 // 并发抢占的最大数量

// Preemptor 负责调度抢占相关的所有操作
// 主要用于判断和执行抢占逻辑
// 其中包含了时钟、k8s client、事件记录器、排序方式、是否启用公平共享等
// applyPreemption 是实际执行抢占的函数指针
type Preemptor struct {
	clock clock.Clock // 时钟对象

	client   client.Client        // k8s 客户端
	recorder record.EventRecorder // 事件记录器

	workloadOrdering  workload.Ordering      // 工作负载排序方式
	enableFairSharing bool                   // 是否启用公平共享
	fsStrategies      []fairsharing.Strategy // 公平共享策略

	// stubs
	applyPreemption func(ctx context.Context, w *kueue.Workload, reason, message string) error // 实际执行抢占的函数
}

// preemptionCtx 用于存储抢占相关的上下文信息
// 包含日志、抢占者、快照、资源需求等
// 便于在抢占流程中传递和复用
type preemptionCtx struct {
	log               logr.Logger                        // 日志对象
	preemptor         workload.Info                      // 当前抢占者的工作负载信息
	preemptorCQ       *cache.ClusterQueueSnapshot        // 抢占者所在的ClusterQueue快照
	snapshot          *cache.Snapshot                    // 全局快照
	workloadUsage     workload.Usage                     // 工作负载资源使用情况
	tasRequests       cache.WorkloadTASRequests          // 工作负载的拓扑请求
	frsNeedPreemption sets.Set[resources.FlavorResource] // 需要抢占的资源集合
}

// OverrideApply 用于替换抢占执行函数，便于测试或自定义
func (p *Preemptor) OverrideApply(f func(context.Context, *kueue.Workload, string, string) error) {
	p.applyPreemption = f // 替换抢占执行函数
}

// Target 结构体，表示一个被抢占的目标
// WorkloadInfo 是被抢占的工作负载，Reason 是抢占原因
type Target struct {
	WorkloadInfo *workload.Info // 被抢占的工作负载信息
	Reason       string         // 抢占原因
}

// GetTargets返回需要抢占的workload列表，以确保wl能够被调度
// log: 日志对象，wl: 预调度的工作负载，assignment: 资源分配，snapshot: 全局快照
func (p *Preemptor) GetTargets(log logr.Logger, wl workload.Info, assignment flavorassigner.Assignment, snapshot *cache.Snapshot) []*Target {
	cq := snapshot.ClusterQueue(wl.ClusterQueue)                 // 获取工作负载所在的ClusterQueue快照
	tasRequests := assignment.WorkloadsTopologyRequests(&wl, cq) // ✅获取拓扑请求
	return p.getTargets(&preemptionCtx{
		log:               log,                                       // 日志
		preemptor:         wl,                                        // 抢占者
		preemptorCQ:       cq,                                        // 抢占者所在CQ
		snapshot:          snapshot,                                  // 全局快照
		tasRequests:       tasRequests,                               // 拓扑请求
		frsNeedPreemption: flavorResourcesNeedPreemption(assignment), // 需要抢占的资源
		workloadUsage: workload.Usage{
			Quota: assignment.TotalRequestsFor(&wl), // 总资源请求
			TAS:   wl.TASUsage(),                    // TAS使用量
		},
	})
}

// HumanReadablePreemptionReasons 用于将抢占原因转为可读中文
var HumanReadablePreemptionReasons = map[string]string{
	kueue.InClusterQueueReason:                "优先级调整",           // 在ClusterQueue中优先级调整
	kueue.InCohortReclamationReason:           "在 cohort 中回收",    // 在cohort中回收资源
	kueue.InCohortFairSharingReason:           "在 cohort 中公平共享",  // 在cohort中公平共享
	kueue.InCohortReclaimWhileBorrowingReason: "在 cohort 中回收时借用", // 在cohort中回收时借用
	"": "未知", // 未知原因
}

// preemptionMessage 生成抢占事件的详细信息
func preemptionMessage(preemptor *kueue.Workload, reason string) string {
	var wUID, jUID string
	if preemptor.UID == "" {
		wUID = "未知" // 如果没有UID，标记为未知
	} else {
		wUID = string(preemptor.UID) // 获取UID
	}
	uid := preemptor.Labels[constants.JobUIDLabel] // 获取JobUID
	if uid != "" {
		jUID = uid
	} else {
		jUID = "未知" // 没有JobUID则为未知
	}

	return fmt.Sprintf("抢占以腾出工作负载 (UID: %s, JobUID: %s) 由于 %s", wUID, jUID, HumanReadablePreemptionReasons[reason]) // 生成抢占信息
}

// IssuePreemptions 标记目标workload为已抢占
// ctx: 上下文，preemptor: 抢占者，targets: 被抢占的目标
// 返回成功抢占的数量和错误
func (p *Preemptor) IssuePreemptions(ctx context.Context, preemptor *workload.Info, targets []*Target) (int, error) {
	log := ctrl.LoggerFrom(ctx)            // 获取日志
	errCh := routine.NewErrorChannel()     // 错误通道
	ctx, cancel := context.WithCancel(ctx) // 创建可取消的上下文
	var successfullyPreempted atomic.Int64 // 统计成功抢占数量
	defer cancel()                         // 结束时取消上下文
	workqueue.ParallelizeUntil(ctx, parallelPreemptions, len(targets), func(i int) {
		target := targets[i] // 获取目标
		if !meta.IsStatusConditionTrue(target.WorkloadInfo.Obj.Status.Conditions, kueue.WorkloadEvicted) {
			message := preemptionMessage(preemptor.Obj, target.Reason)                     // 生成抢占信息
			err := p.applyPreemption(ctx, target.WorkloadInfo.Obj, target.Reason, message) // 执行抢占
			if err != nil {
				errCh.SendErrorWithCancel(err, cancel) // 发送错误并取消
				return
			}

			log.V(3).Info("已抢占", "targetWorkload", klog.KObj(target.WorkloadInfo.Obj), "preemptingWorkload", klog.KObj(preemptor.Obj), "reason", target.Reason, "message", message, "targetClusterQueue", klog.KRef("", string(target.WorkloadInfo.ClusterQueue))) // 记录日志
			p.recorder.Eventf(target.WorkloadInfo.Obj, corev1.EventTypeNormal, "Preempted", message)                                                                                                                                                               // 记录事件
			metrics.ReportPreemption(preemptor.ClusterQueue, target.Reason, target.WorkloadInfo.ClusterQueue)                                                                                                                                                      // 上报指标
		} else {
			log.V(3).Info("抢占中", "targetWorkload", klog.KObj(target.WorkloadInfo.Obj), "preemptingWorkload", klog.KObj(preemptor.Obj)) // 已在抢占中
		}
		successfullyPreempted.Add(1) // 成功数+1
	})
	return int(successfullyPreempted.Load()), errCh.ReceiveError() // 返回抢占数量和错误
}

// applyPreemptionWithSSA 使用 Server Side Apply 标记 workload 被抢占
// w: 被抢占的工作负载，reason: 原因，message: 详细信息
func (p *Preemptor) applyPreemptionWithSSA(ctx context.Context, w *kueue.Workload, reason, message string) error {
	w = w.DeepCopy()                                                                                         // 深拷贝，避免影响原对象
	workload.SetEvictedCondition(w, kueue.WorkloadEvictedByPreemption, message)                              // 设置抢占条件
	workload.ResetChecksOnEviction(w, p.clock.Now())                                                         // 重置检查
	reportWorkloadEvictedOnce := workload.WorkloadEvictionStateInc(w, kueue.WorkloadEvictedByPreemption, "") // 增加抢占状态
	workload.SetPreemptedCondition(w, reason, message)                                                       // 设置抢占原因
	if reportWorkloadEvictedOnce {
		metrics.ReportEvictedWorkloadsOnce(w.Status.Admission.ClusterQueue, kueue.WorkloadEvictedByPreemption, "") // 上报指标
	}
	return workload.ApplyAdmissionStatus(ctx, p.client, w, true, p.clock) // 应用 admission 状态
}

// preemptionAttemptOpts 抢占尝试的选项，主要用于是否允许借用
// borrowing: 是否允许借用资源
type preemptionAttemptOpts struct {
	borrowing bool // 是否允许借用
}

// restoreSnapshot 恢复快照，将所有被移除的工作负载重新加入快照
func restoreSnapshot(snapshot *cache.Snapshot, targets []*Target) {
	for _, t := range targets {
		snapshot.AddWorkload(t.WorkloadInfo) // 恢复工作负载
	}
}

// runSecondFsStrategy实现公平共享规则 S2-b。它返回 (fits, targets)。
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
		// 似乎没有场景可以应用规则 S2-b 超过一次在一个 CQ 中。
		ordering.DropQueue(candCQ)
	}
	return false, targets
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

// findCandidates获取在ClusterQueue和cohort中需要抢占的候选者，
// 这些候选者尊重抢占策略，并使用工作负载需要使用的资源。
func (p *Preemptor) findCandidates(wl *kueue.Workload, cq *cache.ClusterQueueSnapshot, frsNeedPreemption sets.Set[resources.FlavorResource]) []*workload.Info {
	var candidates []*workload.Info
	wlPriority := priority.Priority(wl)
	preemptorTS := p.workloadOrdering.GetQueueOrderTimestamp(wl)

	if cq.Preemption.WithinClusterQueue != kueue.PreemptionPolicyNever {
		// 低优、 相同优先级且新的
		considerSamePrio := cq.Preemption.WithinClusterQueue == kueue.PreemptionPolicyLowerOrNewerEqualPriority

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
		for _, cohortCQ := range cq.Parent().Root().SubtreeClusterQueues() {
			if cq == cohortCQ || !cqIsBorrowing(cohortCQ, frsNeedPreemption) {
				// 不能从自身或不借用配额的ClusterQueues回收配额。
				continue
			}
			for _, candidateWl := range cohortCQ.Workloads {
				// 缺少 判断
				switch cq.Preemption.ReclaimWithinCohort {
				case kueue.PreemptionPolicyAny:
				case kueue.PreemptionPolicyLowerPriority:
					if priority.Priority(candidateWl.Obj) >= priority.Priority(wl) {
						continue
					}
				case kueue.PreemptionPolicyLowerOrNewerEqualPriority:
					if priority.Priority(candidateWl.Obj) > priority.Priority(wl) {
						continue
					}

					if priority.Priority(candidateWl.Obj) == priority.Priority(wl) && !preemptorTS.Before(p.workloadOrdering.GetQueueOrderTimestamp(candidateWl.Obj)) {
						continue
					}
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

func queueUnderNominalInResourcesNeedingPreemption(preemptionCtx *preemptionCtx) bool {
	for fr := range preemptionCtx.frsNeedPreemption {
		if preemptionCtx.preemptorCQ.ResourceNode.Usage[fr] >= preemptionCtx.preemptorCQ.QuotaFor(fr).Nominal {
			return false
		}
	}
	return true
}

// candidatesOrdering 排序标准：
// 0. 首先标记为抢占的工作负载。
// 1. 来自 cohort 中其他 ClusterQueues 的候选者，在同一 ClusterQueue 之前。
// 2. 优先级较低的候选者。
// 3. 更早被调度的候选者。
func CandidatesOrdering(candidates []*workload.Info, cq kueue.ClusterQueueReference, now time.Time) func(int, int) bool {
	return func(i, j int) bool {
		a := candidates[i]
		b := candidates[j]
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
		// 任意比较以确保确定性排序。
		return a.Obj.UID < b.Obj.UID
	}
}

func quotaReservationTime(wl *kueue.Workload, now time.Time) time.Time {
	cond := meta.FindStatusCondition(wl.Status.Conditions, kueue.WorkloadQuotaReserved)
	if cond == nil || cond.Status != metav1.ConditionTrue {
		// 条件尚未填充，使用当前时间。
		return now
	}
	return cond.LastTransitionTime.Time
}

func (p *Preemptor) getTargets(preemptionCtx *preemptionCtx) []*Target {
	if p.enableFairSharing {
		return p.fairPreemptions(preemptionCtx, p.fsStrategies)
	}
	return p.classicalPreemptions(preemptionCtx)
}

func New(cl client.Client, workloadOrdering workload.Ordering, recorder record.EventRecorder, fs config.FairSharing, clock clock.Clock) *Preemptor {
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

func (p *Preemptor) fairPreemptions(preemptionCtx *preemptionCtx, strategies []fairsharing.Strategy) []*Target {
	candidates := p.findCandidates(preemptionCtx.preemptor.Obj, // kueue.Workload
		preemptionCtx.preemptorCQ,
		preemptionCtx.frsNeedPreemption)
	if len(candidates) == 0 {
		return nil
	}
	sort.Slice(candidates, CandidatesOrdering(candidates, preemptionCtx.preemptorCQ.Name, p.clock.Now()))
	if logV := preemptionCtx.log.V(5); logV.Enabled() {
		logV.Info("模拟公平抢占", "candidates", workload.References(candidates), "resourcesRequiringPreemption", preemptionCtx.frsNeedPreemption.UnsortedList(), "preemptingWorkload", klog.KObj(preemptionCtx.preemptor.Obj))
	}

	revertSimulation := preemptionCtx.preemptorCQ.SimulateUsageAddition(preemptionCtx.workloadUsage) // 模拟使用量增加

	fits, targets, retryCandidates := runFirstFsStrategy(preemptionCtx, candidates, strategies[0])

	if !fits && len(strategies) > 1 {
		fits, targets = runSecondFsStrategy(retryCandidates, preemptionCtx, targets)
	}

	revertSimulation()
	if !fits {
		restoreSnapshot(preemptionCtx.snapshot, targets)
		return nil
	}
	targets = fillBackWorkloads(preemptionCtx, targets, true) // ✅
	restoreSnapshot(preemptionCtx.snapshot, targets)
	return targets
}

// runFirstFsStrategy运行第一个配置的公平共享策略，
// 并返回 (fits, targets, retryCandidates) retryCandidates 可能
// 用于规则 S2-b 配置。
func runFirstFsStrategy(preemptionCtx *preemptionCtx, candidates []*workload.Info, strategy fairsharing.Strategy) (bool, []*Target, []*workload.Info) {
	ordering := fairsharing.MakeClusterQueueOrdering(preemptionCtx.preemptorCQ, candidates)
	var targets []*Target
	var retryCandidates []*workload.Info
	for candCQ := range ordering.Iter() {
		if candCQ.InClusterQueuePreemption() { // 同一个CQ
			candWl := candCQ.PopWorkload()
			preemptionCtx.snapshot.RemoveWorkload(candWl)
			targets = append(targets, &Target{
				WorkloadInfo: candWl,
				Reason:       kueue.InClusterQueueReason,
			})
			if workloadFitsForFairSharing(preemptionCtx) { // 1 ✅
				return true, targets, nil
			}
			continue
		}

		preemptorNewShare, targetOldShare := candCQ.ComputeShares() // 1 申请资源 占用比例较大值
		for candCQ.HasWorkload() {
			candWl := candCQ.PopWorkload()
			targetNewShare := candCQ.ComputeTargetShareAfterRemoval(candWl)
			if strategy(preemptorNewShare, targetOldShare, targetNewShare) {
				// 资源占比小于要驱逐的
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

// workloadFits 确定工作负载请求是否适应给定的资源和模拟的ClusterQueue和cohort的使用，
func workloadFits(preemptionCtx *preemptionCtx, allowBorrowing bool) bool {
	for fr, v := range preemptionCtx.workloadUsage.Quota {
		if !allowBorrowing && preemptionCtx.preemptorCQ.BorrowingWith(fr, v) {
			return false
		}
		if v > preemptionCtx.preemptorCQ.Available(fr) {
			return false
		}
	}
	tasResult := preemptionCtx.preemptorCQ.FindTopologyAssignmentsForWorkload(preemptionCtx.tasRequests, false, nil)
	return tasResult.Failure() == nil
}

// workloadFitsForFairSharing是一个轻量级的 workloadFits 包装，
// 因为我们需要移除，然后添加回，工作负载的使用，
// 因为公平共享在处理开始时添加此使用，以准确计算 DominantResourceShare。
func workloadFitsForFairSharing(preemptionCtx *preemptionCtx) bool {
	revertSimulation := preemptionCtx.preemptorCQ.SimulateUsageRemoval(preemptionCtx.workloadUsage)
	res := workloadFits(preemptionCtx, true) // 1 ✅
	revertSimulation()
	return res
}

// fillBackWorkloads 尝试将部分被移除的工作负载回填
// preemptionCtx: 抢占上下文，targets: 当前抢占目标，allowBorrowing: 是否允许借用
func fillBackWorkloads(preemptionCtx *preemptionCtx, targets []*Target, allowBorrowing bool) []*Target {
	for i := len(targets) - 2; i >= 0; i-- { // 逆序遍历，跳过最后一个
		preemptionCtx.snapshot.AddWorkload(targets[i].WorkloadInfo) // 尝试回填
		if workloadFits(preemptionCtx, allowBorrowing) {            // 回填后仍能调度
			targets[i] = targets[len(targets)-1] // O(1) 删除
			targets = targets[:len(targets)-1]
		} else {
			preemptionCtx.snapshot.RemoveWorkload(targets[i].WorkloadInfo) // 回填失败则移除
		}
	}
	return targets // 返回最终抢占目标
}

// classicalPreemptions 实现启发式算法，找到需要抢占的最小工作负载集合
// preemptionCtx: 抢占上下文，包含所有抢占相关信息
func (p *Preemptor) classicalPreemptions(preemptionCtx *preemptionCtx) []*Target {
	hierarchicalReclaimCtx := &classical.HierarchicalPreemptionCtx{
		Wl:                preemptionCtx.preemptor.Obj,       // 抢占者对象
		Cq:                preemptionCtx.preemptorCQ,         // 抢占者所在CQ
		FrsNeedPreemption: preemptionCtx.frsNeedPreemption,   // 需要抢占的资源
		Requests:          preemptionCtx.workloadUsage.Quota, // 资源请求
		WorkloadOrdering:  p.workloadOrdering,                // 排序方式   入队时间
	}
	candidatesGenerator := classical.NewCandidateIterator(hierarchicalReclaimCtx, preemptionCtx.frsNeedPreemption, preemptionCtx.snapshot, p.clock, CandidatesOrdering) // 生成候选者迭代器
	var attemptPossibleOpts []preemptionAttemptOpts
	borrowWithinCohortForbidden, _ := classical.IsBorrowingWithinCohortForbidden(preemptionCtx.preemptorCQ) // 判断cohort内是否禁止借用
	// 三种候选者类型，见上方注释
	switch {
	case candidatesGenerator.NoCandidateFromOtherQueues || (borrowWithinCohortForbidden && !queueUnderNominalInResourcesNeedingPreemption(preemptionCtx)):
		attemptPossibleOpts = []preemptionAttemptOpts{{true}} // 只允许借用
	case borrowWithinCohortForbidden && candidatesGenerator.NoCandidateForHierarchicalReclaim:
		attemptPossibleOpts = []preemptionAttemptOpts{{false}, {true}} // 先不借用再借用
	default:
		attemptPossibleOpts = []preemptionAttemptOpts{{true}, {false}} // 先借用再不借用
	}

	for _, attemptOpts := range attemptPossibleOpts {
		var targets []*Target
		candidatesGenerator.Reset() // 重置候选者生成器
		for candidate, reason := candidatesGenerator.Next(attemptOpts.borrowing); candidate != nil; candidate, reason = candidatesGenerator.Next(attemptOpts.borrowing) {
			preemptionCtx.snapshot.RemoveWorkload(candidate) // 从快照中移除候选者
			targets = append(targets, &Target{
				WorkloadInfo: candidate, // 被抢占的工作负载
				Reason:       reason,    // 抢占原因
			})
			if workloadFits(preemptionCtx, attemptOpts.borrowing) { // 判断抢占后是否能调度
				targets = fillBackWorkloads(preemptionCtx, targets, attemptOpts.borrowing) // 尝试回填部分工作负载
				restoreSnapshot(preemptionCtx.snapshot, targets)                           // 恢复快照
				return targets                                                             // 返回抢占目标
			}
		}
		restoreSnapshot(preemptionCtx.snapshot, targets) // 恢复快照
	}
	return nil // 没有可抢占目标
}
