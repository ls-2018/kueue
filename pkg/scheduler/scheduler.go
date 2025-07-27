package scheduler

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sort"

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

// Scheduler 是 Kueue 的核心调度器结构体，负责管理队列、缓存、调度流程、抢占、事件记录等功能
// 该结构体封装了调度循环、资源分配、抢占、事件上报等核心逻辑
// 每个字段代表调度器的一个重要组件或配置
type Scheduler struct {
	queues                  *queue.Manager        // 队列管理器，负责管理所有本地队列和调度顺序
	cache                   *cache.Cache          // 缓存，保存集群资源、已调度和待调度工作负载的快照
	client                  client.Client         // Kubernetes 客户端，用于与 API Server 交互
	recorder                record.EventRecorder  // 事件记录器，用于向 K8s 事件系统发送事件
	admissionRoutineWrapper routine.Wrapper       // admission 例程包装器，用于异步处理 admission 状态变更
	preemptor               *preemption.Preemptor // 抢占器，负责调度中的抢占逻辑
	workloadOrdering        workload.Ordering     // 工作负载排序策略，决定调度顺序
	fairSharing             config.FairSharing    // 公平调度配置，控制资源分配公平性
	clock                   clock.Clock           // 时钟接口，便于测试和时间控制

	// schedulingCycle 标识自上次重启以来的调度尝试次数
	schedulingCycle int64

	// applyAdmission 是一个函数指针，用于将调度结果应用到 Workload 对象
	applyAdmission func(context.Context, *kueue.Workload) error
}

type options struct {
	podsReadyRequeuingTimestamp config.RequeuingTimestamp
	fairSharing                 config.FairSharing
	clock                       clock.Clock
}

// Option configures the reconciler.
type Option func(*options)

var defaultOptions = options{
	podsReadyRequeuingTimestamp: config.EvictionTimestamp,
	clock:                       realClock,
}

// WithPodsReadyRequeuingTimestamp 设置用于对因“PodsReady”条件而重新排队的作业进行排序的时间戳。
func WithPodsReadyRequeuingTimestamp(ts config.RequeuingTimestamp) Option {
	return func(o *options) {
		o.podsReadyRequeuingTimestamp = ts
	}
}

func New(queues *queue.Manager, cache *cache.Cache, cl client.Client, recorder record.EventRecorder, opts ...Option) *Scheduler {
	options := defaultOptions  // 初始化 options 为默认配置
	for _, opt := range opts { // 遍历所有可选参数
		opt(&options) // 应用每个 Option 配置到 options
	}
	wo := workload.Ordering{
		PodsReadyRequeuingTimestamp: options.podsReadyRequeuingTimestamp, // 设置 PodsReady 重新入队的时间戳
	}
	s := &Scheduler{ // 创建 Scheduler 实例
		fairSharing:             options.fairSharing,                                                  // 公平调度配置
		queues:                  queues,                                                               // 队列管理器
		cache:                   cache,                                                                // 缓存
		client:                  cl,                                                                   // k8s 客户端
		recorder:                recorder,                                                             // 事件记录器
		preemptor:               preemption.New(cl, wo, recorder, options.fairSharing, options.clock), // 抢占器
		admissionRoutineWrapper: routine.DefaultWrapper,                                               // admission 例程包装器
		workloadOrdering:        wo,                                                                   // 工作负载排序策略
		clock:                   options.clock,                                                        // 时钟
	}
	s.applyAdmission = s.applyAdmissionWithSSA // 设置 applyAdmission 方法
	return s                                   // 返回 Scheduler 实例
}

// Start 实现了 Runnable 接口，用于以控制器方式运行调度器。
func (s *Scheduler) Start(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithName("scheduler")
	ctx = ctrl.LoggerInto(ctx, log)
	go wait.UntilWithBackoff(ctx, s.schedule)
	return nil
}

// NeedLeaderElection 实现了 LeaderElectionRunnable 接口，使调度器以主节点选举模式运行。
func (s *Scheduler) NeedLeaderElection() bool {
	return true
}

func (s *Scheduler) setAdmissionRoutineWrapper(wrapper routine.Wrapper) {
	s.admissionRoutineWrapper = wrapper
}

func setSkipped(e *entry, inadmissibleMsg string) {
	e.status = skipped
	e.inadmissibleMsg = inadmissibleMsg
	// Reset assignment so that we retry all flavors
	// after skipping due to Fit no longer fitting,
	// or Preempt being skipped due to an overlapping
	// earlier admission.
	e.LastAssignment = nil
}

func reportSkippedPreemptions(p map[kueue.ClusterQueueReference]int) {
	for cqName, count := range p {
		metrics.AdmissionCyclePreemptionSkips.WithLabelValues(string(cqName)).Set(float64(count))
	}
}

func (s *Scheduler) schedule(ctx context.Context) wait.SpeedSignal {
	s.schedulingCycle++                                                          // 调度周期计数加一，表示新一轮调度开始
	log := ctrl.LoggerFrom(ctx).WithValues("schedulingCycle", s.schedulingCycle) // 获取带有当前调度周期的日志对象
	ctx = ctrl.LoggerInto(ctx, log)                                              // 将日志对象注入上下文

	headWorkloads := s.queues.Heads(ctx) // 获取所有队列的队首工作负载
	if len(headWorkloads) == 0 {         // 如果没有队首工作负载
		return wait.KeepGoing // 返回继续信号，调度循环继续
	}
	startTime := s.clock.Now() // 记录调度开始时间

	snapshot, err := s.cache.Snapshot(ctx) // 获取缓存快照
	if err != nil {                        // 如果快照获取失败
		log.Error(err, "failed to build snapshot for scheduling") // 记录错误日志
		return wait.SlowDown                                      // 返回减速信号，调度循环放慢
	}
	logSnapshotIfVerbose(log, snapshot) // 如果日志级别足够高，打印快照详细信息

	entries, inadmissibleEntries := s.nominate(ctx, headWorkloads, snapshot) // 计算每个工作负载的资源需求和可调度性

	iterator := makeIterator(ctx, entries, s.workloadOrdering, s.fairSharing.Enable) // 创建调度 entry 的迭代器

	preemptedWorkloads := make(preemption.PreemptedWorkloads)       // 记录已被抢占的工作负载集合
	skippedPreemptions := make(map[kueue.ClusterQueueReference]int) // 记录每个 ClusterQueue 被跳过抢占的次数
	for iterator.hasNext() {                                        // 遍历所有可调度 entry
		e := iterator.pop() // 取出下一个 entry

		cq := snapshot.ClusterQueue(e.ClusterQueue)                                                                // 获取 entry 对应的 ClusterQueue 快照
		log := log.WithValues("workload", klog.KObj(e.Obj), "clusterQueue", klog.KRef("", string(e.ClusterQueue))) // 日志中加入工作负载和 ClusterQueue 信息
		if cq.HasParent() {                                                                                        // 如果 ClusterQueue 有父 Cohort
			log = log.WithValues("parentCohort", klog.KRef("", string(cq.Parent().GetName())), "rootCohort", klog.KRef("", string(cq.Parent().Root().GetName()))) // 日志中加入 Cohort 信息
		}
		ctx := ctrl.LoggerInto(ctx, log) // 日志对象注入上下文

		mode := e.assignment.RepresentativeMode() // 获取当前 entry 的分配模式
		if mode == flavorassigner.NoFit {         // 如果没有合适的资源分配
			log.V(3).Info("Skipping workload as FlavorAssigner assigned NoFit mode") // 打印跳过信息
			continue                                                                 // 跳过该 entry
		}
		log.V(2).Info("Attempting to schedule workload") // 打印尝试调度信息

		if mode == flavorassigner.Preempt && len(e.preemptionTargets) == 0 { // 如果需要抢占但没有可抢占目标
			log.V(2).Info("Workload requires preemption, but there are no candidate workloads allowed for preemption", "preemption", cq.Preemption) // 打印无可抢占目标信息
			if !preemption.CanAlwaysReclaim(cq) {                                                                                                   // 如果不能保证后续能回收资源
				cq.AddUsage(resourcesToReserve(e, cq)) // 预留资源，防止被其他低优先级工作负载占用
			}
			continue // 跳过该 entry
		}

		if preemptedWorkloads.HasAny(e.preemptionTargets) { // 如果抢占目标与已抢占工作负载有重叠
			setSkipped(e, "Workload has overlapping preemption targets with another workload") // 标记为跳过
			skippedPreemptions[cq.Name]++                                                      // 记录跳过次数
			continue                                                                           // 跳过该 entry
		}

		usage := e.assignmentUsage()                                    // 计算 entry 的资源使用量
		if !fits(cq, &usage, preemptedWorkloads, e.preemptionTargets) { // 如果资源不再满足调度条件
			setSkipped(e, "Workload no longer fits after processing another workload") // 标记为跳过
			if mode == flavorassigner.Preempt {                                        // 如果是抢占模式
				skippedPreemptions[cq.Name]++ // 记录跳过次数
			}
			continue // 跳过该 entry
		}
		preemptedWorkloads.Insert(e.preemptionTargets) // 记录本次抢占的目标
		cq.AddUsage(usage)                             // 更新 ClusterQueue 的资源使用量

		if e.assignment.RepresentativeMode() == flavorassigner.Preempt { // 如果是抢占模式
			e.LastAssignment = nil                                                            // 清空上次分配，便于下次尝试所有资源类型
			preempted, err := s.preemptor.IssuePreemptions(ctx, &e.Info, e.preemptionTargets) // 发起抢占操作
			if err != nil {                                                                   // 如果抢占失败
				log.Error(err, "Failed to preempt workloads") // 记录错误日志
			}
			if preempted != 0 { // 如果有工作负载被抢占
				e.inadmissibleMsg += fmt.Sprintf(". Pending the preemption of %d workload(s)", preempted) // 记录抢占信息
				e.requeueReason = queue.RequeueReasonPendingPreemption                                    // 设置重新入队原因
			}
			continue // 跳过该 entry
		}
		if !s.cache.PodsReadyForAllAdmittedWorkloads(log) { // 如果所有已调度工作负载的 Pod 未全部就绪
			log.V(5).Info("Waiting for all admitted workloads to be in the PodsReady condition")                                                            // 打印等待信息
			workload.UnsetQuotaReservationWithCondition(e.Obj, "Waiting", "waiting for all admitted workloads to be in PodsReady condition", s.clock.Now()) // 设置等待条件
			if err := workload.ApplyAdmissionStatus(ctx, s.client, e.Obj, false, s.clock); err != nil {                                                     // 更新工作负载状态
				log.Error(err, "Could not update Workload status") // 记录错误日志
			}
			s.cache.WaitForPodsReady(ctx)                                                                 // 阻塞等待所有已调度工作负载的 Pod 就绪
			log.V(5).Info("Finished waiting for all admitted workloads to be in the PodsReady condition") // 打印等待完成信息
		}
		e.status = nominated                        // 标记 entry 已被提名调度
		if err := s.admit(ctx, e, cq); err != nil { // 尝试调度 entry
			e.inadmissibleMsg = fmt.Sprintf("Failed to admit workload: %v", err) // 记录调度失败原因
		}
	}

	result := metrics.AdmissionResultInadmissible // 默认调度结果为不可调度
	for _, e := range entries {                   // 遍历所有 entry
		logAdmissionAttemptIfVerbose(log, &e) // 打印调度尝试日志（如日志级别足够高）
		if e.status != assumed {              // 如果 entry 未被假定调度
			s.requeueAndUpdate(ctx, e) // 重新入队并更新状态
		} else {
			result = metrics.AdmissionResultSuccess // 有 entry 被成功调度，更新结果为成功
		}
	}
	for _, e := range inadmissibleEntries { // 遍历所有不可调度 entry
		logAdmissionAttemptIfVerbose(log, &e) // 打印调度尝试日志
		s.requeueAndUpdate(ctx, e)            // 重新入队并更新状态
	}

	reportSkippedPreemptions(skippedPreemptions)               // 上报跳过的抢占信息
	metrics.AdmissionAttempt(result, s.clock.Since(startTime)) // 上报本轮调度的耗时和结果
	if result != metrics.AdmissionResultSuccess {              // 如果没有 entry 被成功调度
		return wait.SlowDown // 返回减速信号
	}
	return wait.KeepGoing // 返回继续信号
}

// entryStatus 表示工作负载在调度过程中的状态
// 用于标识 entry 在调度周期中的不同阶段
// 可能的状态有：nominated（已提名）、skipped（本轮被跳过）、assumed（已假定调度）、notNominated（未被提名）
type entryStatus string

const (
	// nominated 表示该工作负载已被提名进行调度
	nominated entryStatus = "nominated"
	// skipped 表示该工作负载在本轮调度中被跳过
	skipped entryStatus = "skipped"
	// assumed 表示该工作负载已被假定为调度成功
	assumed entryStatus = "assumed"
	// notNominated 表示该工作负载未被提名进行调度
	notNominated entryStatus = ""
)

// entry 结构体保存了一个工作负载被 ClusterQueue 调度所需的所有信息和调度中间状态
// 主要用于调度循环中的调度决策、资源分配、抢占、重试等逻辑
// 每个字段代表调度过程中的一个关键属性或中间结果
type entry struct {
	workload.Info                                    // 工作负载的详细信息，包括 API 对象、资源使用、分配的资源类型等
	assignment           flavorassigner.Assignment   // 资源分配结果，包括分配的资源类型、数量、借用等
	status               entryStatus                 // 当前 entry 的调度状态（如已提名、跳过、假定调度等）
	inadmissibleMsg      string                      // 不可调度原因的详细信息
	requeueReason        queue.RequeueReason         // 需要重新入队的原因
	preemptionTargets    []*preemption.Target        // 抢占目标列表（如果需要抢占）
	clusterQueueSnapshot *cache.ClusterQueueSnapshot // 当前 entry 对应的 ClusterQueue 快照，用于资源校验和分配
}

func (e *entry) assignmentUsage() workload.Usage {
	return netUsage(e, func() resources.FlavorResourceQuantities {
		return e.assignment.Usage.Quota
	})
}

func fits(cq *cache.ClusterQueueSnapshot, usage *workload.Usage, preemptedWorkloads preemption.PreemptedWorkloads, newTargets []*preemption.Target) bool {
	workloads := slices.Collect(maps.Values(preemptedWorkloads))
	for _, target := range newTargets {
		workloads = append(workloads, target.WorkloadInfo)
	}
	revertUsage := cq.SimulateWorkloadRemoval(workloads)
	defer revertUsage()
	return cq.Fits(*usage)
}

// resourcesToReserve calculates how much of the available resources in cq/cohort assignment should be reserved.
// resourcesToReserve 计算在 cq/cohort 分配中应保留多少可用资源。
func resourcesToReserve(e *entry, cq *cache.ClusterQueueSnapshot) workload.Usage {
	return netUsage(e, func() resources.FlavorResourceQuantities {
		return quotaResourcesToReserve(e, cq)
	})
}

// netUsage calculates the net usage for quota and TAS to reserve
// netUsage 计算配额和 TAS 需要保留的净资源用量。
func netUsage(e *entry, netQuota func() resources.FlavorResourceQuantities) workload.Usage {
	result := workload.Usage{}
	if features.Enabled(features.TopologyAwareScheduling) {
		result.TAS = e.assignment.ComputeTASNetUsage(e.Obj.Status.Admission)
	}
	if !workload.HasQuotaReservation(e.Obj) {
		result.Quota = netQuota()
	}
	return result
}

func quotaResourcesToReserve(e *entry, cq *cache.ClusterQueueSnapshot) resources.FlavorResourceQuantities {
	if e.assignment.RepresentativeMode() != flavorassigner.Preempt { // 如果不是抢占模式
		return e.assignment.Usage.Quota // 直接返回分配的资源用量
	}
	reservedUsage := make(resources.FlavorResourceQuantities) // 初始化预留资源用量映射
	for fr, usage := range e.assignment.Usage.Quota {         // 遍历每种资源类型
		cqQuota := cq.QuotaFor(fr)      // 获取 ClusterQueue 对应资源的配额信息
		if e.assignment.Borrowing > 0 { // 如果有借用
			if cqQuota.BorrowingLimit == nil { // 如果没有借用上限
				reservedUsage[fr] = usage // 全部预留
			} else {
				reservedUsage[fr] = min(usage, cqQuota.Nominal+*cqQuota.BorrowingLimit-cq.ResourceNode.Usage[fr]) // 预留不超过名义配额+借用上限-当前已用
			}
		} else {
			reservedUsage[fr] = max(0, min(usage, cqQuota.Nominal-cq.ResourceNode.Usage[fr])) // 只预留未借用部分，且不小于0
		}
	}
	return reservedUsage // 返回最终预留的资源用量
}

// partialAssignment 结构体用于表示部分调度时的资源分配结果和抢占目标
// 主要用于支持部分 PodSet 可以被调度的场景
type partialAssignment struct {
	assignment        flavorassigner.Assignment // 部分调度的资源分配结果
	preemptionTargets []*preemption.Target      // 部分调度对应的抢占目标列表
}

// admit 将调度的 clusterQueue 和资源类型写入 entry 的工作负载，并在缓存中假定后，异步更新 apiserver 中的对象。
func (s *Scheduler) admit(ctx context.Context, e *entry, cq *cache.ClusterQueueSnapshot) error {
	log := ctrl.LoggerFrom(ctx)     // 获取日志对象
	newWorkload := e.Obj.DeepCopy() // 深拷贝工作负载对象，避免直接修改原对象
	admission := &kueue.Admission{  // 构造 Admission 对象
		ClusterQueue:      e.ClusterQueue,       // 设置调度的 ClusterQueue
		PodSetAssignments: e.assignment.ToAPI(), // 设置 PodSet 的资源分配
	}

	workload.SetQuotaReservation(newWorkload, admission, s.clock)                                                      // 设置配额预留信息
	if workload.HasAllChecks(newWorkload, workload.AdmissionChecksForWorkload(log, newWorkload, cq.AdmissionChecks)) { // 检查 AdmissionChecks 是否全部通过
		_ = workload.SyncAdmittedCondition(newWorkload, s.clock.Now()) // 同步 Admitted 条件（忽略返回值）
	}
	if err := s.cache.AssumeWorkload(log, newWorkload); err != nil { // 假定工作负载已调度，写入缓存
		return err // 如果失败则返回错误
	}
	e.status = assumed                             // 标记 entry 状态为已假定调度
	log.V(2).Info("Workload assumed in the cache") // 打印日志

	s.admissionRoutineWrapper.Run(func() { // 异步执行 admission 操作
		err := s.applyAdmission(ctx, newWorkload) // 调用 applyAdmission 方法，更新 apiserver 状态
		if err == nil {                           // 如果更新成功
			waitTime := workload.QueuedWaitTime(newWorkload, s.clock) // 计算排队等待时间
			if !workload.HasQuotaReservation(e.Obj) {                 // 如果原对象没有配额预留
				s.recorder.Eventf(newWorkload, corev1.EventTypeNormal, "QuotaReserved", "Quota reserved in ClusterQueue %v, wait time since queued was %.0fs", admission.ClusterQueue, waitTime.Seconds()) // 发送配额预留事件
				metrics.QuotaReservedWorkload(admission.ClusterQueue, waitTime)                                                                                                                            // 上报配额预留指标
				if features.Enabled(features.LocalQueueMetrics) {                                                                                                                                          // 如果启用本地队列指标
					metrics.LocalQueueQuotaReservedWorkload(metrics.LQRefFromWorkload(newWorkload), waitTime) // 上报本地队列配额预留指标
				}
			}
			if workload.IsAdmitted(newWorkload) && !workload.HasNodeToReplace(e.Obj) { // 如果已被正式调度且没有节点替换
				s.recorder.Eventf(newWorkload, corev1.EventTypeNormal, "Admitted", "Admitted by ClusterQueue %v, wait time since reservation was 0s", admission.ClusterQueue) // 发送调度成功事件
				metrics.AdmittedWorkload(admission.ClusterQueue, waitTime)                                                                                                    // 上报调度成功指标
				if features.Enabled(features.LocalQueueMetrics) {                                                                                                             // 如果启用本地队列指标
					metrics.LocalQueueAdmittedWorkload(metrics.LQRefFromWorkload(newWorkload), waitTime) // 上报本地队列调度成功指标
				}
				if len(newWorkload.Status.AdmissionChecks) > 0 { // 如果有 AdmissionChecks
					metrics.AdmissionChecksWaitTime(admission.ClusterQueue, 0) // 上报 AdmissionChecks 等待时间
					if features.Enabled(features.LocalQueueMetrics) {          // 如果启用本地队列指标
						metrics.LocalQueueAdmissionChecksWaitTime(metrics.LQRefFromWorkload(newWorkload), 0) // 上报本地队列 AdmissionChecks 等待时间
					}
				}
			}
			log.V(2).Info("Workload successfully admitted and assigned flavors", "assignments", admission.PodSetAssignments) // 打印调度成功日志
			return
		}
		_ = s.cache.ForgetWorkload(log, newWorkload) // 如果失败，忘记缓存中的该工作负载
		if apierrors.IsNotFound(err) {               // 如果对象已被删除
			log.V(2).Info("Workload not admitted because it was deleted") // 打印日志
			return
		}

		log.Error(err, errCouldNotAdmitWL) // 打印错误日志
		s.requeueAndUpdate(ctx, *e)        // 重新入队并更新状态
	})

	return nil // 返回 nil 表示成功
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

// Less is the ordering criteria
// Less 是排序标准。
func (e entryOrdering) Less(i, j int) bool {
	a := e.entries[i]
	b := e.entries[j]

	// First process workloads which already have quota reserved. Such workload
	// may be considered if this is their second pass.
	aHasQuota := workload.HasQuotaReservation(a.Obj)
	bHasQuota := workload.HasQuotaReservation(b.Obj)
	if aHasQuota != bHasQuota {
		return aHasQuota
	}

	// 1. Request under nominal quota.
	aBorrows := a.assignment.Borrows()
	bBorrows := b.assignment.Borrows()
	if aBorrows != bBorrows {
		return aBorrows < bBorrows
	}

	// 2. Higher priority first if not disabled.
	if features.Enabled(features.PrioritySortingWithinCohort) {
		p1 := priority.Priority(a.Obj)
		p2 := priority.Priority(b.Obj)
		if p1 != p2 {
			return p1 > p2
		}
	}

	// 3. FIFO.
	aComparisonTimestamp := e.workloadOrdering.GetQueueOrderTimestamp(a.Obj)
	bComparisonTimestamp := e.workloadOrdering.GetQueueOrderTimestamp(b.Obj)
	return aComparisonTimestamp.Before(bComparisonTimestamp)
}

// entryInterator defines order that entries are returned.
// pop->nil IFF hasNext->False
// entryInterator 定义了 entry 返回的顺序。
// 只有 hasNext 为 false 时，pop 才会返回 nil。
type entryIterator interface {
	pop() *entry
	hasNext() bool
}

func (s *Scheduler) requeueAndUpdate(ctx context.Context, e entry) {
	log := ctrl.LoggerFrom(ctx)                                                    // 获取日志对象
	if e.status != notNominated && e.requeueReason == queue.RequeueReasonGeneric { // 如果 entry 已被提名且重入队原因为通用
		// 被提名后失败是工作负载会被下游重新入队的唯一原因。
		e.requeueReason = queue.RequeueReasonFailedAfterNomination // 设置为被提名后失败
	}

	if s.queues.QueueSecondPassIfNeeded(ctx, e.Obj) { // 如果需要二次调度
		log.V(2).Info("Workload re-queued for second pass", "workload", klog.KObj(e.Obj), "clusterQueue", klog.KRef("", string(e.ClusterQueue)), "queue", klog.KRef(e.Obj.Namespace, string(e.Obj.Spec.QueueName)), "requeueReason", e.requeueReason, "status", e.status) // 打印日志
		s.recorder.Eventf(e.Obj, corev1.EventTypeWarning, "SecondPassFailed", api.TruncateEventMessage(e.inadmissibleMsg))                                                                                                                                                // 发送二次调度失败事件
		return                                                                                                                                                                                                                                                            // 直接返回
	}

	added := s.queues.RequeueWorkload(ctx, &e.Info, e.requeueReason)                                                                                                                                                                                                  // 将 entry 重新入队
	log.V(2).Info("Workload re-queued", "workload", klog.KObj(e.Obj), "clusterQueue", klog.KRef("", string(e.ClusterQueue)), "queue", klog.KRef(e.Obj.Namespace, string(e.Obj.Spec.QueueName)), "requeueReason", e.requeueReason, "added", added, "status", e.status) // 打印日志

	if e.status == notNominated || e.status == skipped { // 如果 entry 未被提名或被跳过
		patch := workload.PrepareWorkloadPatch(e.Obj, true, s.clock)                                                            // 构造 patch 对象
		reservationIsChanged := workload.UnsetQuotaReservationWithCondition(patch, "Pending", e.inadmissibleMsg, s.clock.Now()) // 取消配额预留并设置 Pending 条件
		resourceRequestsIsChanged := workload.PropagateResourceRequests(patch, &e.Info)                                         // 传播资源请求
		if reservationIsChanged || resourceRequestsIsChanged {                                                                  // 如果有变更
			if err := workload.ApplyAdmissionStatusPatch(ctx, s.client, patch); err != nil { // 应用 patch 更新 admission 状态
				log.Error(err, "Could not update Workload status") // 打印错误日志
			}
		}
		s.recorder.Eventf(e.Obj, corev1.EventTypeWarning, "Pending", api.TruncateEventMessage(e.inadmissibleMsg)) // 发送 Pending 事件
	}
}
func (s *Scheduler) getAssignments(log logr.Logger, wl *workload.Info, snap *cache.Snapshot) (flavorassigner.Assignment, []*preemption.Target) {
	assignment, targets := s.getInitialAssignments(log, wl, snap) // ✅
	cq := snap.ClusterQueue(wl.ClusterQueue)
	updateAssignmentForTAS(cq, wl, &assignment, targets)
	return assignment, targets
}

func WithFairSharing(fs *config.FairSharing) Option {
	return func(o *options) {
		if fs != nil {
			o.fairSharing = *fs
		}
	}
}

func (s *Scheduler) getInitialAssignments(log logr.Logger, wl *workload.Info, snap *cache.Snapshot) (flavorassigner.Assignment, []*preemption.Target) {
	cq := snap.ClusterQueue(wl.ClusterQueue)                                                                                       // 获取当前工作负载对应的 ClusterQueue 快照
	flvAssigner := flavorassigner.New(wl, cq, snap.ResourceFlavors, s.fairSharing.Enable, preemption.NewOracle(s.preemptor, snap)) // 创建资源分配器

	fullAssignment := flvAssigner.Assign(log, nil) // ✅ 尝试为所有 PodSet 分配资源

	arm := fullAssignment.RepresentativeMode() // 获取分配模式
	if arm == flavorassigner.Fit {             // 如果所有资源都能直接分配
		return fullAssignment, nil // 返回分配结果，无需抢占
	}

	if arm == flavorassigner.Preempt { // 如果需要抢占
		faPreemptionTargets := s.preemptor.GetTargets(log, *wl, fullAssignment, snap) // 获取可抢占的目标
		if len(faPreemptionTargets) > 0 {                                             // 如果有可抢占目标
			return fullAssignment, faPreemptionTargets // 返回分配结果和抢占目标
		}
	}

	if features.Enabled(features.PartialAdmission) && wl.CanBePartiallyAdmitted() { // 如果支持部分调度且工作负载允许部分调度  ✅
		reducer := flavorassigner.NewPodSetReducer(
			wl.Obj.Spec.PodSets,
			func(nextCounts []int32) (*partialAssignment, bool) { // 创建 PodSet 数量缩减器
				assignment := flvAssigner.Assign(log, nextCounts) // 尝试为缩减后的 PodSet 分配资源
				mode := assignment.RepresentativeMode()           // 获取分配模式
				if mode == flavorassigner.Fit {                   // 如果缩减后可以直接分配
					return &partialAssignment{assignment: assignment}, true // 返回部分分配结果
				}
				if mode == flavorassigner.Preempt { // 如果缩减后需要抢占
					preemptionTargets := s.preemptor.GetTargets(log, *wl, assignment, snap) // 获取可抢占目标
					if len(preemptionTargets) > 0 {                                         // 如果有可抢占目标
						return &partialAssignment{assignment: assignment, preemptionTargets: preemptionTargets}, true // 返回部分分配和抢占目标
					}
				}
				return nil, false // 否则继续缩减
			},
		)
		if pa, found := reducer.Search(); found { // 搜索可行的部分分配方案
			return pa.assignment, pa.preemptionTargets // 返回部分分配和抢占目标
		}
	}
	return fullAssignment, nil // 返回原始分配结果，未找到可行方案
}

func updateAssignmentForTAS(cq *cache.ClusterQueueSnapshot, wl *workload.Info, assignment *flavorassigner.Assignment, targets []*preemption.Target) {
	if features.Enabled(features.TopologyAwareScheduling) && assignment.RepresentativeMode() == flavorassigner.Preempt && (wl.IsRequestingTAS() || cq.IsTASOnly()) && !workload.HasTopologyAssignmentWithNodeToReplace(wl.Obj) {
		tasRequests := assignment.WorkloadsTopologyRequests(wl, cq) // 获取工作负载的拓扑请求
		var tasResult cache.TASAssignmentsResult                    // 定义 TAS 分配结果变量
		if len(targets) > 0 {                                       // 如果有抢占目标
			var targetWorkloads []*workload.Info // 定义抢占目标工作负载切片
			for _, target := range targets {     // 遍历所有抢占目标
				targetWorkloads = append(targetWorkloads, target.WorkloadInfo) // 添加到切片
			}
			revertUsage := cq.SimulateWorkloadRemoval(targetWorkloads)                 // 模拟移除抢占目标后的资源使用
			tasResult = cq.FindTopologyAssignmentsForWorkload(tasRequests, false, nil) // 查找拓扑分配
			revertUsage()                                                              // 恢复资源使用
		} else {
			// 在没有抢占候选的情况下，需要预留 TAS 资源，防止低优先级工作负载被调度后又被抢占。
			// 这里假设集群为空，运行算法获得 TAS 分配用于资源预留。
			tasResult = cq.FindTopologyAssignmentsForWorkload(tasRequests, true, nil) // 查找拓扑分配（假设集群为空）
		}
		assignment.UpdateForTASResult(tasResult) // 更新分配结果
	}
}

// nominate 返回如果被快照中的 clusterQueue 调度时，工作负载的资源需求（资源类型、借用等）。第二个返回值是不可调度的 entry 列表。
func (s *Scheduler) nominate(ctx context.Context, workloads []workload.Info, snap *cache.Snapshot) ([]entry, []entry) {
	log := ctrl.LoggerFrom(ctx)                 // 获取日志对象
	entries := make([]entry, 0, len(workloads)) // 初始化可调度 entry 列表
	var inadmissibleEntries []entry             // 初始化不可调度 entry 列表
	for _, w := range workloads {               // 遍历所有待调度工作负载
		log := log.WithValues("workload", klog.KObj(w.Obj), "clusterQueue", klog.KRef("", string(w.ClusterQueue))) // 日志中加入工作负载和 ClusterQueue 信息
		ns := corev1.Namespace{}                                                                                   // 用于存储命名空间对象
		e := entry{Info: w}                                                                                        // 构造 entry 对象
		e.clusterQueueSnapshot = snap.ClusterQueue(w.ClusterQueue)                                                 // 获取对应的 ClusterQueue 快照
		if !workload.NeedsSecondPass(w.Obj) && s.cache.IsAssumedOrAdmittedWorkload(w) {                            // 如果不需要二次调度且已在缓存中
			log.Info("Workload skipped from admission because it's already accounted in cache, and it does not need second pass", "workload", klog.KObj(w.Obj)) // 打印跳过信息
			continue                                                                                                                                            // 跳过该 entry
		} else if workload.HasRetryChecks(w.Obj) || workload.HasRejectedChecks(w.Obj) { // 如果 admission check 失败
			e.inadmissibleMsg = "The workload has failed admission checks" // 记录失败原因
		} else if snap.InactiveClusterQueueSets.Has(w.ClusterQueue) { // 如果 ClusterQueue 不可用
			e.inadmissibleMsg = fmt.Sprintf("ClusterQueue %s is inactive", w.ClusterQueue) // 记录失败原因
		} else if e.clusterQueueSnapshot == nil { // 如果找不到 ClusterQueue 快照
			e.inadmissibleMsg = fmt.Sprintf("ClusterQueue %s not found", w.ClusterQueue) // 记录失败原因
		} else if err := s.client.Get(ctx, types.NamespacedName{Name: w.Obj.Namespace}, &ns); err != nil { // 获取命名空间失败
			e.inadmissibleMsg = fmt.Sprintf("Could not obtain workload namespace: %v", err) // 记录失败原因
		} else if !e.clusterQueueSnapshot.NamespaceSelector.Matches(labels.Set(ns.Labels)) { // 命名空间标签不匹配
			e.inadmissibleMsg = "Workload namespace doesn't match ClusterQueue selector" // 记录失败原因
			e.requeueReason = queue.RequeueReasonNamespaceMismatch                       // 设置重新入队原因
		} else if err := workload.ValidateResources(&w); err != nil { // 资源校验失败
			e.inadmissibleMsg = fmt.Sprintf("%s: %v", errInvalidWLResources, err.ToAggregate()) // 记录失败原因
		} else if err := workload.ValidateLimitRange(ctx, s.client, &w); err != nil { // LimitRange 校验失败
			e.inadmissibleMsg = fmt.Sprintf("%s: %v", errLimitRangeConstraintsUnsatisfiedResources, err.ToAggregate()) // 记录失败原因
		} else {
			e.assignment, e.preemptionTargets = s.getAssignments(log, &e.Info, snap) // 获取资源分配和抢占目标
			e.inadmissibleMsg = e.assignment.Message()                               // 记录分配信息
			e.LastAssignment = &e.assignment.LastState                               // 记录上一次分配状态
			entries = append(entries, e)                                             // 加入可调度 entry 列表
			continue                                                                 // 进入下一个工作负载
		}
		inadmissibleEntries = append(inadmissibleEntries, e) // 加入不可调度 entry 列表
	}
	return entries, inadmissibleEntries // 返回可调度和不可调度 entry 列表
}

// classicalIterator returns entries ordered on:
// 1. request under nominal quota before borrowing.
// 2. Fair Sharing: lower DominantResourceShare first.
// 3. higher priority first.
// 4. FIFO on eviction or creation timestamp.
// classicalIterator 返回按如下顺序排列的 entry：
// 1. 先处理未借用的名义配额请求。
// 2. 公平共享：优先处理主导资源份额较低的。
// 3. 优先级高的优先。
// 4. 按驱逐或创建时间 FIFO。
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
func makeIterator(ctx context.Context, entries []entry, workloadOrdering workload.Ordering, enableFairSharing bool) entryIterator {
	if enableFairSharing {
		return makeFairSharingIterator(ctx, entries, workloadOrdering)
	}
	return makeClassicalIterator(entries, workloadOrdering)
}
func (s *Scheduler) applyAdmissionWithSSA(ctx context.Context, w *kueue.Workload) error {
	return workload.ApplyAdmissionStatus(ctx, s.client, w, false, s.clock)
}
