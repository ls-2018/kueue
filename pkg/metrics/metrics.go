package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/constants"
	"sigs.k8s.io/kueue/pkg/features"
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

	// CQStatusPending 表示 ClusterQueue 已被接受但尚未激活，
	// 可能原因包括：
	// - ClusterQueue 引用的 ResourceFlavor 缺失
	// - ClusterQueue 引用的 AdmissionCheck 缺失或未激活
	// - ClusterQueue 已停止
	// 在此状态下，ClusterQueue 不能接收新的工作负载，其配额也不能被同组中的其他激活 ClusterQueue 借用。
	CQStatusPending ClusterQueueStatus = "pending"
	// CQStatusActive means the ClusterQueue can admit new workloads and its quota
	// can be borrowed by other ClusterQueues in the cohort.
	// CQStatusActive 表示 ClusterQueue 可以接收新的工作负载，其配额也可以被同组中的其他 ClusterQueue 借用。
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
			Subsystem: constants.KueueName,
			Name:      "admission_attempts_total",
			Help: `The total number of attempts to admit workloads.
Each admission attempt might try to admit more than one workload.
The label 'result' can have the following values:
- 'success' means that at least one workload was admitted.,
- 'inadmissible' means that no workload was admitted.`,
			// 尝试接收工作负载的总次数。
			// 每次接收尝试可能会尝试接收多个工作负载。
			// 标签 'result' 可能有以下值：
			// - 'success' 表示至少有一个工作负载被接收。
			// - 'inadmissible' 表示没有工作负载被接收。
		}, []string{"result"},
	)

	admissionAttemptDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: constants.KueueName,
			Name:      "admission_attempt_duration_seconds",
			Help: `The latency of an admission attempt.
The label 'result' can have the following values:
- 'success' means that at least one workload was admitted.,
- 'inadmissible' means that no workload was admitted.`,
			// 一次接收尝试的延迟。
			// 标签 'result' 可能有以下值：
			// - 'success' 表示至少有一个工作负载被接收。
			// - 'inadmissible' 表示没有工作负载被接收。
		}, []string{"result"},
	)

	AdmissionCyclePreemptionSkips = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: constants.KueueName,
			Name:      "admission_cycle_preemption_skips",
			Help: "The number of Workloads in the ClusterQueue that got preemption candidates " +
				"but had to be skipped because other ClusterQueues needed the same resources in the same cycle",
			// 在 ClusterQueue 中获得抢占候选但由于其他 ClusterQueue 在同一周期需要相同资源而被跳过的工作负载数量
		}, []string{"cluster_queue"},
	)

	// Metrics tied to the queue system.

	PendingWorkloads = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: constants.KueueName,
			Name:      "pending_workloads",
			Help: `The number of pending workloads, per 'cluster_queue' and 'status'.
'status' can have the following values:
- "active" means that the workloads are in the admission queue.
- "inadmissible" means there was a failed admission attempt for these workloads and they won't be retried until cluster conditions, which could make this workload admissible, change`,
			// 每个 'cluster_queue' 和 'status' 下的待处理工作负载数量。
			// 'status' 可能有以下值：
			// - "active" 表示这些工作负载在接收队列中。
			// - "inadmissible" 表示这些工作负载的接收尝试失败，只有当集群条件发生变化使其可接收时才会重试。
		}, []string{"cluster_queue", "status"},
	)

	LocalQueuePendingWorkloads = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: constants.KueueName,
			Name:      "local_queue_pending_workloads",
			Help: `The number of pending workloads, per 'local_queue' and 'status'.
'status' can have the following values:
- "active" means that the workloads are in the admission queue.
- "inadmissible" means there was a failed admission attempt for these workloads and they won't be retried until cluster conditions, which could make this workload admissible, change`,
			// 每个 'local_queue' 和 'status' 下的待处理工作负载数量。
			// 'status' 可能有以下值：
			// - "active" 表示这些工作负载在接收队列中。
			// - "inadmissible" 表示这些工作负载的接收尝试失败，只有当集群条件发生变化使其可接收时才会重试。
		}, []string{"name", "namespace", "status"},
	)

	QuotaReservedWorkloadsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: constants.KueueName,
			Name:      "quota_reserved_workloads_total",
			Help:      "The total number of quota reserved workloads per 'cluster_queue'",
			// 每个 'cluster_queue' 下已保留配额的工作负载总数
		}, []string{"cluster_queue"},
	)

	LocalQueueQuotaReservedWorkloadsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: constants.KueueName,
			Name:      "local_queue_quota_reserved_workloads_total",
			Help:      "The total number of quota reserved workloads per 'local_queue'",
			// 每个 'local_queue' 下已保留配额的工作负载总数
		}, []string{"name", "namespace"},
	)

	quotaReservedWaitTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: constants.KueueName,
			Name:      "quota_reserved_wait_time_seconds",
			Help:      "The time between a workload was created or requeued until it got quota reservation, per 'cluster_queue'",
			// 每个 'cluster_queue' 下，工作负载从创建或重新排队到获得配额保留的时间
			Buckets: generateExponentialBuckets(14),
		}, []string{"cluster_queue"},
	)

	localQueueQuotaReservedWaitTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: constants.KueueName,
			Name:      "local_queue_quota_reserved_wait_time_seconds",
			Help:      "The time between a workload was created or requeued until it got quota reservation, per 'local_queue'",
			// 每个 'local_queue' 下，工作负载从创建或重新排队到获得配额保留的时间
			Buckets: generateExponentialBuckets(14),
		}, []string{"name", "namespace"},
	)

	AdmittedWorkloadsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: constants.KueueName,
			Name:      "admitted_workloads_total",
			Help:      "The total number of admitted workloads per 'cluster_queue'",
			// 每个 'cluster_queue' 下已接收的工作负载总数
		}, []string{"cluster_queue"},
	)

	LocalQueueAdmittedWorkloadsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: constants.KueueName,
			Name:      "local_queue_admitted_workloads_total",
			Help:      "The total number of admitted workloads per 'local_queue'",
			// 每个 'local_queue' 下已接收的工作负载总数
		}, []string{"name", "namespace"},
	)

	admissionWaitTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: constants.KueueName,
			Name:      "admission_wait_time_seconds",
			Help:      "The time between a workload was created or requeued until admission, per 'cluster_queue'",
			// 每个 'cluster_queue' 下，工作负载从创建或重新排队到被接收的时间
			Buckets: generateExponentialBuckets(14),
		}, []string{"cluster_queue"},
	)

	queuedUntilReadyWaitTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: constants.KueueName,
			Name:      "ready_wait_time_seconds",
			Help:      "The time between a workload was created or requeued until ready, per 'cluster_queue'",
			// 每个 'cluster_queue' 下，工作负载从创建或重新排队到就绪的时间
			Buckets: generateExponentialBuckets(14),
		}, []string{"cluster_queue"},
	)

	admittedUntilReadyWaitTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: constants.KueueName,
			Name:      "admitted_until_ready_wait_time_seconds",
			Help:      "The time between a workload was admitted until ready, per 'cluster_queue'",
			// 每个 'cluster_queue' 下，工作负载从被接收到就绪的时间
			Buckets: generateExponentialBuckets(14),
		}, []string{"cluster_queue"},
	)

	localQueueAdmissionWaitTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: constants.KueueName,
			Name:      "local_queue_admission_wait_time_seconds",
			Help:      "The time between a workload was created or requeued until admission, per 'local_queue'",
			// 每个 'local_queue' 下，工作负载从创建或重新排队到被接收的时间
			Buckets: generateExponentialBuckets(14),
		}, []string{"name", "namespace"},
	)

	admissionChecksWaitTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: constants.KueueName,
			Name:      "admission_checks_wait_time_seconds",
			Help:      "The time from when a workload got the quota reservation until admission, per 'cluster_queue'",
			// 每个 'cluster_queue' 下，工作负载从获得配额保留到被接收的时间
			Buckets: generateExponentialBuckets(14),
		}, []string{"cluster_queue"},
	)

	localQueueAdmissionChecksWaitTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: constants.KueueName,
			Name:      "local_queue_admission_checks_wait_time_seconds",
			Help:      "The time from when a workload got the quota reservation until admission, per 'local_queue'",
			// 每个 'local_queue' 下，工作负载从获得配额保留到被接收的时间
			Buckets: generateExponentialBuckets(14),
		}, []string{"name", "namespace"},
	)

	localQueueQueuedUntilReadyWaitTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: constants.KueueName,
			Name:      "local_queue_ready_wait_time_seconds",
			Help:      "The time between a workload was created or requeued until ready, per 'local_queue'",
			// 每个 'local_queue' 下，工作负载从创建或重新排队到就绪的时间
			Buckets: generateExponentialBuckets(14),
		}, []string{"name", "namespace"},
	)

	localQueueAdmittedUntilReadyWaitTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: constants.KueueName,
			Name:      "local_queue_admitted_until_ready_wait_time_seconds",
			Help:      "The time between a workload was admitted until ready, per 'local_queue'",
			// 每个 'local_queue' 下，工作负载从被接收到就绪的时间
			Buckets: generateExponentialBuckets(14),
		}, []string{"name", "namespace"},
	)

	EvictedWorkloadsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: constants.KueueName,
			Name:      "evicted_workloads_total",
			Help: `The number of evicted workloads per 'cluster_queue',
The label 'reason' can have the following values:
- "Preempted" means that the workload was evicted in order to free resources for a workload with a higher priority or reclamation of nominal quota.
- "PodsReadyTimeout" means that the eviction took place due to a PodsReady timeout.
- "AdmissionCheck" means that the workload was evicted because at least one admission check transitioned to False.
- "ClusterQueueStopped" means that the workload was evicted because the ClusterQueue is stopped.
- "Deactivated" means that the workload was evicted because spec.active is set to false`,
			// 每个 'cluster_queue' 下被驱逐的工作负载数量，
			// 标签 'reason' 可能有以下值：
			// - "Preempted" 表示该工作负载被驱逐以释放资源给更高优先级的工作负载或回收名义配额。
			// - "PodsReadyTimeout" 表示因 PodsReady 超时而被驱逐。
			// - "AdmissionCheck" 表示因至少一个接收检查变为 False 而被驱逐。
			// - "ClusterQueueStopped" 表示因 ClusterQueue 停止而被驱逐。
			// - "Deactivated" 表示因 spec.active 设为 false 而被驱逐。
		}, []string{"cluster_queue", "reason"},
	)

	LocalQueueEvictedWorkloadsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: constants.KueueName,
			Name:      "local_queue_evicted_workloads_total",
			Help: `The number of evicted workloads per 'local_queue',
The label 'reason' can have the following values:
- "Preempted" means that the workload was evicted in order to free resources for a workload with a higher priority or reclamation of nominal quota.
- "PodsReadyTimeout" means that the eviction took place due to a PodsReady timeout.
- "AdmissionCheck" means that the workload was evicted because at least one admission check transitioned to False.
- "ClusterQueueStopped" means that the workload was evicted because the ClusterQueue is stopped.
- "Deactivated" means that the workload was evicted because spec.active is set to false`,
			// 每个 'local_queue' 下被驱逐的工作负载数量，
			// 标签 'reason' 可能有以下值：
			// - "Preempted" 表示该工作负载被驱逐以释放资源给更高优先级的工作负载或回收名义配额。
			// - "PodsReadyTimeout" 表示因 PodsReady 超时而被驱逐。
			// - "AdmissionCheck" 表示因至少一个接收检查变为 False 而被驱逐。
			// - "ClusterQueueStopped" 表示因 ClusterQueue 停止而被驱逐。
			// - "Deactivated" 表示因 spec.active 设为 false 而被驱逐。
		}, []string{"name", "namespace", "reason"},
	)

	EvictedWorkloadsOnceTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: constants.KueueName,
			Name:      "evicted_workloads_once_total",
			Help: `The number of unique workload evictions per 'cluster_queue',
The label 'reason' can have the following values:
- "Preempted" means that the workload was evicted in order to free resources for a workload with a higher priority or reclamation of nominal quota.
- "PodsReadyTimeout" means that the eviction took place due to a PodsReady timeout.
- "AdmissionCheck" means that the workload was evicted because at least one admission check transitioned to False.
- "ClusterQueueStopped" means that the workload was evicted because the ClusterQueue is stopped.
- "Deactivated" means that the workload was evicted because spec.active is set to false`,
			// 每个 'cluster_queue' 下唯一工作负载驱逐的数量，
			// 标签 'reason' 可能有以下值：
			// - "Preempted" 表示该工作负载被驱逐以释放资源给更高优先级的工作负载或回收名义配额。
			// - "PodsReadyTimeout" 表示因 PodsReady 超时而被驱逐。
			// - "AdmissionCheck" 表示因至少一个接收检查变为 False 而被驱逐。
			// - "ClusterQueueStopped" 表示因 ClusterQueue 停止而被驱逐。
			// - "Deactivated" 表示因 spec.active 设为 false 而被驱逐。
		}, []string{"cluster_queue", "reason", "detailed_reason"},
	)

	PreemptedWorkloadsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: constants.KueueName,
			Name:      "preempted_workloads_total",
			Help: `The number of preempted workloads per 'preempting_cluster_queue',
The label 'reason' can have the following values:
- "InClusterQueue" means that the workload was preempted by a workload in the same ClusterQueue.
- "InCohortReclamation" means that the workload was preempted by a workload in the same cohort due to reclamation of nominal quota.
- "InCohortFairSharing" means that the workload was preempted by a workload in the same cohort Fair Sharing.
- "InCohortReclaimWhileBorrowing" means that the workload was preempted by a workload in the same cohort due to reclamation of nominal quota while borrowing.`,
			// 每个 'preempting_cluster_queue' 下被抢占的工作负载数量，
			// 标签 'reason' 可能有以下值：
			// - "InClusterQueue" 表示该工作负载被同一 ClusterQueue 中的工作负载抢占。
			// - "InCohortReclamation" 表示该工作负载被同组中因回收名义配额的工作负载抢占。
			// - "InCohortFairSharing" 表示该工作负载被同组中公平共享的工作负载抢占。
			// - "InCohortReclaimWhileBorrowing" 表示该工作负载被同组中借用时回收名义配额的工作负载抢占。
		}, []string{"preempting_cluster_queue", "reason"},
	)

	// Metrics tied to the cache.
	// 与缓存相关的指标。

	ReservingActiveWorkloads = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: constants.KueueName,
			Name:      "reserving_active_workloads",
			Help:      "The number of Workloads that are reserving quota, per 'cluster_queue'",
			// 每个 'cluster_queue' 下正在保留配额的工作负载数量
		}, []string{"cluster_queue"},
	)

	LocalQueueReservingActiveWorkloads = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: constants.KueueName,
			Name:      "local_queue_reserving_active_workloads",
			Help:      "The number of Workloads that are reserving quota, per 'localQueue'",
			// 每个 'localQueue' 下正在保留配额的工作负载数量
		}, []string{"name", "namespace"},
	)

	AdmittedActiveWorkloads = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: constants.KueueName,
			Name:      "admitted_active_workloads",
			Help:      "The number of admitted Workloads that are active (unsuspended and not finished), per 'cluster_queue'",
			// 每个 'cluster_queue' 下已接收且处于活动状态（未挂起且未完成）的工作负载数量
		}, []string{"cluster_queue"},
	)

	LocalQueueAdmittedActiveWorkloads = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: constants.KueueName,
			Name:      "local_queue_admitted_active_workloads",
			Help:      "The number of admitted Workloads that are active (unsuspended and not finished), per 'localQueue'",
			// 每个 'localQueue' 下已接收且处于活动状态（未挂起且未完成）的工作负载数量
		}, []string{"name", "namespace"},
	)

	ClusterQueueByStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: constants.KueueName,
			Name:      "cluster_queue_status",
			Help: `Reports 'cluster_queue' with its 'status' (with possible values 'pending', 'active' or 'terminated').
For a ClusterQueue, the metric only reports a value of 1 for one of the statuses.`,
			// 报告 'cluster_queue' 及其 'status'（可能的值有 'pending'、'active' 或 'terminated'）。
			// 对于一个 ClusterQueue，该指标只会对其中一个状态报告值为 1。
		}, []string{"cluster_queue", "status"},
	)

	LocalQueueByStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: constants.KueueName,
			Name:      "local_queue_status",
			Help: `Reports 'localQueue' with its 'active' status (with possible values 'True', 'False', or 'Unknown').
For a LocalQueue, the metric only reports a value of 1 for one of the statuses.`,
			// 报告 'localQueue' 及其 'active' 状态（可能的值有 'True'、'False' 或 'Unknown'）。
			// 对于一个 LocalQueue，该指标只会对其中一个状态报告值为 1。
		}, []string{"name", "namespace", "active"},
	)

	// Optional cluster queue metrics
	// 可选的 cluster queue 指标

	ClusterQueueResourceReservations = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: constants.KueueName,
			Name:      "cluster_queue_resource_reservation",
			Help:      `Reports the cluster_queue's total resource reservation within all the flavors`,
			// 报告 cluster_queue 在所有 flavor 下的总资源保留量
		}, []string{"cohort", "cluster_queue", "flavor", "resource"},
	)

	ClusterQueueResourceUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: constants.KueueName,
			Name:      "cluster_queue_resource_usage",
			Help:      `Reports the cluster_queue's total resource usage within all the flavors`,
			// 报告 cluster_queue 在所有 flavor 下的总资源使用量
		}, []string{"cohort", "cluster_queue", "flavor", "resource"},
	)

	LocalQueueResourceReservations = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: constants.KueueName,
			Name:      "local_queue_resource_reservation",
			Help:      `Reports the localQueue's total resource reservation within all the flavors`,
			// 报告 localQueue 在所有 flavor 下的总资源保留量
		}, []string{"name", "namespace", "flavor", "resource"},
	)

	LocalQueueResourceUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: constants.KueueName,
			Name:      "local_queue_resource_usage",
			Help:      `Reports the localQueue's total resource usage within all the flavors`,
			// 报告 localQueue 在所有 flavor 下的总资源使用量
		}, []string{"name", "namespace", "flavor", "resource"},
	)

	ClusterQueueResourceNominalQuota = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: constants.KueueName,
			Name:      "cluster_queue_nominal_quota",
			Help:      `Reports the cluster_queue's resource nominal quota within all the flavors`,
			// 报告 cluster_queue 在所有 flavor 下的资源名义配额
		}, []string{"cohort", "cluster_queue", "flavor", "resource"},
	)

	ClusterQueueResourceBorrowingLimit = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: constants.KueueName,
			Name:      "cluster_queue_borrowing_limit",
			Help:      `Reports the cluster_queue's resource borrowing limit within all the flavors`,
			// 报告 cluster_queue 在所有 flavor 下的资源借用上限
		}, []string{"cohort", "cluster_queue", "flavor", "resource"},
	)

	ClusterQueueResourceLendingLimit = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: constants.KueueName,
			Name:      "cluster_queue_lending_limit",
			Help:      `Reports the cluster_queue's resource lending limit within all the flavors`,
			// 报告 cluster_queue 在所有 flavor 下的资源出借上限
		}, []string{"cohort", "cluster_queue", "flavor", "resource"},
	)

	ClusterQueueWeightedShare = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: constants.KueueName,
			Name:      "cluster_queue_weighted_share",
			Help: `Reports a value that representing the maximum of the ratios of usage above nominal 
quota to the lendable resources in the cohort, among all the resources provided by 
the ClusterQueue, and divided by the weight.
If zero, it means that the usage of the ClusterQueue is below the nominal quota.
If the ClusterQueue has a weight of zero and is borrowing, this will return 9223372036854775807,
the maximum possible share value.`,
			// 报告一个值，表示 ClusterQueue 所有资源中，超出名义配额的使用量与同组可借用资源的最大比值，并除以权重。
			// 如果为零，表示 ClusterQueue 的使用量低于名义配额。
			// 如果 ClusterQueue 权重为零且正在借用，则返回 9223372036854775807，即最大可能份额值。
		}, []string{"cluster_queue"},
	)

	CohortWeightedShare = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: constants.KueueName,
			Name:      "cohort_weighted_share",
			Help: `Reports a value that representing the maximum of the ratios of usage above nominal 
quota to the lendable resources in the Cohort, among all the resources provided by 
the Cohort, and divided by the weight.
If zero, it means that the usage of the Cohort is below the nominal quota.
If the Cohort has a weight of zero and is borrowing, this will return 9223372036854775807,
the maximum possible share value.`,
			// 报告一个值，表示 Cohort 所有资源中，超出名义配额的使用量与同组可借用资源的最大比值，并除以权重。
			// 如果为零，表示 Cohort 的使用量低于名义配额。
			// 如果 Cohort 权重为零且正在借用，则返回 9223372036854775807，即最大可能份额值。
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
