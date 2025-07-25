package v1beta1

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkloadSpec 定义了 Workload 的期望状态
// +kubebuilder:validation:XValidation:rule="has(self.priorityClassName) ? has(self.priority) : true", message="priority should not be nil when priorityClassName is set"
type WorkloadSpec struct {
	// podSets 是一组同质 Pod 的集合，每个由 Pod 规范和数量描述。
	// 必须至少有一个元素，最多 8 个。
	// podSets 不可更改。
	//
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=8
	// +kubebuilder:validation:MinItems=1
	PodSets []PodSet `json:"podSets"`

	// queueName 是 Workload 关联的 LocalQueue 的名称。
	// 当 .status.admission 不为 null 时，queueName 不可更改。
	QueueName LocalQueueName `json:"queueName,omitempty"`

	// 如果指定，表示 workload 的优先级。
	// "system-node-critical" 和 "system-cluster-critical" 是两个特殊关键字，分别表示最高优先级和次高优先级。
	// 其他名称必须通过创建具有该名称的 PriorityClass 对象来定义。如果未指定，workload 的优先级将为默认值或零（如果没有默认值）。
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern="^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$"
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// Priority 决定了在 ClusterQueue 中访问资源的顺序。
	// 优先级值由 PriorityClassName 填充。
	// 值越高，优先级越高。
	// 如果指定了 priorityClassName，则 priority 不能为空。
	Priority *int32 `json:"priority,omitempty"`

	// priorityClassSource 决定 priorityClass 字段是指 pod PriorityClass 还是 kueue.x-k8s.io/workloadpriorityclass。
	// Workload 的 PriorityClass 可以接受 pod priorityClass 或 workloadPriorityClass 的名称。
	// 当使用 pod PriorityClass 时，priorityClassSource 字段值为 scheduling.k8s.io/priorityclass。
	// +kubebuilder:default=""
	// +kubebuilder:validation:Enum=kueue.x-k8s.io/workloadpriorityclass;scheduling.k8s.io/priorityclass;""
	PriorityClassSource string `json:"priorityClassSource,omitempty"`

	// Active 决定 workload 是否可以被接纳到队列中。
	// 将 active 从 true 改为 false 会驱逐所有正在运行的 workload。
	// 可能的值：
	//
	//   - false: 表示 workload 永远不会被接纳，并驱逐正在运行的 workload
	//   - true: 表示 workload 可以被评估是否接纳到其所属队列。
	//
	// 默认为 true
	// +kubebuilder:default=true
	Active *bool `json:"active,omitempty"`

	// maximumExecutionTimeSeconds 如果提供，表示 workload 被接纳后允许的最大执行时间（秒），超时后会自动被停用。
	//
	// 如果未指定，则不对 Workload 强制执行时间限制。
	//
	// +optional
	// +kubebuilder:validation:Minimum=1
	MaximumExecutionTimeSeconds *int32 `json:"maximumExecutionTimeSeconds,omitempty"`
}

// PodSetTopologyRequest 定义了 PodSet 的拓扑请求。
type PodSetTopologyRequest struct {
	// required 表示 PodSet 所需的拓扑级别，由 `kueue.x-k8s.io/podset-required-topology` PodSet 注解指示。
	//
	// +optional
	Required *string `json:"required,omitempty"`

	// preferred 表示 PodSet 偏好的拓扑级别，由 `kueue.x-k8s.io/podset-preferred-topology` PodSet 注解指示。
	//
	// +optional
	Preferred *string `json:"preferred,omitempty"`

	// unconstrained 表示 Kueue 在完全可用容量内调度 PodSet 时没有约束，没有放置紧凑性的约束。
	// 这由 `kueue.x-k8s.io/podset-unconstrained-topology` PodSet 注解指示。
	//
	// +optional
	// +kubebuilder:validation:Type=boolean
	Unconstrained *bool `json:"unconstrained,omitempty"`

	// PodIndexLabel 表示 Pod 的索引标签名称。
	// 例如，在 kubernetes job 中是：kubernetes.io/job-completion-index
	PodIndexLabel *string `json:"podIndexLabel,omitempty"`

	// SubGroupIndexLabel 表示 PodSet 中复制的 Jobs（组）实例的索引标签名称。例如，在 JobSet 中是 jobset.sigs.k8s.io/job-index。
	SubGroupIndexLabel *string `json:"subGroupIndexLabel,omitempty"`

	// SubGroupIndexLabel 表示 PodSet 中复制的 Jobs（组）数量。例如，在 JobSet 中，此值从 jobset.sigs.k8s.io/replicatedjob-replicas 读取。
	SubGroupCount *int32 `json:"subGroupCount,omitempty"`
}

type Admission struct {
	// clusterQueue 是集群队列的名称，该集群队列接纳了此 workload。
	ClusterQueue ClusterQueueReference `json:"clusterQueue"`

	// PodSetAssignments 持有每个 .spec.podSets 条目对应的接纳结果。
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=8
	PodSetAssignments []PodSetAssignment `json:"podSetAssignments"`
}

// PodSetReference 是 PodSet 的名称。
// +kubebuilder:validation:MaxLength=63
// +kubebuilder:validation:Pattern="^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"
type PodSetReference string

func NewPodSetReference(name string) PodSetReference {
	return PodSetReference(strings.ToLower(name))
}

type PodSetAssignment struct {
	// Name 是 podSet 的名称。它应该与 .spec.podSets 中的名称之一匹配。
	// +kubebuilder:default=main
	Name PodSetReference `json:"name"`

	// Flavors 是工作负载为每个资源分配的口味。
	Flavors map[corev1.ResourceName]ResourceFlavorReference `json:"flavors,omitempty"`

	// resourceUsage 跟踪所有在 podSet 中运行的 Pod 所需的资源总量。
	//
	// 除了 podSet 规范中提供的内容外，此计算还考虑了接纳时的 LimitRange 默认值和 RuntimeClass 开销。
	// 此字段在配额回收时不会改变。
	ResourceUsage corev1.ResourceList `json:"resourceUsage,omitempty"`

	// count 是接纳时考虑的 Pod 数量。
	// 此字段在配额回收时不会改变。
	// 如果 Workload 创建于添加此字段之前，则此值可能缺失，在这种情况下，spec.podSets[*].count 值将用于。
	//
	// +optional
	// +kubebuilder:validation:Minimum=0
	Count *int32 `json:"count,omitempty"`

	// topologyAssignment 表示拓扑分配，分为拓扑域，对应拓扑的最低级别。
	// 分配指定每个拓扑域中要调度的 Pod 数量，并指定每个拓扑域的节点选择器，如下所示：
	// 节点选择器键由 levels 字段指定（所有域都相同），
	// 对应的节点选择器值由 domains.values 子字段指定。
	// 如果 TopologySpec.Levels 字段包含 "kubernetes.io/hostname" 标签，
	// topologyAssignment 将仅包含此标签的数据，并省略拓扑中的更高级别
	//
	// 示例：
	//
	// topologyAssignment:
	//   levels:
	//   - cloud.provider.com/topology-block
	//   - cloud.provider.com/topology-rack
	//   domains:
	//   - values: [block-1, rack-1]
	//     count: 4
	//   - values: [block-1, rack-2]
	//     count: 2
	//
	// 这里：
	// - 4 个 Pod 要调度到匹配节点选择器的节点：
	//   cloud.provider.com/topology-block: block-1
	//   cloud.provider.com/topology-rack: rack-1
	// - 2 个 Pod 要调度到匹配节点选择器的节点：
	//   cloud.provider.com/topology-block: block-1
	//   cloud.provider.com/topology-rack: rack-2
	//
	// 示例：
	// 下面是一个等效于上述示例的示例，假设 Topology 对象将 kubernetes.io/hostname 定义为拓扑的最低级别。
	// 因此，我们省略了拓扑中的更高级别，因为主机名标签足以明确识别一个合适的节点。
	//
	// topologyAssignment:
	//   levels:
	//   - kubernetes.io/hostname
	//   domains:
	//   - values: [hostname-1]
	//     count: 4
	//   - values: [hostname-2]
	//     count: 2
	//
	// +optional
	TopologyAssignment *TopologyAssignment `json:"topologyAssignment,omitempty"`

	// delayedTopologyRequest 表示拓扑请求被延迟。
	// 当使用 ProvisioningRequest AdmissionCheck 时，拓扑分配可能会延迟。
	// Kueue 对每个 workload 调度第二个调度周期，其中至少有一个 PodSet 具有 delayedTopologyRequest=true 且没有 topologyAssignment。
	//
	// +optional
	DelayedTopologyRequest *DelayedTopologyRequestState `json:"delayedTopologyRequest,omitempty"`
}

// DelayedTopologyRequestState 表示延迟拓扑请求的状态。
// +enum
type DelayedTopologyRequestState string

const (
	// 此状态表示延迟拓扑请求正在等待确定。
	DelayedTopologyRequestStatePending DelayedTopologyRequestState = "Pending"

	// 此状态表示延迟拓扑请求已请求并完成。
	DelayedTopologyRequestStateReady DelayedTopologyRequestState = "Ready"
)

type TopologyAssignment struct {
	// levels 是拓扑分配中按顺序列出的键，表示拓扑级别（即节点标签键），从最高到最低。
	//
	// +required
	// +listType=atomic
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=8
	Levels []string `json:"levels"`

	// domains 是按拓扑域拆分的拓扑分配列表，对应拓扑的最低级别。
	//
	// +required
	Domains []TopologyDomainAssignment `json:"domains"`
}

type TopologyDomainAssignment struct {
	// values 是描述拓扑域的节点选择器值的有序列表。
	// 值对应连续的拓扑级别，从最高到最低。
	//
	// +required
	// +listType=atomic
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=8
	Values []string `json:"values"`

	// count 表示在值字段指示的拓扑域中要调度的 Pod 数量。
	//
	// +required
	// +kubebuilder:validation:Minimum=1
	Count int32 `json:"count"`
}

// +kubebuilder:validation:XValidation:rule="has(self.minCount) ? self.minCount <= self.count : true", message="minCount should be positive and less or equal to count"
type PodSet struct {
	// name 是 PodSet 的名称。
	// +kubebuilder:default=main
	Name PodSetReference `json:"name,omitempty"`

	// template 是 Pod 模板。
	//
	// template.metadata 中仅允许标签和注解。
	//
	// 如果容器或 initContainer 的请求被省略，
	// 它们默认为容器或 initContainer 的限制。
	//
	// 在接纳时，nodeSelector 和
	// nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution 中与 ResourceFlavors 考虑的 nodeLabels 匹配的规则用于过滤可以分配给此 podSet 的 ResourceFlavors。
	Template corev1.PodTemplateSpec `json:"template"`

	// count 是规范中的 Pod 数量。
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	Count int32 `json:"count"`

	// minCount 是规范中接受的 Pod 最小数量
	// 如果工作负载支持部分接纳，则此值未提供。
	// 只有工作负载中的一个 podSet 可以使用此值。
	// 这是一个 alpha 字段，需要启用 PartialAdmission 功能门。
	//
	// +optional
	// +kubebuilder:validation:Minimum=1
	MinCount *int32 `json:"minCount,omitempty"`

	// topologyRequest 定义了 PodSet 的拓扑请求。
	//
	// +optional
	TopologyRequest *PodSetTopologyRequest `json:"topologyRequest,omitempty"`
}

// WorkloadStatus 定义了 Workload 的观察状态
type WorkloadStatus struct {
	// admission 持有工作负载被集群队列接纳的参数。admission 可以设置为 null，但一旦设置，其字段不可更改。
	Admission *Admission `json:"admission,omitempty"`

	// requeueState 持有工作负载在 PodsReadyTimeout 原因下被驱逐时的重队列状态。
	//
	// +optional
	RequeueState *RequeueState `json:"requeueState,omitempty"`

	// conditions 持有工作负载的最新可用观察结果。
	//
	// 条件的类型可以是：
	//
	// - Admitted: 工作负载通过集群队列被接纳。
	// - Finished: 关联的工作负载运行完成（失败或成功）。
	// - PodsReady: 至少 `.spec.podSets[*].count` Pods 已就绪或成功。
	//
	// +optional
	// +listType=map
	// +listMapKey=type
	// +patchStrategy=merge
	// +patchMergeKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// reclaimablePods 跟踪需要资源预留的 Pod 数量。
	// +optional
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=8
	ReclaimablePods []ReclaimablePod `json:"reclaimablePods,omitempty"`

	// admissionChecks 列出工作负载所需的所有接纳检查及其当前状态
	// +optional
	// +listType=map
	// +listMapKey=name
	// +patchStrategy=merge
	// +patchMergeKey=name
	// +kubebuilder:validation:MaxItems=8
	AdmissionChecks []AdmissionCheckState `json:"admissionChecks,omitempty" patchStrategy:"merge" patchMergeKey:"name"`

	// resourceRequests 提供了工作负载在考虑接纳时请求的资源详细视图。
	// 如果 admission 不为 null，则 resourceRequests 将为空，因为 admission.resourceUsage 包含详细信息。
	//
	// +optional
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=8
	ResourceRequests []PodSetRequest `json:"resourceRequests,omitempty"`

	// accumulatedPastExexcutionTimeSeconds 持有工作负载在 Admitted 状态下花费的总时间（秒），
	// 在之前的 `Admit` - `Evict` 周期中。
	//
	// +optional
	AccumulatedPastExexcutionTimeSeconds *int32 `json:"accumulatedPastExexcutionTimeSeconds,omitempty"`

	// schedulingStats 跟踪调度统计信息
	//
	// +optional
	SchedulingStats *SchedulingStats `json:"schedulingStats,omitempty"`
}

type SchedulingStats struct {
	// evictions 按原因和根本原因对驱逐事件的数据进行统计。
	//
	// +optional
	// +listType=map
	// +listMapKey=reason
	// +listMapKey=underlyingCause
	// +patchStrategy=merge
	// +patchMergeKey=reason
	// +patchMergeKey=underlyingCause
	Evictions []WorkloadSchedulingStatsEviction `json:"evictions,omitempty"`
}

type WorkloadSchedulingStatsEviction struct {
	// reason 表示驱逐原因的程序化标识符。
	//
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=316
	Reason string `json:"reason"`

	// underlyingCause 提供了更详细的解释，补充了驱逐原因。
	// 这可能是空字符串。
	//
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=316
	UnderlyingCause string `json:"underlyingCause"`

	// count 跟踪此原因和详细原因的驱逐次数。
	//
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	Count int32 `json:"count"`
}

type RequeueState struct {
	// count 记录工作负载被重队列的次数
	// 当 deactivated (`.spec.activate`=`false`) 的工作负载被 reactivated (`.spec.activate`=`true`) 时，
	// 此计数将重置为 null。
	//
	// +optional
	// +kubebuilder:validation:Minimum=0
	Count *int32 `json:"count,omitempty"`

	// requeueAt 记录工作负载将被重队列的时间。
	// 当 deactivated (`.spec.activate`=`false`) 的工作负载被 reactivated (`.spec.activate`=`true`) 时，
	// 此时间将重置为 null。
	//
	// +optional
	RequeueAt *metav1.Time `json:"requeueAt,omitempty"`
}

// AdmissionCheckReference 是接纳检查的名称。
// +kubebuilder:validation:MaxLength=316
type AdmissionCheckReference string

type AdmissionCheckState struct {
	// name 标识接纳检查。
	// +required
	// +kubebuilder:validation:Required
	Name AdmissionCheckReference `json:"name"`
	// 接纳检查的状态，Pending, Ready, Retry, Rejected
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=Pending;Ready;Retry;Rejected
	State CheckState `json:"state"`
	// lastTransitionTime 是条件从一种状态转换到另一种状态的最后时间。
	// 这应该是基础条件发生变化的时间。 如果不知道，则使用 API 字段更改的时间是可接受的。
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Format=date-time
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	// message 是描述转换的易读消息。
	// 这可能是空字符串。
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=32768
	Message string `json:"message" protobuf:"bytes,6,opt,name=message"`

	// +optional
	// +listType=atomic
	// +kubebuilder:validation:MaxItems=8
	PodSetUpdates []PodSetUpdate `json:"podSetUpdates,omitempty"`
}

// PodSetUpdate 包含 AdmissionChecks 建议的 PodSet 修改列表。
// 修改应仅添加 - 修改已存在的键或由多个 AdmissionChecks 提供相同键的修改是不允许的，
// 并且在工作负载接纳期间会导致失败。
type PodSetUpdate struct {
	// 要修改的 PodSet 名称。应与 Workload 的 PodSets 之一匹配。
	// +required
	// +kubebuilder:validation:Required
	Name PodSetReference `json:"name"`

	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +optional
	// +kubebuilder:validation:MaxItems=8
	// +kubebuilder:validation:XValidation:rule="self.all(x, !has(x.key) ? x.operator == 'Exists' : true)", message="operator must be Exists when 'key' is empty, which means 'match all values and all keys'"
	// +kubebuilder:validation:XValidation:rule="self.all(x, has(x.tolerationSeconds) ? x.effect == 'NoExecute' : true)", message="effect must be 'NoExecute' when 'tolerationSeconds' is set"
	// +kubebuilder:validation:XValidation:rule="self.all(x, !has(x.operator) || x.operator in ['Equal', 'Exists'])", message="supported toleration values: 'Equal'(default), 'Exists'"
	// +kubebuilder:validation:XValidation:rule="self.all(x, has(x.operator) && x.operator == 'Exists' ? !has(x.value) : true)", message="a value must be empty when 'operator' is 'Exists'"
	// +kubebuilder:validation:XValidation:rule="self.all(x, !has(x.effect) || x.effect in ['NoSchedule', 'PreferNoSchedule', 'NoExecute'])", message="supported taint effect values: 'NoSchedule', 'PreferNoSchedule', 'NoExecute'"
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

type ReclaimablePod struct {
	// name 是 PodSet 的名称。
	Name PodSetReference `json:"name"`

	// count 是请求资源不再需要的 Pod 数量。
	// +kubebuilder:validation:Minimum=0
	Count int32 `json:"count"`
}

type PodSetRequest struct {
	// name 是 podSet 的名称。它应该与 .spec.podSets 中的名称之一匹配。
	// +kubebuilder:default=main
	// +required
	// +kubebuilder:validation:Required
	Name PodSetReference `json:"name"`

	// resources 是 podSet 中所有 Pod 运行所需的资源总量。
	//
	// 除了 podSet 规范中提供的内容外，此值还考虑了考虑接纳时的 LimitRange 默认值和 RuntimeClass 开销，
	// 以及 resource.excludeResourcePrefixes 和 resource.transformations 的应用。
	// +optional
	Resources corev1.ResourceList `json:"resources,omitempty"`
}

const (
	WorkloadAdmitted      = "Admitted"      // 表示工作负载已保留配额，且 ClusterQueue 中定义的所有接纳检查均已满足。
	WorkloadQuotaReserved = "QuotaReserved" // 表示工作负载已保留配额
	WorkloadFinished      = "Finished"      // 表示与 ResourceClaim 关联的工作负载运行完成（失败或成功）。
	WorkloadPodsReady     = "PodsReady"     // 表示至少 `.spec.podSets[*].count` Pods 已就绪或成功。

	// WorkloadEvicted 表示工作负载被驱逐。可能的原因是：
	// - "Preempted": 工作负载被抢占
	// - "PodsReadyTimeout": 工作负载超过 PodsReady 超时
	// - "AdmissionCheck": 至少一个接纳检查状态变为 False
	// - "ClusterQueueStopped": 集群队列已停止
	// - "Deactivated": 工作负载 spec.active 设置为 false
	// 当工作负载被抢占时，此条件伴随 "Preempted" 条件，其中包含更详细的抢占原因。
	WorkloadEvicted = "Evicted"

	// WorkloadPreempted 表示工作负载被抢占。
	// 原因字段的可能值是 "InClusterQueue"、"InCohort"。
	// 将来可以引入更多原因，包括传达更详细信息的原因。更详细的原因应以前缀 "base" 之一开头。
	WorkloadPreempted = "Preempted"

	// WorkloadRequeued 表示工作负载因驱逐而被重队列。
	WorkloadRequeued = "Requeued"

	// WorkloadDeactivationTarget 表示工作负载应被停用。
	// 此条件是临时的，因此应在停用后移除。
	WorkloadDeactivationTarget = "DeactivationTarget"
)

// WorkloadPreempted 条件的原因。
const (
	// InClusterQueueReason 表示工作负载因集群队列中的优先级而被抢占。
	InClusterQueueReason string = "InClusterQueue"

	// InCohortReclamationReason 表示工作负载因 cohort 中的回收而被抢占。
	InCohortReclamationReason string = "InCohortReclamation"

	// InCohortFairSharingReason 表示工作负载因 cohort 中的公平共享而被抢占。
	InCohortFairSharingReason string = "InCohortFairSharing"

	// InCohortReclaimWhileBorrowingReason 表示工作负载因 cohort 中的回收而被抢占，同时正在借用。
	InCohortReclaimWhileBorrowingReason string = "InCohortReclaimWhileBorrowing"
)

const (
	// WorkloadInadmissible 表示工作负载无法保留配额
	// 由于 LocalQueue 或 ClusterQueue 不存在或处于非活动状态。
	WorkloadInadmissible = "Inadmissible"

	// WorkloadEvictedByPreemption 表示工作负载因抢占而驱逐，
	// 以便为具有更高优先级的工作负载释放资源。
	WorkloadEvictedByPreemption = "Preempted"

	// WorkloadEvictedByPodsReadyTimeout 表示驱逐是由于 PodsReady 超时而发生的。
	WorkloadEvictedByPodsReadyTimeout = "PodsReadyTimeout"

	// WorkloadEvictedByAdmissionCheck 表示工作负载因至少一个接纳检查状态变为 False 而驱逐。
	WorkloadEvictedByAdmissionCheck = "AdmissionCheck"

	// WorkloadEvictedByClusterQueueStopped 表示工作负载因集群队列已停止而驱逐。
	WorkloadEvictedByClusterQueueStopped = "ClusterQueueStopped"

	// WorkloadEvictedByLocalQueueStopped 表示工作负载因 LocalQueue 已停止而驱逐。
	WorkloadEvictedByLocalQueueStopped = "LocalQueueStopped"

	// WorkloadEvictedDueToNodeFailures 表示工作负载因不可恢复的节点故障而驱逐。
	WorkloadEvictedDueToNodeFailures = "NodeFailures"

	// WorkloadDeactivated 表示工作负载因 spec.active 设置为 false 而驱逐。
	WorkloadDeactivated = "Deactivated"

	// WorkloadReactivated 表示工作负载因 spec.active 设置为 true 而重队列，
	// 在停用后。
	WorkloadReactivated = "Reactivated"

	// WorkloadBackoffFinished 表示工作负载因回退完成而重队列。
	WorkloadBackoffFinished = "BackoffFinished"

	// WorkloadClusterQueueRestarted 表示工作负载因集群队列重启而重队列，
	// 在停止后。
	WorkloadClusterQueueRestarted = "ClusterQueueRestarted"

	// WorkloadLocalQueueRestarted 表示工作负载因 LocalQueue 重启而重队列，
	// 在停止后。
	WorkloadLocalQueueRestarted = "LocalQueueRestarted"

	// WorkloadRequeuingLimitExceeded 表示工作负载超过最大重队列重试次数。
	WorkloadRequeuingLimitExceeded = "RequeuingLimitExceeded"

	// WorkloadMaximumExecutionTimeExceeded 表示工作负载超过其最大执行时间。
	WorkloadMaximumExecutionTimeExceeded = "MaximumExecutionTimeExceeded"

	// WorkloadWaitForStart 表示 PodsReady=False 条件的原因，
	// 当 pods 自接纳以来未就绪，或工作负载未被接纳。
	WorkloadWaitForStart = "WaitForStart"

	// WorkloadWaitForRecovery 表示 PodsReady=False 条件的原因，
	// 当 pods 自工作负载接纳以来就绪，但某些 pod 失败，
	// 工作负载正在等待恢复。
	WorkloadWaitForRecovery = "WaitForRecovery"

	// WorkloadStarted 表示所有 Pods 已就绪，且工作负载已成功启动。
	WorkloadStarted = "Started"

	// WorkloadRecovered 表示至少一个 Pod 失败后，工作负载已恢复并正在运行。
	WorkloadRecovered = "Recovered"
)

const (
	// WorkloadFinishedReasonSucceeded 表示工作负载的作业成功完成。
	WorkloadFinishedReasonSucceeded = "Succeeded"

	// WorkloadFinishedReasonFailed 表示工作负载的作业因错误而完成。
	WorkloadFinishedReasonFailed = "Failed"

	// WorkloadFinishedReasonOutOfSync 表示预构建的工作负载与其父作业不同步。
	WorkloadFinishedReasonOutOfSync = "OutOfSync"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Queue",JSONPath=".spec.queueName",type="string",description="Name of the queue this workload was submitted to"
// +kubebuilder:printcolumn:name="Reserved in",JSONPath=".status.admission.clusterQueue",type="string",description="Name of the ClusterQueue where the workload is reserving quota"
// +kubebuilder:printcolumn:name="Admitted",JSONPath=".status.conditions[?(@.type=='Admitted')].status",type="string",description="Admission status"
// +kubebuilder:printcolumn:name="Finished",JSONPath=".status.conditions[?(@.type=='Finished')].status",type="string",description="Workload finished"
// +kubebuilder:printcolumn:name="Age",JSONPath=".metadata.creationTimestamp",type="date",description="Time this workload was created"
// +kubebuilder:resource:shortName={wl}

// Workload 是工作负载 API 的 Schema
// +kubebuilder:validation:XValidation:rule="has(self.status) && has(self.status.conditions) && self.status.conditions.exists(c, c.type == 'QuotaReserved' && c.status == 'True') && has(self.status.admission) ? size(self.spec.podSets) == size(self.status.admission.podSetAssignments) : true", message="podSetAssignments 的数量必须与 spec 中的 podSets 数量相同"
// +kubebuilder:validation:XValidation:rule="(has(oldSelf.status) && has(oldSelf.status.conditions) && oldSelf.status.conditions.exists(c, c.type == 'QuotaReserved' && c.status == 'True')) ? (oldSelf.spec.priorityClassSource == self.spec.priorityClassSource) : true", message="字段不可变"
// +kubebuilder:validation:XValidation:rule="(has(oldSelf.status) && has(oldSelf.status.conditions) && oldSelf.status.conditions.exists(c, c.type == 'QuotaReserved' && c.status == 'True') && has(oldSelf.spec.priorityClassName) && has(self.spec.priorityClassName)) ? (oldSelf.spec.priorityClassName == self.spec.priorityClassName) : true", message="字段不可变"
// +kubebuilder:validation:XValidation:rule="(has(oldSelf.status) && has(oldSelf.status.conditions) && oldSelf.status.conditions.exists(c, c.type == 'QuotaReserved' && c.status == 'True')) && (has(self.status) && has(self.status.conditions) && self.status.conditions.exists(c, c.type == 'QuotaReserved' && c.status == 'True')) && has(oldSelf.spec.queueName) && has(self.spec.queueName) ? oldSelf.spec.queueName == self.spec.queueName : true", message="字段不可变"
// +kubebuilder:validation:XValidation:rule="((has(oldSelf.status) && has(oldSelf.status.conditions) && oldSelf.status.conditions.exists(c, c.type == 'Admitted' && c.status == 'True')) && (has(self.status) && has(self.status.conditions) && self.status.conditions.exists(c, c.type == 'Admitted' && c.status == 'True')))?((has(oldSelf.spec.maximumExecutionTimeSeconds)?oldSelf.spec.maximumExecutionTimeSeconds:0) ==  (has(self.spec.maximumExecutionTimeSeconds)?self.spec.maximumExecutionTimeSeconds:0)):true", message="maximumExecutionTimeSeconds 在已接纳时不可变"
type Workload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkloadSpec   `json:"spec,omitempty"`
	Status WorkloadStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WorkloadList 包含 ResourceClaim 列表
type WorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workload `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Workload{}, &WorkloadList{})
}
