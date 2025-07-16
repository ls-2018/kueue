package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/features"
	"sigs.k8s.io/kueue/pkg/over_constants"
)

type AdmissionResult string
type ClusterQueueStatus string

type LocalQueueReference struct {
	Name      kueue.LocalQueueName
	Namespace string
}

const (
	AdmissionResultSuccess      AdmissionResult = "success"
	AdmissionResultInadmissible AdmissionResult = "inadmissible"

	PendingStatusActive       = "active"
	PendingStatusInadmissible = "inadmissible"

	// CQStatusPending means the ClusterQueue is accepted but not yet active,
	// this can be because of:
	// - a missing ResourceFlavor referenced by the ClusterQueue
	// - a missing or inactive AdmissionCheck referenced by the ClusterQueue
	// - the ClusterQueue is stopped
	// In this state, the ClusterQueue can't admit new workloads and its quota can't be borrowed
	// by other active ClusterQueues in the cohort.
	// CQStatusPending 表示 ClusterQueue 已被接受但尚未激活，可能原因包括：
	// - ClusterQueue 引用的 ResourceFlavor 缺失
	// - ClusterQueue 引用的 AdmissionCheck 缺失或未激活
	// - ClusterQueue 已停止
	// 在此状态下，ClusterQueue 不能接收新的 workload，其配额也不能被 cohort 中其他激活的 ClusterQueue 借用。
	CQStatusPending ClusterQueueStatus = "pending"
	// CQStatusActive means the ClusterQueue can admit new workloads and its quota
	// can be borrowed by other ClusterQueues in the cohort.
	// CQStatusActive 表示 ClusterQueue 可以接收新的 workload，其配额也可以被 cohort 中其他 ClusterQueue 借用。
	CQStatusActive ClusterQueueStatus = "active"
	// CQStatusTerminating means the clusterQueue is in pending deletion.
	// CQStatusTerminating 表示 ClusterQueue 正在等待删除。
	CQStatusTerminating ClusterQueueStatus = "terminating"
)

var (
	CQStatuses = []ClusterQueueStatus{CQStatusPending, CQStatusActive, CQStatusTerminating}

	// Metrics tied to the scheduler

	AdmissionAttemptsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: over_constants.KueueName,
			Name:      "admission_attempts_total",
			Help: `尝试接收 workload 的总次数。
每次尝试可能会接收多个 workload。
标签 'result' 可能的取值：
- 'success'：至少有一个 workload 被接收。
- 'inadmissible'：没有 workload 被接收。`,
		}, []string{"result"},
	)

	admissionAttemptDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: over_constants.KueueName,
			Name:      "admission_attempt_duration_seconds",
			Help: `一次接收尝试的延迟。
标签 'result' 可能的取值：
- 'success'：至少有一个 workload 被接收。
- 'inadmissible'：没有 workload 被接收。`,
		}, []string{"result"},
	)

	AdmissionCyclePreemptionSkips = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: over_constants.KueueName,
			Name:      "admission_cycle_preemption_skips",
			Help:      "ClusterQueue 中获得抢占候选但因同一周期内其他 ClusterQueue 也需要相同资源而被跳过的 Workload 数量。",
		}, []string{"cluster_queue"},
	)

	// Metrics tied to the queue system.
	// 与队列系统相关的指标。

	PendingWorkloads = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: over_constants.KueueName,
			Name:      "pending_workloads",
			Help: `每个 'cluster_queue' 和 'status' 下的待处理 workload 数量。
'status' 可能的取值：
- "active"：workload 在接收队列中。
- "inadmissible"：这些 workload 的接收尝试失败，只有当集群条件变化使其可接收时才会重试。`,
		}, []string{"cluster_queue", "status"},
	)

	LocalQueuePendingWorkloads = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: over_constants.KueueName,
			Name:      "local_queue_pending_workloads",
			Help: `每个 'local_queue' 和 'status' 下的待处理 workload 数量。
'status' 可能的取值：
- "active"：workload 在接收队列中。
- "inadmissible"：这些 workload 的接收尝试失败，只有当集群条件变化使其可接收时才会重试。`,
		}, []string{"name", "namespace", "status"},
	)

	QuotaReservedWorkloadsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: over_constants.KueueName,
			Name:      "quota_reserved_workloads_total",
			Help:      "每个 'cluster_queue' 下被预留配额的 workload 总数。",
		}, []string{"cluster_queue"},
	)

	LocalQueueQuotaReservedWorkloadsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: over_constants.KueueName,
			Name:      "local_queue_quota_reserved_workloads_total",
			Help:      "每个 'local_queue' 下被预留配额的 workload 总数。",
		}, []string{"name", "namespace"},
	)

	quotaReservedWaitTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: over_constants.KueueName,
			Name:      "quota_reserved_wait_time_seconds",
			Help:      "每个 'cluster_queue' 下，workload 从创建或重新排队到获得配额预留的时间。",
			Buckets:   generateExponentialBuckets(14),
		}, []string{"cluster_queue"},
	)

	localQueueQuotaReservedWaitTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: over_constants.KueueName,
			Name:      "local_queue_quota_reserved_wait_time_seconds",
			Help:      "每个 'local_queue' 下，workload 从创建或重新排队到获得配额预留的时间。",
			Buckets:   generateExponentialBuckets(14),
		}, []string{"name", "namespace"},
	)

	AdmittedWorkloadsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: over_constants.KueueName,
			Name:      "admitted_workloads_total",
			Help:      "每个 'cluster_queue' 下已接收的 workload 总数。",
		}, []string{"cluster_queue"},
	)

	LocalQueueAdmittedWorkloadsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: over_constants.KueueName,
			Name:      "local_queue_admitted_workloads_total",
			Help:      "每个 'local_queue' 下已接收的 workload 总数。",
		}, []string{"name", "namespace"},
	)

	admissionWaitTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: over_constants.KueueName,
			Name:      "admission_wait_time_seconds",
			Help:      "每个 'cluster_queue' 下，workload 从创建或重新排队到被接收的时间。",
			Buckets:   generateExponentialBuckets(14),
		}, []string{"cluster_queue"},
	)

	queuedUntilReadyWaitTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: over_constants.KueueName,
			Name:      "ready_wait_time_seconds",
			Help:      "每个 'cluster_queue' 下，workload 从创建或重新排队到 ready 的时间。",
			Buckets:   generateExponentialBuckets(14),
		}, []string{"cluster_queue"},
	)

	admittedUntilReadyWaitTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: over_constants.KueueName,
			Name:      "admitted_until_ready_wait_time_seconds",
			Help:      "每个 'cluster_queue' 下，workload 从被接收到 ready 的时间。",
			Buckets:   generateExponentialBuckets(14),
		}, []string{"cluster_queue"},
	)

	localQueueAdmissionWaitTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: over_constants.KueueName,
			Name:      "local_queue_admission_wait_time_seconds",
			Help:      "每个 'local_queue' 下，workload 从创建或重新排队到被接收的时间。",
			Buckets:   generateExponentialBuckets(14),
		}, []string{"name", "namespace"},
	)

	admissionChecksWaitTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: over_constants.KueueName,
			Name:      "admission_checks_wait_time_seconds",
			Help:      "每个 'cluster_queue' 下，workload 获得配额预留到被接收的时间。",
			Buckets:   generateExponentialBuckets(14),
		}, []string{"cluster_queue"},
	)

	localQueueAdmissionChecksWaitTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: over_constants.KueueName,
			Name:      "local_queue_admission_checks_wait_time_seconds",
			Help:      "每个 'local_queue' 下，workload 获得配额预留到被接收的时间。",
			Buckets:   generateExponentialBuckets(14),
		}, []string{"name", "namespace"},
	)

	localQueueQueuedUntilReadyWaitTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: over_constants.KueueName,
			Name:      "local_queue_ready_wait_time_seconds",
			Help:      "每个 'local_queue' 下，workload 从创建或重新排队到 ready 的时间。",
			Buckets:   generateExponentialBuckets(14),
		}, []string{"name", "namespace"},
	)

	localQueueAdmittedUntilReadyWaitTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: over_constants.KueueName,
			Name:      "local_queue_admitted_until_ready_wait_time_seconds",
			Help:      "每个 'local_queue' 下，workload 从被接收到 ready 的时间。",
			Buckets:   generateExponentialBuckets(14),
		}, []string{"name", "namespace"},
	)

	EvictedWorkloadsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: over_constants.KueueName,
			Name:      "evicted_workloads_total",
			Help: `每个 'cluster_queue' 下被驱逐的 workload 数量。
标签 'reason' 可能的取值：
- "Preempted"：为更高优先级 workload 或名义配额回收释放资源而驱逐。
- "PodsReadyTimeout"：因 PodsReady 超时而驱逐。
- "AdmissionCheck"：因至少一个 admission check 变为 False 而驱逐。
- "ClusterQueueStopped"：因 ClusterQueue 停止而驱逐。
- "Deactivated"：因 spec.active 设为 false 而驱逐。`,
		}, []string{"cluster_queue", "reason"},
	)

	LocalQueueEvictedWorkloadsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: over_constants.KueueName,
			Name:      "local_queue_evicted_workloads_total",
			Help: `每个 'local_queue' 下被驱逐的 workload 数量。
标签 'reason' 可能的取值：
- "Preempted"：为更高优先级 workload 或名义配额回收释放资源而驱逐。
- "PodsReadyTimeout"：因 PodsReady 超时而驱逐。
- "AdmissionCheck"：因至少一个 admission check 变为 False 而驱逐。
- "ClusterQueueStopped"：因 ClusterQueue 停止而驱逐。
- "Deactivated"：因 spec.active 设为 false 而驱逐。`,
		}, []string{"name", "namespace", "reason"},
	)

	EvictedWorkloadsOnceTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: over_constants.KueueName,
			Name:      "evicted_workloads_once_total",
			Help: `每个 'cluster_queue' 下唯一 workload 驱逐的数量。
标签 'reason' 可能的取值：
- "Preempted"：为更高优先级 workload 或名义配额回收释放资源而驱逐。
- "PodsReadyTimeout"：因 PodsReady 超时而驱逐。
- "AdmissionCheck"：因至少一个 admission check 变为 False 而驱逐。
- "ClusterQueueStopped"：因 ClusterQueue 停止而驱逐。
- "Deactivated"：因 spec.active 设为 false 而驱逐。`,
		}, []string{"cluster_queue", "reason", "detailed_reason"},
	)

	PreemptedWorkloadsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: over_constants.KueueName,
			Name:      "preempted_workloads_total",
			Help: `每个 'preempting_cluster_queue' 下被抢占的 workload 数量。
标签 'reason' 可能的取值：
- "InClusterQueue"：被同一 ClusterQueue 中的 workload 抢占。
- "InCohortReclamation"：被同一 cohort 中因名义配额回收的 workload 抢占。
- "InCohortFairSharing"：被同一 cohort 中因公平共享的 workload 抢占。
- "InCohortReclaimWhileBorrowing"：被同一 cohort 中因借用时名义配额回收的 workload 抢占。`,
		}, []string{"preempting_cluster_queue", "reason"},
	)

	// Metrics tied to the cache.
	// 与缓存相关的指标。

	ReservingActiveWorkloads = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: over_constants.KueueName,
			Name:      "reserving_active_workloads",
			Help:      "每个 'cluster_queue' 下正在预留配额的 workload 数量。",
		}, []string{"cluster_queue"},
	)

	LocalQueueReservingActiveWorkloads = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: over_constants.KueueName,
			Name:      "local_queue_reserving_active_workloads",
			Help:      "每个 'localQueue' 下正在预留配额的 workload 数量。",
		}, []string{"name", "namespace"},
	)

	AdmittedActiveWorkloads = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: over_constants.KueueName,
			Name:      "admitted_active_workloads",
			Help:      "每个 'cluster_queue' 下已接收且活跃的 workload 数量。",
		}, []string{"cluster_queue"},
	)

	LocalQueueAdmittedActiveWorkloads = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: over_constants.KueueName,
			Name:      "local_queue_admitted_active_workloads",
			Help:      "每个 'localQueue' 下已接收且活跃的 workload 数量。",
		}, []string{"name", "namespace"},
	)

	ClusterQueueByStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: over_constants.KueueName,
			Name:      "cluster_queue_status",
			Help: `报告 'cluster_queue' 及其 'status' (可能值 'pending', 'active' 或 'terminated')。
对于 ClusterQueue，指标只报告一个状态值为 1。`,
		}, []string{"cluster_queue", "status"},
	)

	LocalQueueByStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: over_constants.KueueName,
			Name:      "local_queue_status",
			Help: `报告 'localQueue' 及其 'active' 状态 (可能值 'True', 'False', 或 'Unknown')。
对于 LocalQueue，指标只报告一个状态值为 1。`,
		}, []string{"name", "namespace", "active"},
	)

	// Optional cluster queue metrics

	ClusterQueueResourceReservations = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: over_constants.KueueName,
			Name:      "cluster_queue_resource_reservation",
			Help:      `报告 ClusterQueue 所有 flavor 的总资源预留。`,
		}, []string{"cohort", "cluster_queue", "flavor", "resource"},
	)

	ClusterQueueResourceUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: over_constants.KueueName,
			Name:      "cluster_queue_resource_usage",
			Help:      `报告 ClusterQueue 所有 flavor 的总资源使用。`,
		}, []string{"cohort", "cluster_queue", "flavor", "resource"},
	)

	LocalQueueResourceReservations = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: over_constants.KueueName,
			Name:      "local_queue_resource_reservation",
			Help:      `报告 LocalQueue 所有 flavor 的总资源预留。`,
		}, []string{"name", "namespace", "flavor", "resource"},
	)

	LocalQueueResourceUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: over_constants.KueueName,
			Name:      "local_queue_resource_usage",
			Help:      `报告 LocalQueue 所有 flavor 的总资源使用。`,
		}, []string{"name", "namespace", "flavor", "resource"},
	)

	ClusterQueueResourceNominalQuota = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: over_constants.KueueName,
			Name:      "cluster_queue_nominal_quota",
			Help:      `报告 ClusterQueue 所有 flavor 的名义配额。`,
		}, []string{"cohort", "cluster_queue", "flavor", "resource"},
	)

	ClusterQueueResourceBorrowingLimit = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: over_constants.KueueName,
			Name:      "cluster_queue_borrowing_limit",
			Help:      `报告 ClusterQueue 所有 flavor 的借用限制。`,
		}, []string{"cohort", "cluster_queue", "flavor", "resource"},
	)

	ClusterQueueResourceLendingLimit = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: over_constants.KueueName,
			Name:      "cluster_queue_lending_limit",
			Help:      `报告 ClusterQueue 所有 flavor 的借出限制。`,
		}, []string{"cohort", "cluster_queue", "flavor", "resource"},
	)

	ClusterQueueWeightedShare = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: over_constants.KueueName,
			Name:      "cluster_queue_weighted_share",
			Help: `报告一个值，该值表示使用量超过名义配额与可借出资源的比例，
在 ClusterQueue 提供的所有资源中，除以权重。
如果为零，则表示 ClusterQueue 的使用量低于名义配额。
如果 ClusterQueue 权重为零且正在借用，这将返回 9223372036854775807，
这是可能的最大共享值。`,
		}, []string{"cluster_queue"},
	)

	CohortWeightedShare = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: over_constants.KueueName,
			Name:      "cohort_weighted_share",
			Help: `报告一个值，该值表示使用量超过名义配额与可借出资源的比例，
在 cohort 提供的所有资源中，除以权重。
如果为零，则表示 cohort 的使用量低于名义配额。
如果 cohort 权重为零且正在借用，这将返回 9223372036854775807，
这是可能的最大共享值。`,
		}, []string{"cohort"},
	)
)

func generateExponentialBuckets(count int) []float64 {
	return append([]float64{1}, prometheus.ExponentialBuckets(2.5, 2, count-1)...)
}

func AdmissionAttempt(result AdmissionResult, duration time.Duration) {
	AdmissionAttemptsTotal.WithLabelValues(string(result)).Inc()
	admissionAttemptDuration.WithLabelValues(string(result)).Observe(duration.Seconds())
}

func QuotaReservedWorkload(cqName kueue.ClusterQueueReference, waitTime time.Duration) {
	QuotaReservedWorkloadsTotal.WithLabelValues(string(cqName)).Inc()
	quotaReservedWaitTime.WithLabelValues(string(cqName)).Observe(waitTime.Seconds())
}

func LocalQueueQuotaReservedWorkload(lq LocalQueueReference, waitTime time.Duration) {
	LocalQueueQuotaReservedWorkloadsTotal.WithLabelValues(string(lq.Name), lq.Namespace).Inc()
	localQueueQuotaReservedWaitTime.WithLabelValues(string(lq.Name), lq.Namespace).Observe(waitTime.Seconds())
}

func AdmittedWorkload(cqName kueue.ClusterQueueReference, waitTime time.Duration) {
	AdmittedWorkloadsTotal.WithLabelValues(string(cqName)).Inc()
	admissionWaitTime.WithLabelValues(string(cqName)).Observe(waitTime.Seconds())
}

func LocalQueueAdmittedWorkload(lq LocalQueueReference, waitTime time.Duration) {
	LocalQueueAdmittedWorkloadsTotal.WithLabelValues(string(lq.Name), lq.Namespace).Inc()
	localQueueAdmissionWaitTime.WithLabelValues(string(lq.Name), lq.Namespace).Observe(waitTime.Seconds())
}

func AdmissionChecksWaitTime(cqName kueue.ClusterQueueReference, waitTime time.Duration) {
	admissionChecksWaitTime.WithLabelValues(string(cqName)).Observe(waitTime.Seconds())
}

func LocalQueueAdmissionChecksWaitTime(lq LocalQueueReference, waitTime time.Duration) {
	localQueueAdmissionChecksWaitTime.WithLabelValues(string(lq.Name), lq.Namespace).Observe(waitTime.Seconds())
}

func ReadyWaitTime(cqName kueue.ClusterQueueReference, waitTime time.Duration) {
	queuedUntilReadyWaitTime.WithLabelValues(string(cqName)).Observe(waitTime.Seconds())
}

func LocalQueueReadyWaitTime(lq LocalQueueReference, waitTime time.Duration) {
	localQueueQueuedUntilReadyWaitTime.WithLabelValues(string(lq.Name), lq.Namespace).Observe(waitTime.Seconds())
}

func AdmittedUntilReadyWaitTime(cqName kueue.ClusterQueueReference, waitTime time.Duration) {
	admittedUntilReadyWaitTime.WithLabelValues(string(cqName)).Observe(waitTime.Seconds())
}

func LocalQueueAdmittedUntilReadyWaitTime(lq LocalQueueReference, waitTime time.Duration) {
	localQueueAdmittedUntilReadyWaitTime.WithLabelValues(string(lq.Name), lq.Namespace).Observe(waitTime.Seconds())
}

func ReportPendingWorkloads(cqName kueue.ClusterQueueReference, active, inadmissible int) {
	PendingWorkloads.WithLabelValues(string(cqName), PendingStatusActive).Set(float64(active))
	PendingWorkloads.WithLabelValues(string(cqName), PendingStatusInadmissible).Set(float64(inadmissible))
}

func ReportLocalQueuePendingWorkloads(lq LocalQueueReference, active, inadmissible int) {
	LocalQueuePendingWorkloads.WithLabelValues(string(lq.Name), lq.Namespace, PendingStatusActive).Set(float64(active))
	LocalQueuePendingWorkloads.WithLabelValues(string(lq.Name), lq.Namespace, PendingStatusInadmissible).Set(float64(inadmissible))
}

func ReportEvictedWorkloads(cqName kueue.ClusterQueueReference, reason string) {
	EvictedWorkloadsTotal.WithLabelValues(string(cqName), reason).Inc()
}

func ReportLocalQueueEvictedWorkloads(lq LocalQueueReference, reason string) {
	LocalQueueEvictedWorkloadsTotal.WithLabelValues(string(lq.Name), lq.Namespace, reason).Inc()
}

func ReportEvictedWorkloadsOnce(cqName kueue.ClusterQueueReference, reason, underlyingCause string) {
	EvictedWorkloadsOnceTotal.WithLabelValues(string(cqName), reason, underlyingCause).Inc()
}

func ReportPreemption(preemptingCqName kueue.ClusterQueueReference, preemptingReason string, targetCqName kueue.ClusterQueueReference) {
	PreemptedWorkloadsTotal.WithLabelValues(string(preemptingCqName), preemptingReason).Inc()
	ReportEvictedWorkloads(targetCqName, kueue.WorkloadEvictedByPreemption)
}

func LQRefFromWorkload(wl *kueue.Workload) LocalQueueReference {
	return LocalQueueReference{
		Name:      wl.Spec.QueueName,
		Namespace: wl.Namespace,
	}
}

func ClearClusterQueueMetrics(cqName string) {
	AdmissionCyclePreemptionSkips.DeleteLabelValues(cqName)
	PendingWorkloads.DeleteLabelValues(cqName, PendingStatusActive)
	PendingWorkloads.DeleteLabelValues(cqName, PendingStatusInadmissible)
	QuotaReservedWorkloadsTotal.DeleteLabelValues(cqName)
	quotaReservedWaitTime.DeleteLabelValues(cqName)
	AdmittedWorkloadsTotal.DeleteLabelValues(cqName)
	admissionWaitTime.DeleteLabelValues(cqName)
	admissionChecksWaitTime.DeleteLabelValues(cqName)
	queuedUntilReadyWaitTime.DeleteLabelValues(cqName)
	admittedUntilReadyWaitTime.DeleteLabelValues(cqName)
	EvictedWorkloadsTotal.DeletePartialMatch(prometheus.Labels{"cluster_queue": cqName})
	EvictedWorkloadsOnceTotal.DeletePartialMatch(prometheus.Labels{"cluster_queue": cqName})
	PreemptedWorkloadsTotal.DeletePartialMatch(prometheus.Labels{"preempting_cluster_queue": cqName})
}

func ClearLocalQueueMetrics(lq LocalQueueReference) {
	LocalQueuePendingWorkloads.DeleteLabelValues(string(lq.Name), lq.Namespace, PendingStatusActive)
	LocalQueuePendingWorkloads.DeleteLabelValues(string(lq.Name), lq.Namespace, PendingStatusInadmissible)
	LocalQueueQuotaReservedWorkloadsTotal.DeleteLabelValues(string(lq.Name), lq.Namespace)
	localQueueQuotaReservedWaitTime.DeleteLabelValues(string(lq.Name), lq.Namespace)
	LocalQueueAdmittedWorkloadsTotal.DeleteLabelValues(string(lq.Name), lq.Namespace)
	localQueueAdmissionWaitTime.DeleteLabelValues(string(lq.Name), lq.Namespace)
	localQueueAdmissionChecksWaitTime.DeleteLabelValues(string(lq.Name), lq.Namespace)
	localQueueQueuedUntilReadyWaitTime.DeleteLabelValues(string(lq.Name), lq.Namespace)
	localQueueAdmittedUntilReadyWaitTime.DeleteLabelValues(string(lq.Name), lq.Namespace)
	LocalQueueEvictedWorkloadsTotal.DeletePartialMatch(prometheus.Labels{"name": string(lq.Name), "namespace": lq.Namespace})
}

func ReportClusterQueueStatus(cqName kueue.ClusterQueueReference, cqStatus ClusterQueueStatus) {
	for _, status := range CQStatuses {
		var v float64
		if status == cqStatus {
			v = 1
		}
		ClusterQueueByStatus.WithLabelValues(string(cqName), string(status)).Set(v)
	}
}

var (
	ConditionStatusValues = []metav1.ConditionStatus{metav1.ConditionTrue, metav1.ConditionFalse, metav1.ConditionUnknown}
)

func ReportLocalQueueStatus(lq LocalQueueReference, conditionStatus metav1.ConditionStatus) {
	for _, status := range ConditionStatusValues {
		var v float64
		if status == conditionStatus {
			v = 1
		}
		LocalQueueByStatus.WithLabelValues(string(lq.Name), lq.Namespace, string(status)).Set(v)
	}
}

func ClearCacheMetrics(cqName string) {
	ReservingActiveWorkloads.DeleteLabelValues(cqName)
	AdmittedActiveWorkloads.DeleteLabelValues(cqName)
	for _, status := range CQStatuses {
		ClusterQueueByStatus.DeleteLabelValues(cqName, string(status))
	}
}

func ClearLocalQueueCacheMetrics(lq LocalQueueReference) {
	LocalQueueReservingActiveWorkloads.DeleteLabelValues(string(lq.Name), lq.Namespace)
	LocalQueueAdmittedActiveWorkloads.DeleteLabelValues(string(lq.Name), lq.Namespace)
	for _, status := range ConditionStatusValues {
		LocalQueueByStatus.DeleteLabelValues(string(lq.Name), lq.Namespace, string(status))
	}
}

func ReportClusterQueueQuotas(cohort kueue.CohortReference, queue, flavor, resource string, nominal, borrowing, lending float64) {
	ClusterQueueResourceNominalQuota.WithLabelValues(string(cohort), queue, flavor, resource).Set(nominal)
	ClusterQueueResourceBorrowingLimit.WithLabelValues(string(cohort), queue, flavor, resource).Set(borrowing)
	if features.Enabled(features.LendingLimit) {
		ClusterQueueResourceLendingLimit.WithLabelValues(string(cohort), queue, flavor, resource).Set(lending)
	}
}

func ReportClusterQueueResourceReservations(cohort kueue.CohortReference, queue, flavor, resource string, usage float64) {
	ClusterQueueResourceReservations.WithLabelValues(string(cohort), queue, flavor, resource).Set(usage)
}

func ReportLocalQueueResourceReservations(lq LocalQueueReference, flavor, resource string, usage float64) {
	LocalQueueResourceReservations.WithLabelValues(string(lq.Name), lq.Namespace, flavor, resource).Set(usage)
}

func ReportClusterQueueResourceUsage(cohort kueue.CohortReference, queue, flavor, resource string, usage float64) {
	ClusterQueueResourceUsage.WithLabelValues(string(cohort), queue, flavor, resource).Set(usage)
}

func ReportLocalQueueResourceUsage(lq LocalQueueReference, flavor, resource string, usage float64) {
	LocalQueueResourceUsage.WithLabelValues(string(lq.Name), lq.Namespace, flavor, resource).Set(usage)
}

func ReportClusterQueueWeightedShare(cq string, weightedShare int64) {
	ClusterQueueWeightedShare.WithLabelValues(cq).Set(float64(weightedShare))
}

func ReportCohortWeightedShare(cohort string, weightedShare int64) {
	CohortWeightedShare.WithLabelValues(cohort).Set(float64(weightedShare))
}

func ClearClusterQueueResourceMetrics(cqName string) {
	lbls := prometheus.Labels{
		"cluster_queue": cqName,
	}
	ClusterQueueResourceNominalQuota.DeletePartialMatch(lbls)
	ClusterQueueResourceBorrowingLimit.DeletePartialMatch(lbls)
	if features.Enabled(features.LendingLimit) {
		ClusterQueueResourceLendingLimit.DeletePartialMatch(lbls)
	}
	ClusterQueueResourceUsage.DeletePartialMatch(lbls)
	ClusterQueueResourceReservations.DeletePartialMatch(lbls)
}

func ClearLocalQueueResourceMetrics(lq LocalQueueReference) {
	lbls := prometheus.Labels{
		"name":      string(lq.Name),
		"namespace": lq.Namespace,
	}
	LocalQueueResourceReservations.DeletePartialMatch(lbls)
	LocalQueueResourceUsage.DeletePartialMatch(lbls)
}

func ClearClusterQueueResourceQuotas(cqName, flavor, resource string) {
	lbls := prometheus.Labels{
		"cluster_queue": cqName,
		"flavor":        flavor,
	}

	if len(resource) != 0 {
		lbls["resource"] = resource
	}

	ClusterQueueResourceNominalQuota.DeletePartialMatch(lbls)
	ClusterQueueResourceBorrowingLimit.DeletePartialMatch(lbls)
	if features.Enabled(features.LendingLimit) {
		ClusterQueueResourceLendingLimit.DeletePartialMatch(lbls)
	}
}

func ClearClusterQueueResourceUsage(cqName, flavor, resource string) {
	lbls := prometheus.Labels{
		"cluster_queue": cqName,
		"flavor":        flavor,
	}

	if len(resource) != 0 {
		lbls["resource"] = resource
	}

	ClusterQueueResourceUsage.DeletePartialMatch(lbls)
}

func ClearClusterQueueResourceReservations(cqName, flavor, resource string) {
	lbls := prometheus.Labels{
		"cluster_queue": cqName,
		"flavor":        flavor,
	}

	if len(resource) != 0 {
		lbls["resource"] = resource
	}

	ClusterQueueResourceReservations.DeletePartialMatch(lbls)
}

func Register() {
	metrics.Registry.MustRegister(
		AdmissionAttemptsTotal,
		admissionAttemptDuration,
		AdmissionCyclePreemptionSkips,
		PendingWorkloads,
		ReservingActiveWorkloads,
		AdmittedActiveWorkloads,
		QuotaReservedWorkloadsTotal,
		quotaReservedWaitTime,
		AdmittedWorkloadsTotal,
		EvictedWorkloadsTotal,
		EvictedWorkloadsOnceTotal,
		PreemptedWorkloadsTotal,
		admissionWaitTime,
		admissionChecksWaitTime,
		queuedUntilReadyWaitTime,
		admittedUntilReadyWaitTime,
		ClusterQueueResourceUsage,
		ClusterQueueByStatus,
		ClusterQueueResourceReservations,
		ClusterQueueResourceNominalQuota,
		ClusterQueueResourceBorrowingLimit,
		ClusterQueueResourceLendingLimit,
		ClusterQueueWeightedShare,
		CohortWeightedShare,
	)
	if features.Enabled(features.LocalQueueMetrics) {
		RegisterLQMetrics()
	}
}

func RegisterLQMetrics() {
	metrics.Registry.MustRegister(
		LocalQueuePendingWorkloads,
		LocalQueueReservingActiveWorkloads,
		LocalQueueAdmittedActiveWorkloads,
		LocalQueueQuotaReservedWorkloadsTotal,
		localQueueQuotaReservedWaitTime,
		LocalQueueAdmittedWorkloadsTotal,
		localQueueAdmissionWaitTime,
		localQueueAdmissionChecksWaitTime,
		localQueueQueuedUntilReadyWaitTime,
		localQueueAdmittedUntilReadyWaitTime,
		LocalQueueEvictedWorkloadsTotal,
		LocalQueueByStatus,
		LocalQueueResourceReservations,
		LocalQueueResourceUsage,
	)
}
