/*
调度器（Scheduler）是 Kueue 的核心组件，负责从队列中挑选待调度的 Workload，
并根据资源快照、优先级、配额、抢占等策略，决定哪些 Workload 可以被准入（admit），
并为其分配资源风味（ResourceFlavor）和集群队列（ClusterQueue）。

本文件实现了调度器的主要流程，包括：
- 队列头部 Workload 的获取
- 资源快照的构建
- Workload 的资源需求分析与分配尝试
- 公平/经典调度策略的迭代器
- 抢占与资源借用处理
- Workload 的准入与事件上报
- Workload 的重入队与状态更新
*/
package scheduler

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sort"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	config "sigs.k8s.io/kueue/apis/config/v1beta1"
	kueuealpha "sigs.k8s.io/kueue/apis/kueue/v1alpha1"
	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/cache"
	"sigs.k8s.io/kueue/pkg/features"
	"sigs.k8s.io/kueue/pkg/metrics"
	"sigs.k8s.io/kueue/pkg/queue"
	"sigs.k8s.io/kueue/pkg/resources"
	"sigs.k8s.io/kueue/pkg/scheduler/flavorassigner"
	"sigs.k8s.io/kueue/pkg/scheduler/preemption"
	"sigs.k8s.io/kueue/pkg/util/api"
	"sigs.k8s.io/kueue/pkg/util/priority"
	"sigs.k8s.io/kueue/pkg/util/routine"
	"sigs.k8s.io/kueue/pkg/util/wait"
	"sigs.k8s.io/kueue/pkg/workload"
)

const (
	errCouldNotAdmitWL                           = "Could not admit Workload and assign flavors in apiserver"
	errInvalidWLResources                        = "resources validation failed"
	errLimitRangeConstraintsUnsatisfiedResources = "resources didn't satisfy LimitRange constraints"
)

var (
	realClock = clock.RealClock{}
)

// Scheduler 结构体定义了调度器的核心依赖和配置项。
type Scheduler struct {
	queues                  *queue.Manager        // 队列管理器，负责管理所有队列及其 Workload
	cache                   *cache.Cache          // 资源缓存，维护集群资源快照和已准入 Workload
	client                  client.Client         // K8s 客户端，用于与 API Server 交互
	recorder                record.EventRecorder  // 事件记录器，用于上报事件
	admissionRoutineWrapper routine.Wrapper       // 准入异步包装器
	preemptor               *preemption.Preemptor // 抢占器，负责抢占逻辑
	workloadOrdering        workload.Ordering     // Workload 排序策略
	fairSharing             config.FairSharing    // 公平调度配置
	clock                   clock.Clock           // 时钟接口，便于测试

	// schedulingCycle 表示自上次重启以来的调度周期数。
	// 每次 schedule 调用自增，用于日志和调度追踪。
	schedulingCycle int64

	// applyAdmission 是准入操作的实现函数，默认使用 SSA。
	applyAdmission func(context.Context, *kueue.Workload) error
}

type options struct {
	podsReadyRequeuingTimestamp config.RequeuingTimestamp
	fairSharing                 config.FairSharing
	clock                       clock.Clock
}

// Option 用于配置 reconciler。
type Option func(*options)

var defaultOptions = options{
	podsReadyRequeuingTimestamp: config.EvictionTimestamp,
	clock:                       realClock,
}

// WithPodsReadyRequeuingTimestamp 设置用于对因 PodsReady 条件被重新入队的工作负载进行排序的时间戳。
func WithPodsReadyRequeuingTimestamp(ts config.RequeuingTimestamp) Option {
	return func(o *options) {
		o.podsReadyRequeuingTimestamp = ts
	}
}

func WithFairSharing(fs *config.FairSharing) Option {
	return func(o *options) {
		if fs != nil {
			o.fairSharing = *fs
		}
	}
}

// New 创建一个新的 Scheduler 实例。
// queues: 队列管理器
// cache: 资源缓存
// cl: K8s 客户端
// recorder: 事件记录器
// opts: 可选配置项
func New(queues *queue.Manager, cache *cache.Cache, cl client.Client, recorder record.EventRecorder, opts ...Option) *Scheduler {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	wo := workload.Ordering{
		PodsReadyRequeuingTimestamp: options.podsReadyRequeuingTimestamp,
	}
	s := &Scheduler{
		fairSharing:             options.fairSharing,
		queues:                  queues,
		cache:                   cache,
		client:                  cl,
		recorder:                recorder,
		preemptor:               preemption.New(cl, wo, recorder, options.fairSharing, options.clock),
		admissionRoutineWrapper: routine.DefaultWrapper,
		workloadOrdering:        wo,
		clock:                   options.clock,
	}
	s.applyAdmission = s.applyAdmissionWithSSA
	return s
}

// NeedLeaderElection 实现 LeaderElectionRunnable 接口，使调度器以主节点选举模式运行。
func (s *Scheduler) NeedLeaderElection() bool {
	return true
}

func (s *Scheduler) setAdmissionRoutineWrapper(wrapper routine.Wrapper) {
	s.admissionRoutineWrapper = wrapper
}

func setSkipped(e *entry, inadmissibleMsg string) {
	e.status = skipped
	e.inadmissibleMsg = inadmissibleMsg
	// 重置分配，以便在由于 Fit 不再适配而跳过，
	// 或由于与先前准入重叠而跳过 Preempt 后，重试所有资源类型。
	e.LastAssignment = nil
}

func reportSkippedPreemptions(p map[kueue.ClusterQueueReference]int) {
	for cqName, count := range p {
		metrics.AdmissionCyclePreemptionSkips.WithLabelValues(string(cqName)).Set(float64(count))
	}
}

// schedule 是调度器的主循环，每次调度周期会尝试准入一批 Workload。
// 返回值用于控制调度速率（快/慢/继续）。
func (s *Scheduler) schedule(ctx context.Context) wait.SpeedSignal {
	s.schedulingCycle++
	log := ctrl.LoggerFrom(ctx).WithValues("schedulingCycle", s.schedulingCycle)
	ctx = ctrl.LoggerInto(ctx, log)

	// 步骤1：从队列管理器获取所有队列的头部 Workload。
	// 这些 Workload 是当前最有可能被调度的。
	headWorkloads := s.queues.Heads(ctx)
	// 如果没有元素，说明程序正在结束。
	if len(headWorkloads) == 0 {
		return wait.KeepGoing
	}
	startTime := s.clock.Now()

	// 步骤2：对当前集群资源和已准入 Workload 进行快照，便于后续分配和回滚。
	snapshot, err := s.cache.Snapshot(ctx)
	if err != nil {
		log.Error(err, "failed to build snapshot for scheduling")
		return wait.SlowDown
	}
	logSnapshotIfVerbose(log, snapshot)

	// 步骤3：分析每个 Workload 的资源需求，尝试分配资源风味和借用额度。
	entries, inadmissibleEntries := s.nominate(ctx, headWorkloads, snapshot)

	// 步骤4：根据配置选择公平调度或经典调度的迭代器，对 entries 进行排序和遍历。
	iterator := makeIterator(ctx, entries, s.workloadOrdering, s.fairSharing.Enable)

	// 步骤5：遍历排序后的 entries，依次尝试准入。
	// 处理抢占、借用、PodsReady 等特殊场景。
	preemptedWorkloads := make(preemption.PreemptedWorkloads)
	skippedPreemptions := make(map[kueue.ClusterQueueReference]int)
	for iterator.hasNext() {
		e := iterator.pop()

		cq := snapshot.ClusterQueue(e.Info.ClusterQueue)
		log := log.WithValues("workload", klog.KObj(e.Info.Obj), "clusterQueue", klog.KRef("", string(e.Info.ClusterQueue)))
		if cq.HasParent() {
			log = log.WithValues("parentCohort", klog.KRef("", string(cq.Parent().GetName())), "rootCohort", klog.KRef("", string(cq.Parent().Root().GetName())))
		}
		ctx := ctrl.LoggerInto(ctx, log)

		mode := e.assignment.RepresentativeMode()

		if features.Enabled(features.TASFailedNodeReplacementFailFast) && workload.HasTopologyAssignmentWithNodeToReplace(e.Info.Obj) && mode != flavorassigner.Fit {
			// 驱逐无法找到替换节点的工作负载
			if err := s.evictWorkloadAfterFailedTASReplacement(ctx, log, e.Info.Obj); err != nil {
				log.V(2).Error(err, "Failed to evict workload after failed try to find a node replacement")
				continue
			}
			e.status = evicted
			continue
		}

		if mode == flavorassigner.NoFit {
			log.V(3).Info("Skipping workload as FlavorAssigner assigned NoFit mode")
			continue
		}
		log.V(2).Info("Attempting to schedule workload")

		// 处理抢占模式下的特殊逻辑：
		// - 如果需要抢占但没有候选目标，则保留资源，防止被低优先级抢占
		// - 如果抢占目标有重叠，跳过本次准入
		// - 如果资源不再适配，跳过
		// - 如果发起抢占，记录抢占目标，等待抢占完成
		// - 如果需要等待所有已准入 Workload PodsReady，则阻塞准入
		// - 最终调用 admit 进行准入
		if mode == flavorassigner.Preempt && len(e.preemptionTargets) == 0 {
			log.V(2).Info("Workload requires preemption, but there are no candidate workloads allowed for preemption", "preemption", cq.Preemption)
			// 如果我们不确定是否能回收容量，则保留容量
			// 否则，允许 Cohort 中的其他工作负载借用该容量，
			// 并确信我们之后可以回收。
			if !preemption.CanAlwaysReclaim(cq) {
				// 保留容量直到借用上限，
				// 这样其他 Cohort 中优先级较低的工作负载就不能在我们之前被批准。
				cq.AddUsage(resourcesToReserve(e, cq))
			}
			continue
		}

		// 如果任何目标有重叠，则每个 cohort 跳过多次抢占
		if preemptedWorkloads.HasAny(e.preemptionTargets) {
			setSkipped(e, "Workload has overlapping preemption targets with another workload")
			skippedPreemptions[cq.Name]++
			continue
		}

		usage := e.assignmentUsage()
		if !fits(cq, &usage, preemptedWorkloads, e.preemptionTargets) {
			setSkipped(e, "Workload no longer fits after processing another workload")
			if mode == flavorassigner.Preempt {
				skippedPreemptions[cq.Name]++
			}
			continue
		}
		preemptedWorkloads.Insert(e.preemptionTargets)
		cq.AddUsage(usage)

		if e.assignment.RepresentativeMode() == flavorassigner.Preempt {
			// 如果发起了抢占，下次尝试应尝试所有资源类型。
			e.LastAssignment = nil
			preempted, err := s.preemptor.IssuePreemptions(ctx, &e.Info, e.preemptionTargets)
			if err != nil {
				log.Error(err, "Failed to preempt workloads")
			}
			if preempted != 0 {
				e.inadmissibleMsg += fmt.Sprintf(". Pending the preemption of %d workload(s)", preempted)
				e.requeueReason = queue.RequeueReasonPendingPreemption
			}
			continue
		}
		if !s.cache.PodsReadyForAllAdmittedWorkloads(log) {
			log.V(5).Info("Waiting for all admitted workloads to be in the PodsReady condition")
			// If WaitForPodsReady is enabled and WaitForPodsReady.BlockAdmission is true
			// Block admission until all currently admitted workloads are in
			// PodsReady condition if the waitForPodsReady is enabled
			workload.UnsetQuotaReservationWithCondition(e.Info.Obj, "Waiting", "waiting for all admitted workloads to be in PodsReady condition", s.clock.Now())
			if err := workload.ApplyAdmissionStatus(ctx, s.client, e.Info.Obj, false, s.clock); err != nil {
				log.Error(err, "Could not update Workload status")
			}
			s.cache.WaitForPodsReady(ctx)
			log.V(5).Info("Finished waiting for all admitted workloads to be in the PodsReady condition")
		}
		e.status = nominated
		if err := s.admit(ctx, e, cq); err != nil {
			e.inadmissibleMsg = fmt.Sprintf("Failed to admit workload: %v", err)
		}
	}

	// 步骤6：对未被准入的 Workload 进行重入队和状态更新。
	result := metrics.AdmissionResultInadmissible
	for _, e := range entries {
		logAdmissionAttemptIfVerbose(log, &e)
		// When the workload is evicted by scheduler we skip requeueAndUpdate.
		// The eviction process will be finalized by the workload controller.
		if e.status != assumed && e.status != evicted {
			s.requeueAndUpdate(ctx, e)
		} else {
			result = metrics.AdmissionResultSuccess
		}
	}
	for _, e := range inadmissibleEntries {
		logAdmissionAttemptIfVerbose(log, &e)
		s.requeueAndUpdate(ctx, e)
	}

	reportSkippedPreemptions(skippedPreemptions)
	metrics.AdmissionAttempt(result, s.clock.Since(startTime))
	if result != metrics.AdmissionResultSuccess {
		return wait.SlowDown
	}
	return wait.KeepGoing
}

type entryStatus string

const (
	// 表示工作负载已被提名准入。
	nominated entryStatus = "nominated"
	// 表示工作负载在本周期被跳过。
	skipped entryStatus = "skipped"
	// 表示工作负载在本周期被驱逐。
	evicted entryStatus = "evicted"
	// 表示工作负载已被假设已准入。
	assumed entryStatus = "assumed"
	// 表示工作负载从未被提名准入。
	notNominated entryStatus = ""
)

// entry 结构体表示一个待准入的 Workload 及其分配需求和状态。
type entry struct {
	workload.Info                                              // Workload 及其资源需求
	assignment           flavorassigner.Assignment             // 资源风味分配结果
	status               entryStatus                           // 当前准入状态
	inadmissibleMsg      string                                // 不可准入原因
	requeueReason        queue.RequeueReason                   // 重入队原因
	preemptionTargets    []*preemption.Target                  // 抢占目标列表
	clusterQueueSnapshot *cache.ClusterQueueSnapshot           // 所属 ClusterQueue 快照
	LastAssignment       *workload.AssignmentClusterQueueState // 上一次分配的 AssignmentClusterQueueState，用于重试和回滚
}

func (e *entry) assignmentUsage() workload.Usage {
	return netUsage(e, e.assignment.Usage.Quota)
}

// nominate 尝试为每个队首 Workload 分配资源风味和借用额度，返回可准入和不可准入的 entry 列表。
// - entries: 可准入的 entry
// - inadmissibleEntries: 不可准入的 entry
func (s *Scheduler) nominate(ctx context.Context, workloads []workload.Info, snap *cache.Snapshot) ([]entry, []entry) {
	log := ctrl.LoggerFrom(ctx)
	entries := make([]entry, 0, len(workloads))
	var inadmissibleEntries []entry
	for _, w := range workloads {
		log := log.WithValues("workload", klog.KObj(w.Obj), "clusterQueue", klog.KRef("", string(w.ClusterQueue)))
		ns := corev1.Namespace{}
		e := entry{Info: w}
		e.clusterQueueSnapshot = snap.ClusterQueue(w.ClusterQueue)
		if !workload.NeedsSecondPass(w.Obj) && s.cache.IsAssumedOrAdmittedWorkload(w) {
			log.Info("Workload skipped from admission because it's already accounted in cache, and it does not need second pass", "workload", klog.KObj(w.Obj))
			continue
		} else if workload.HasRetryChecks(w.Obj) || workload.HasRejectedChecks(w.Obj) {
			e.inadmissibleMsg = "The workload has failed admission checks"
		} else if snap.InactiveClusterQueueSets.Has(w.ClusterQueue) {
			e.inadmissibleMsg = fmt.Sprintf("ClusterQueue %s is inactive", w.ClusterQueue)
		} else if e.clusterQueueSnapshot == nil {
			e.inadmissibleMsg = fmt.Sprintf("ClusterQueue %s not found", w.ClusterQueue)
		} else if err := s.client.Get(ctx, types.NamespacedName{Name: w.Obj.Namespace}, &ns); err != nil {
			e.inadmissibleMsg = fmt.Sprintf("Could not obtain workload namespace: %v", err)
		} else if !e.clusterQueueSnapshot.NamespaceSelector.Matches(labels.Set(ns.Labels)) {
			e.inadmissibleMsg = "Workload namespace doesn't match ClusterQueue selector"
			e.requeueReason = queue.RequeueReasonNamespaceMismatch
		} else if err := workload.ValidateResources(&w); err != nil {
			e.inadmissibleMsg = fmt.Sprintf("%s: %v", errInvalidWLResources, err.ToAggregate())
		} else if err := workload.ValidateLimitRange(ctx, s.client, &w); err != nil {
			e.inadmissibleMsg = fmt.Sprintf("%s: %v", errLimitRangeConstraintsUnsatisfiedResources, err.ToAggregate())
		} else {
			e.assignment, e.preemptionTargets = s.getAssignments(log, &e.Info, snap)
			e.inadmissibleMsg = e.assignment.Message()
			e.LastAssignment = &e.assignment.LastState
			entries = append(entries, e)
			continue
		}
		inadmissibleEntries = append(inadmissibleEntries, e)
	}
	return entries, inadmissibleEntries
}

// fits 判断在考虑已抢占 Workload 后，当前 usage 是否还能适配 ClusterQueue。
func fits(cq *cache.ClusterQueueSnapshot, usage *workload.Usage, preemptedWorkloads preemption.PreemptedWorkloads, newTargets []*preemption.Target) bool {
	workloads := slices.Collect(maps.Values(preemptedWorkloads))
	for _, target := range newTargets {
		workloads = append(workloads, target.WorkloadInfo)
	}
	revertUsage := cq.SimulateWorkloadRemoval(workloads)
	defer revertUsage()
	return cq.Fits(*usage)
}

// resourcesToReserve 计算在 cq/cohort 分配中需要预留的资源量。
func resourcesToReserve(e *entry, cq *cache.ClusterQueueSnapshot) workload.Usage {
	return netUsage(e, quotaResourcesToReserve(e, cq))
}

// netUsage 计算配额和 TAS 预留的净用量。
func netUsage(e *entry, netQuota resources.FlavorResourceQuantities) workload.Usage {
	result := workload.Usage{}
	if features.Enabled(features.TopologyAwareScheduling) {
		result.TAS = e.assignment.ComputeTASNetUsage(e.Info.Obj.Status.Admission)
	}
	if !workload.HasQuotaReservation(e.Info.Obj) {
		result.Quota = netQuota
	}
	return result
}

func quotaResourcesToReserve(e *entry, cq *cache.ClusterQueueSnapshot) resources.FlavorResourceQuantities {
	if e.assignment.RepresentativeMode() != flavorassigner.Preempt {
		return e.assignment.Usage.Quota
	}
	reservedUsage := make(resources.FlavorResourceQuantities)
	for fr, usage := range e.assignment.Usage.Quota {
		cqQuota := cq.QuotaFor(fr)
		if e.assignment.Borrowing > 0 {
			if cqQuota.BorrowingLimit == nil {
				reservedUsage[fr] = usage
			} else {
				reservedUsage[fr] = min(usage, cqQuota.Nominal+*cqQuota.BorrowingLimit-cq.ResourceNode.Usage[fr])
			}
		} else {
			reservedUsage[fr] = max(0, min(usage, cqQuota.Nominal-cq.ResourceNode.Usage[fr]))
		}
	}
	return reservedUsage
}

type partialAssignment struct {
	assignment        flavorassigner.Assignment
	preemptionTargets []*preemption.Target
}

// getAssignments 尝试为 Workload 分配资源风味和抢占目标。
// 先尝试完整分配，不行则尝试部分分配（PartialAdmission）。
func (s *Scheduler) getAssignments(log logr.Logger, wl *workload.Info, snap *cache.Snapshot) (flavorassigner.Assignment, []*preemption.Target) {
	assignment, targets := s.getInitialAssignments(log, wl, snap)
	cq := snap.ClusterQueue(wl.ClusterQueue)
	updateAssignmentForTAS(cq, wl, &assignment, targets)
	return assignment, targets
}

func (s *Scheduler) getInitialAssignments(log logr.Logger, wl *workload.Info, snap *cache.Snapshot) (flavorassigner.Assignment, []*preemption.Target) {
	cq := snap.ClusterQueue(wl.ClusterQueue)
	flvAssigner := flavorassigner.New(wl, cq, snap.ResourceFlavors, s.fairSharing.Enable, preemption.NewOracle(s.preemptor, snap))
	fullAssignment := flvAssigner.Assign(log, nil)

	arm := fullAssignment.RepresentativeMode()
	if arm == flavorassigner.Fit {
		return fullAssignment, nil
	}

	if arm == flavorassigner.Preempt {
		faPreemptionTargets := s.preemptor.GetTargets(log, *wl, fullAssignment, snap)
		if len(faPreemptionTargets) > 0 {
			return fullAssignment, faPreemptionTargets
		}
	}

	if features.Enabled(features.PartialAdmission) && wl.CanBePartiallyAdmitted() {
		reducer := flavorassigner.NewPodSetReducer(wl.Obj.Spec.PodSets, func(nextCounts []int32) (*partialAssignment, bool) {
			assignment := flvAssigner.Assign(log, nextCounts)
			mode := assignment.RepresentativeMode()
			if mode == flavorassigner.Fit {
				return &partialAssignment{assignment: assignment}, true
			}

			if mode == flavorassigner.Preempt {
				preemptionTargets := s.preemptor.GetTargets(log, *wl, assignment, snap)
				if len(preemptionTargets) > 0 {
					return &partialAssignment{assignment: assignment, preemptionTargets: preemptionTargets}, true
				}
			}
			return nil, false
		})
		if pa, found := reducer.Search(); found {
			return pa.assignment, pa.preemptionTargets
		}
	}
	return fullAssignment, nil
}

func (s *Scheduler) evictWorkloadAfterFailedTASReplacement(ctx context.Context, log logr.Logger, wl *kueue.Workload) error {
	log.V(3).Info("Evicting workload after failed try to find a node replacement; TASFailedNodeReplacementFailFast enabled")
	msg := fmt.Sprintf("Workload was evicted as there was no replacement for a failed node: %s", workload.NodeToReplace(wl))
	if err := workload.EvictWorkload(ctx, s.client, s.recorder, wl, kueue.WorkloadEvictedDueToNodeFailures, msg, s.clock); err != nil {
		return err
	}
	if err := workload.RemoveAnnotation(ctx, s.client, wl, kueuealpha.NodeToReplaceAnnotation); err != nil {
		return fmt.Errorf("failed to remove annotation for node replacement %s", kueuealpha.NodeToReplaceAnnotation)
	}
	return nil
}

func updateAssignmentForTAS(cq *cache.ClusterQueueSnapshot, wl *workload.Info, assignment *flavorassigner.Assignment, targets []*preemption.Target) {
	if features.Enabled(features.TopologyAwareScheduling) && assignment.RepresentativeMode() == flavorassigner.Preempt && (wl.IsRequestingTAS() || cq.IsTASOnly()) && !workload.HasTopologyAssignmentWithNodeToReplace(wl.Obj) {
		tasRequests := assignment.WorkloadsTopologyRequests(wl, cq)
		var tasResult cache.TASAssignmentsResult
		if len(targets) > 0 {
			var targetWorkloads []*workload.Info
			for _, target := range targets {
				targetWorkloads = append(targetWorkloads, target.WorkloadInfo)
			}
			revertUsage := cq.SimulateWorkloadRemoval(targetWorkloads)
			tasResult = cq.FindTopologyAssignmentsForWorkload(tasRequests)
			revertUsage()
		} else {
			// In this scenario we don't have any preemption candidates, yet we need
			// to reserve the TAS resources to avoid the situation when a lower
			// priority workload further in the queue gets admitted and preempted
			// in the next scheduling cycle by the waiting workload. To obtain
			// a TAS assignment for reserving the resources we run the algorithm
			// assuming the cluster is empty.
			tasResult = cq.FindTopologyAssignmentsForWorkload(tasRequests, cache.WithSimulateEmpty(true))
		}
		assignment.UpdateForTASResult(tasResult)
	}
}

// admit 将 entry 的资源分配写入 Workload，并异步更新到 apiserver。
// 先在本地 cache 假定准入，后续异步 applyAdmission。
func (s *Scheduler) admit(ctx context.Context, e *entry, cq *cache.ClusterQueueSnapshot) error {
	log := ctrl.LoggerFrom(ctx)
	newWorkload := e.Info.Obj.DeepCopy()
	admission := &kueue.Admission{
		ClusterQueue:      e.Info.ClusterQueue,
		PodSetAssignments: e.assignment.ToAPI(),
	}

	workload.SetQuotaReservation(newWorkload, admission, s.clock)
	if workload.HasAllChecks(newWorkload, workload.AdmissionChecksForWorkload(log, newWorkload, cq.AdmissionChecks)) {
		// sync Admitted, ignore the result since an API update is always done.
		_ = workload.SyncAdmittedCondition(newWorkload, s.clock.Now())
	}
	if err := s.cache.AssumeWorkload(log, newWorkload); err != nil {
		return err
	}
	e.status = assumed
	log.V(2).Info("Workload assumed in the cache")

	s.admissionRoutineWrapper.Run(func() {
		err := s.applyAdmission(ctx, newWorkload)
		if err == nil {
			// Record metrics and events for quota reservation and admission
			s.recordWorkloadAdmissionMetrics(newWorkload, e.Info.Obj, admission)

			log.V(2).Info("Workload successfully admitted and assigned flavors", "assignments", admission.PodSetAssignments)
			return
		}
		// Ignore errors because the workload or clusterQueue could have been deleted
		// by an event.
		_ = s.cache.ForgetWorkload(log, newWorkload)
		if apierrors.IsNotFound(err) {
			log.V(2).Info("Workload not admitted because it was deleted")
			return
		}

		log.Error(err, errCouldNotAdmitWL)
		s.requeueAndUpdate(ctx, *e)
	})

	return nil
}

type entryOrdering struct {
	entries          []entry
	workloadOrdering workload.Ordering
}

func (e entryOrdering) Len() int {
	return len(e.entries)
}

func (e entryOrdering) Swap(i, j int) {
	e.entries[i], e.entries[j] = e.entries[j], e.entries[i]
}

// Less 是排序依据
func (e entryOrdering) Less(i, j int) bool {
	a := e.entries[i]
	b := e.entries[j]

	// 首先处理已预留配额的工作负载。这样的工作负载
	// 如果这是它们的第二次通过，则可能被考虑。
	aHasQuota := workload.HasQuotaReservation(a.Info.Obj)
	bHasQuota := workload.HasQuotaReservation(b.Info.Obj)
	if aHasQuota != bHasQuota {
		return aHasQuota
	}

	// 1. 请求在名义配额下。
	aBorrows := a.assignment.Borrows()
	bBorrows := b.assignment.Borrows()
	if aBorrows != bBorrows {
		return aBorrows < bBorrows
	}

	// 2. 优先级较高的优先级（如果未禁用）。
	if features.Enabled(features.PrioritySortingWithinCohort) {
		p1 := priority.Priority(a.Info.Obj)
		p2 := priority.Priority(b.Info.Obj)
		if p1 != p2 {
			return p1 > p2
		}
	}

	// 3. FIFO。
	aComparisonTimestamp := e.workloadOrdering.GetQueueOrderTimestamp(a.Info.Obj)
	bComparisonTimestamp := e.workloadOrdering.GetQueueOrderTimestamp(b.Info.Obj)
	return aComparisonTimestamp.Before(bComparisonTimestamp)
}

// entryInterator 定义了条目返回的顺序。
// pop->nil IFF hasNext->False
type entryIterator interface {
	pop() *entry
	hasNext() bool
}

func makeIterator(ctx context.Context, entries []entry, workloadOrdering workload.Ordering, enableFairSharing bool) entryIterator {
	if enableFairSharing {
		return makeFairSharingIterator(ctx, entries, workloadOrdering)
	}
	return makeClassicalIterator(entries, workloadOrdering)
}

// classicalIterator 返回按以下顺序排序的条目：
// 1. 请求在名义配额之前。
// 2. 公平共享：优先级较低的 DominantResourceShare。
// 3. 优先级较高的优先级。
// 4. 驱逐或创建时间戳的 FIFO。
type classicalIterator struct {
	entries []entry
}

func (co *classicalIterator) hasNext() bool {
	return len(co.entries) > 0
}

func (co *classicalIterator) pop() *entry {
	head := &co.entries[0]
	co.entries = co.entries[1:]
	return head
}

func makeClassicalIterator(entries []entry, workloadOrdering workload.Ordering) *classicalIterator {
	sort.Sort(entryOrdering{
		entries:          entries,
		workloadOrdering: workloadOrdering,
	})
	return &classicalIterator{
		entries: entries,
	}
}

// requeueAndUpdate 负责将未准入的 Workload 重入队，并根据状态更新其 Admission/Pending 状态。
func (s *Scheduler) requeueAndUpdate(ctx context.Context, e entry) {
	log := ctrl.LoggerFrom(ctx)
	if e.status != notNominated && e.requeueReason == queue.RequeueReasonGeneric {
		// Failed after nomination is the only reason why a workload would be requeued downstream.
		e.requeueReason = queue.RequeueReasonFailedAfterNomination
	}

	if s.queues.QueueSecondPassIfNeeded(ctx, e.Info.Obj) {
		log.V(2).Info("Workload re-queued for second pass", "workload", klog.KObj(e.Info.Obj), "clusterQueue", klog.KRef("", string(e.Info.ClusterQueue)), "queue", klog.KRef(e.Info.Obj.Namespace, string(e.Info.Obj.Spec.QueueName)), "requeueReason", e.requeueReason, "status", e.status)
		s.recorder.Eventf(e.Info.Obj, corev1.EventTypeWarning, "SecondPassFailed", api.TruncateEventMessage(e.inadmissibleMsg))
		return
	}

	added := s.queues.RequeueWorkload(ctx, &e.Info, e.requeueReason)
	log.V(2).Info("Workload re-queued", "workload", klog.KObj(e.Info.Obj), "clusterQueue", klog.KRef("", string(e.Info.ClusterQueue)), "queue", klog.KRef(e.Info.Obj.Namespace, string(e.Info.Obj.Spec.QueueName)), "requeueReason", e.requeueReason, "added", added, "status", e.status)

	if e.status == notNominated || e.status == skipped {
		patch := workload.PrepareWorkloadPatch(e.Info.Obj, true, s.clock)
		reservationIsChanged := workload.UnsetQuotaReservationWithCondition(patch, "Pending", e.inadmissibleMsg, s.clock.Now())
		resourceRequestsIsChanged := workload.PropagateResourceRequests(patch, &e.Info)
		if reservationIsChanged || resourceRequestsIsChanged {
			if err := workload.ApplyAdmissionStatusPatch(ctx, s.client, patch); err != nil {
				log.Error(err, "Could not update Workload status")
			}
		}
		s.recorder.Eventf(e.Info.Obj, corev1.EventTypeWarning, "Pending", api.TruncateEventMessage(e.inadmissibleMsg))
	}
}

// recordWorkloadAdmissionMetrics 记录 Workload 准入过程的指标和事件。
func (s *Scheduler) recordWorkloadAdmissionMetrics(newWorkload, originalWorkload *kueue.Workload, admission *kueue.Admission) {
	waitTime := workload.QueuedWaitTime(newWorkload, s.clock)

	s.recordQuotaReservationMetrics(newWorkload, originalWorkload, admission, waitTime)
	s.recordWorkloadAdmissionEvents(newWorkload, originalWorkload, admission, waitTime)
}

// recordQuotaReservationMetrics 记录配额预留的指标和事件。
func (s *Scheduler) recordQuotaReservationMetrics(newWorkload, originalWorkload *kueue.Workload, admission *kueue.Admission, waitTime time.Duration) {
	if workload.HasQuotaReservation(originalWorkload) {
		return
	}

	s.recorder.Eventf(newWorkload, corev1.EventTypeNormal, "QuotaReserved", "配额在集群队列 %v 中已预留，等待时间为 %.0fs", admission.ClusterQueue, waitTime.Seconds())

	metrics.QuotaReservedWorkload(admission.ClusterQueue, waitTime)
	if features.Enabled(features.LocalQueueMetrics) {
		metrics.LocalQueueQuotaReservedWorkload(metrics.LQRefFromWorkload(newWorkload), waitTime)
	}
}

// recordWorkloadAdmissionEvents 记录 Workload 准入事件。
func (s *Scheduler) recordWorkloadAdmissionEvents(newWorkload, originalWorkload *kueue.Workload, admission *kueue.Admission, waitTime time.Duration) {
	if !workload.IsAdmitted(newWorkload) || workload.HasNodeToReplace(originalWorkload) {
		return
	}

	s.recorder.Eventf(newWorkload, corev1.EventTypeNormal, "Admitted", "集群队列 %v 已准入，等待时间为 0s", admission.ClusterQueue)
	metrics.AdmittedWorkload(admission.ClusterQueue, waitTime)

	if features.Enabled(features.LocalQueueMetrics) {
		metrics.LocalQueueAdmittedWorkload(metrics.LQRefFromWorkload(newWorkload), waitTime)
	}

	if len(newWorkload.Status.AdmissionChecks) > 0 {
		metrics.AdmissionChecksWaitTime(admission.ClusterQueue, 0)
		if features.Enabled(features.LocalQueueMetrics) {
			metrics.LocalQueueAdmissionChecksWaitTime(metrics.LQRefFromWorkload(newWorkload), 0)
		}
	}
}

// Start 启动调度器主循环，作为 controller-runtime 的 Runnable 实现。
func (s *Scheduler) Start(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithName("scheduler")
	ctx = ctrl.LoggerInto(ctx, log)
	go wait.UntilWithBackoff(ctx, s.schedule)
	return nil
}

func (s *Scheduler) applyAdmissionWithSSA(ctx context.Context, w *kueue.Workload) error {
	return workload.ApplyAdmissionStatus(ctx, s.client, w, false, s.clock)
}
