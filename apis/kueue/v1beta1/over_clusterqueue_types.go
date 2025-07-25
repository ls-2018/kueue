package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterQueue 活跃状态的原因。
const (
	ClusterQueueActiveReasonTerminating                              = "Terminating"
	ClusterQueueActiveReasonStopped                                  = "Stopped"
	ClusterQueueActiveReasonFlavorNotFound                           = "FlavorNotFound"
	ClusterQueueActiveReasonAdmissionCheckNotFound                   = "AdmissionCheckNotFound"
	ClusterQueueActiveReasonAdmissionCheckInactive                   = "AdmissionCheckInactive"
	ClusterQueueActiveReasonMultipleMultiKueueAdmissionChecks        = "MultipleMultiKueueAdmissionChecks"
	ClusterQueueActiveReasonMultiKueueAdmissionCheckAppliedPerFlavor = "MultiKueueAdmissionCheckAppliedPerFlavor"
	ClusterQueueActiveReasonNotSupportedWithTopologyAwareScheduling  = "NotSupportedWithTopologyAwareScheduling"
	ClusterQueueActiveReasonTopologyNotFound                         = "TopologyNotFound"
	ClusterQueueActiveReasonUnknown                                  = "Unknown"
	ClusterQueueActiveReasonReady                                    = "Ready"
)

// ClusterQueueReference 是 ClusterQueue 的名称。
// 它必须是 DNS（RFC 1123）格式，最大长度为 253 个字符。
//
// +kubebuilder:validation:MaxLength=253
// +kubebuilder:validation:Pattern="^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$"
type ClusterQueueReference string

// CohortReference 是 Cohort 的名称。
//
// Cohort 名称的校验等同于对象名称的校验：DNS（RFC 1123）中的子域名。
// +kubebuilder:validation:MaxLength=253
// +kubebuilder:validation:Pattern="^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$"
type CohortReference string

// ClusterQueueSpec 定义了 ClusterQueue 的期望状态
// +kubebuilder:validation:XValidation:rule="!has(self.cohort) && has(self.resourceGroups) ? self.resourceGroups.all(rg, rg.flavors.all(f, f.resources.all(r, !has(r.borrowingLimit)))) : true", message="borrowingLimit must be nil when cohort is empty"
type ClusterQueueSpec struct {
	// resourceGroups 描述资源组。
	// 每个资源组定义了资源列表和为这些资源提供配额的 flavor 列表。
	// 每个资源和每个 flavor 只能属于一个资源组。
	// resourceGroups 最多可有 16 个。
	// +listType=atomic
	// +kubebuilder:validation:MaxItems=16
	ResourceGroups []ResourceGroup `json:"resourceGroups,omitempty"`

	// cohort 表示该 ClusterQueue 所属的组。属于同一 cohort 的 CQ 可以相互借用未使用的资源。
	//
	// 一个 CQ 只能属于一个借用 cohort。提交到引用此 CQ 的队列的工作负载可以从 cohort 中的任何 CQ 借用配额。
	// 只有 CQ 中列出的 [resource, flavor] 对的配额可以被借用。
	// 如果为空，则该 ClusterQueue 不能从其他 ClusterQueue 借用，也不能被借用。
	//
	// cohort 是将 CQ 关联在一起的名称，但它不引用任何对象。
	Cohort CohortReference `json:"cohort,omitempty"`

	// QueueingStrategy 表示该 ClusterQueue 中工作负载的排队策略。
	// 当前支持的策略：
	//
	// - StrictFIFO：工作负载严格按创建时间排序。
	//   不能被接纳的较旧工作负载会阻塞即使适配现有配额的新工作负载的接纳。
	// - BestEffortFIFO：工作负载按创建时间排序，
	//   但不能被接纳的较旧工作负载不会阻塞适配现有配额的新工作负载的接纳。
	//
	// +kubebuilder:default=BestEffortFIFO
	// +kubebuilder:validation:Enum=StrictFIFO;BestEffortFIFO
	QueueingStrategy QueueingStrategy `json:"queueingStrategy,omitempty"`

	// namespaceSelector 定义哪些命名空间可以向该 clusterQueue 提交工作负载。更高级的策略建议使用 Gatekeeper 等策略代理实现。
	// 默认为 null，即无命名空间可用。
	// 如果设置为空选择器 `{}`，则所有命名空间都可用。
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`

	// flavorFungibility 定义在借用或抢占当前 flavor 前，工作负载是否应尝试下一个 flavor。
	// +kubebuilder:default={}
	FlavorFungibility *FlavorFungibility `json:"flavorFungibility,omitempty"`

	// +kubebuilder:default={}
	Preemption *ClusterQueuePreemption `json:"preemption,omitempty"`

	// admissionChecks 列出该 ClusterQueue 所需的 AdmissionChecks。
	// 不能与 AdmissionCheckStrategy 同时使用。
	// +optional
	AdmissionChecks []AdmissionCheckReference `json:"admissionChecks,omitempty"`

	// admissionCheckStrategy 定义用于确定哪些 ResourceFlavors 需要 AdmissionChecks 的策略列表。
	// 此属性不能与 'admissionChecks' 属性同时使用。
	// +optional
	AdmissionChecksStrategy *AdmissionChecksStrategy `json:"admissionChecksStrategy,omitempty"`

	// stopPolicy - 如果设置为非 None，则该 ClusterQueue 被视为非活跃状态，不会进行新的资源预留。
	//
	// 根据其值，相关工作负载将：
	//
	// - None - 工作负载被接纳
	// - HoldAndDrain - 已接纳的工作负载会被驱逐，预留中的工作负载会取消预留。
	// - Hold - 已接纳的工作负载会运行至完成，预留中的工作负载会取消预留。
	//
	// +optional
	// +kubebuilder:validation:Enum=None;Hold;HoldAndDrain
	// +kubebuilder:default="None"
	StopPolicy *StopPolicy `json:"stopPolicy,omitempty"`

	// fairSharing 定义该 ClusterQueue 参与公平共享时的属性。仅在 Kueue 配置启用公平共享时有效。
	// +optional
	FairSharing *FairSharing `json:"fairSharing,omitempty"`

	// admissionScope 指示 ClusterQueue 是否使用 Admission Fair Sharing。
	// +optional
	AdmissionScope *AdmissionScope `json:"admissionScope,omitempty"`
}

// AdmissionChecksStrategy 定义用于确定哪些 ResourceFlavors 需要 AdmissionChecks 的策略列表。
type AdmissionChecksStrategy struct {
	// admissionChecks 是 AdmissionChecks 的策略列表。
	AdmissionChecks []AdmissionCheckStrategyRule `json:"admissionChecks,omitempty"`
}

// AdmissionCheckStrategyRule 定义单个 AdmissionCheck 的规则。
type AdmissionCheckStrategyRule struct {
	// name 是 AdmissionCheck 的名称。
	Name AdmissionCheckReference `json:"name"`

	// onFlavors 是该 AdmissionCheck 应运行的 ResourceFlavors 名称列表。
	// 如果为空，则该 AdmissionCheck 将运行于提交到 ClusterQueue 的所有工作负载。
	// +optional
	OnFlavors []ResourceFlavorReference `json:"onFlavors,omitempty"`
}

type QueueingStrategy string

const (
	// StrictFIFO 表示工作负载按优先级严格按创建时间排序。
	// 不能被接纳的较旧工作负载会阻塞即使适配现有配额的新工作负载的接纳。
	StrictFIFO QueueingStrategy = "StrictFIFO"

	// BestEffortFIFO 表示工作负载按创建时间排序，
	// 但不能被接纳的较旧工作负载不会阻塞适配现有配额的新工作负载的接纳。
	BestEffortFIFO QueueingStrategy = "BestEffortFIFO"
)

// +kubebuilder:validation:XValidation:rule="self.flavors.all(x, size(x.resources) == size(self.coveredResources))", message="flavors must have the same number of resources as the coveredResources"
type ResourceGroup struct {
	// coveredResources 是该组覆盖的资源列表。
	// 示例：cpu, memory, vendor.com/gpu。
	// 列表不能为空，最多可包含 16 种资源。
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	CoveredResources []corev1.ResourceName `json:"coveredResources"`

	// flavors 是提供该组资源的 flavor 列表。
	// 通常，不同的 flavor 代表不同的硬件模型
	// (例如，gpu 型号，cpu 架构) 或定价模型 (按需 vs 竞价
	// cpus)。
	// 每个 flavor 必须按与 .resources 字段相同的顺序列出所有资源。
	// 列表不能为空，最多可包含 16 个 flavor。
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	Flavors []FlavorQuotas `json:"flavors"`
}

type FlavorQuotas struct {
	// name 是此 flavor 的名称。名称应与 ResourceFlavor 的 .metadata.name 匹配。
	// 如果不存在匹配的 ResourceFlavor，ClusterQueue 的 Active 条件将被设置为 False。
	Name ResourceFlavorReference `json:"name"`

	// resources 是该 flavor 针对每种资源的配额列表。
	// 最多可有 16 种资源。
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	Resources []ResourceQuota `json:"resources"`
}

type ResourceQuota struct {
	// name 是此资源的名称。
	Name corev1.ResourceName `json:"name"`

	// nominalQuota 是该资源在某一时刻可用于工作负载的配额数量。
	// nominalQuota 必须为非负数。
	// nominalQuota 应代表集群中可用于运行作业的资源（扣除系统组件和非 kueue 管理的 pod 所消耗的资源后）。
	// 在自动扩缩容集群中，nominalQuota 应考虑如 Kubernetes cluster-autoscaler 等组件可提供的资源。
	//
	// 如果 ClusterQueue 属于 cohort，则每个（flavor, resource）组合的配额总和定义了 cohort 中 ClusterQueue 可分配的最大数量。
	NominalQuota resource.Quantity `json:"nominalQuota"`

	// borrowingLimit 是该 ClusterQueue 可从同一 cohort 其他 ClusterQueue 未使用配额中借用的最大数量。
	// 在任意时刻，ClusterQueue 中的工作负载最多可消耗 nominalQuota+borrowingLimit 的配额，前提是 cohort 中其他 ClusterQueue 有足够未用配额。
	// 如果为 null，表示没有借用上限。
	// 如果不为 null，必须为非负数。
	// 如果 spec.cohort 为空，borrowingLimit 必须为 null。
	// +optional
	BorrowingLimit *resource.Quantity `json:"borrowingLimit,omitempty"`

	// lendingLimit 是该 ClusterQueue 可借给同一 cohort 其他 ClusterQueue 的未用配额的最大数量。
	// 在任意时刻，ClusterQueue 为自身专用保留的配额为 nominalQuota - lendingLimit。
	// 如果为 null，表示没有出借上限，即所有 nominalQuota 都可被 cohort 中其他 ClusterQueue 借用。
	// 如果不为 null，必须为非负数。
	// 如果 spec.cohort 为空，lendingLimit 必须为 null。
	// 此字段为 beta 阶段，默认启用。
	// +optional
	LendingLimit *resource.Quantity `json:"lendingLimit,omitempty"`
}

// ResourceFlavorReference 是 ResourceFlavor 的名称。
// +kubebuilder:validation:MaxLength=253
// +kubebuilder:validation:Pattern="^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$"
type ResourceFlavorReference string

// ClusterQueueStatus 定义了 ClusterQueue 的当前状态。
type ClusterQueueStatus struct {
	// flavorsReservation 是当前分配给该 ClusterQueue 的工作负载的预留配额，按 flavor 分组。
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=16
	// +optional
	FlavorsReservation []FlavorUsage `json:"flavorsReservation"`

	// flavorsUsage 是当前分配给该 ClusterQueue 的工作负载的已使用配额，按 flavor 分组。
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=16
	// +optional
	FlavorsUsage []FlavorUsage `json:"flavorsUsage"`

	// pendingWorkloads 是当前等待被接纳到该 clusterQueue 的工作负载数量。
	// +optional
	PendingWorkloads int32 `json:"pendingWorkloads"`

	// reservingWorkloads 是当前预留配额的工作负载数量。
	// +optional
	ReservingWorkloads int32 `json:"reservingWorkloads"`

	// admittedWorkloads 是当前已接纳到该 clusterQueue 且尚未完成的工作负载数量。
	// +optional
	AdmittedWorkloads int32 `json:"admittedWorkloads"`

	// conditions 持有该 ClusterQueue 的最新可用观察结果。
	// +optional
	// +listType=map
	// +listMapKey=type
	// +patchStrategy=merge
	// +patchMergeKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// PendingWorkloadsStatus 包含关于集群队列中等待工作负载状态的信息。
	// 弃用：此字段将在 v1beta2 中移除，请改用 VisibilityOnDemand
	// (https://kueue.sigs.k8s.io/docs/tasks/manage/monitor_pending_workloads/pending_workloads_on_demand/)
	// 替代。
	// +optional
	PendingWorkloadsStatus *ClusterQueuePendingWorkloadsStatus `json:"pendingWorkloadsStatus"`

	// fairSharing 仅在 Kueue 配置启用公平共享时记录该 ClusterQueue 的当前状态。
	// +optional
	FairSharing *FairSharingStatus `json:"fairSharing,omitempty"`
}

type ClusterQueuePendingWorkloadsStatus struct {
	// Head 包含等待工作负载的顶部列表。
	// +listType=atomic
	// +optional
	Head []ClusterQueuePendingWorkload `json:"clusterQueuePendingWorkload"`

	// LastChangeTime 表示结构的最后更改时间。
	LastChangeTime metav1.Time `json:"lastChangeTime"`
}

// ClusterQueuePendingWorkload 包含识别等待工作负载的信息。
type ClusterQueuePendingWorkload struct {
	// Name 表示等待工作负载的名称。
	Name string `json:"name"`

	// Namespace 表示等待工作负载的命名空间。
	Namespace string `json:"namespace"`
}

type FlavorUsage struct {
	// name 是 flavor 的名称。
	Name ResourceFlavorReference `json:"name"`

	// resources 是该 flavor 的资源配额使用情况。
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=16
	Resources []ResourceUsage `json:"resources"`
}

type ResourceUsage struct {
	// name 是资源名称。
	Name corev1.ResourceName `json:"name"`

	// total 是已使用的配额总量，包括从 cohort 借用的配额。
	Total resource.Quantity `json:"total,omitempty"`

	// Borrowed 是借用自 cohort 的配额数量。换句话说，它是超过 nominalQuota 的已使用配额。
	Borrowed resource.Quantity `json:"borrowed,omitempty"`
}

const (
	// ClusterQueueActive 表示 ClusterQueue 可以接纳新工作负载，其配额可以被同一 cohort 中的其他 ClusterQueues 借用。
	ClusterQueueActive string = "Active"
)

type PreemptionPolicy string

const (
	PreemptionPolicyNever                     PreemptionPolicy = "Never"
	PreemptionPolicyAny                       PreemptionPolicy = "Any"
	PreemptionPolicyLowerPriority             PreemptionPolicy = "LowerPriority"
	PreemptionPolicyLowerOrNewerEqualPriority PreemptionPolicy = "LowerOrNewerEqualPriority"
)

type FlavorFungibilityPolicy string

const (
	Borrow        FlavorFungibilityPolicy = "Borrow"
	Preempt       FlavorFungibilityPolicy = "Preempt"
	TryNextFlavor FlavorFungibilityPolicy = "TryNextFlavor"
)

// FlavorFungibility 确定工作负载在当前 flavor 借用或抢占前是否应尝试下一个 flavor。
type FlavorFungibility struct {
	// whenCanBorrow 确定工作负载在当前 flavor 借用前是否应尝试下一个 flavor。可能的值是：
	//
	// - `Borrow` (default): 如果可以借用，则在当前 flavor 中分配。
	// - `TryNextFlavor`: 即使当前 flavor 有足够资源可以借用，也尝试下一个 flavor。
	//
	// +kubebuilder:validation:Enum={Borrow,TryNextFlavor}
	// +kubebuilder:default="Borrow"
	WhenCanBorrow FlavorFungibilityPolicy `json:"whenCanBorrow,omitempty"`
	// whenCanPreempt 确定工作负载在当前 flavor 借用前是否应尝试下一个 flavor。可能的值是：
	//
	// - `Preempt`: 如果可以抢占一些工作负载，则在当前 flavor 中分配。
	// - `TryNextFlavor` (default): 即使当前 flavor 有足够候选者进行抢占，也尝试下一个 flavor。
	//
	// +kubebuilder:validation:Enum={Preempt,TryNextFlavor}
	// +kubebuilder:default="TryNextFlavor"
	WhenCanPreempt FlavorFungibilityPolicy `json:"whenCanPreempt,omitempty"`
}

// ClusterQueuePreemption 包含抢占工作负载的策略。
//
// 抢占可能配置在以下场景中工作：
//
//   - 当一个工作负载适合 ClusterQueue 的配额，但该配额当前被 cohort 中的其他 ClusterQueues 借用。
//     我们抢占其他 ClusterQueues 的工作负载，以允许此 ClusterQueue 回收其配额。使用 reclaimWithinCohort 配置。
//   - 当一个工作负载不适合 ClusterQueue 的配额，且 ClusterQueue 中有优先级较低的工作负载。
//     使用 withinClusterQueue 配置。
//   - 当一个工作负载可能同时借用和抢占
//     低优先级工作负载在 cohort 中。使用 borrowWithinCohort 配置。
//   - 当启用公平共享时，以维护公平分配
//     未使用的资源。请参阅公平共享文档。
//
// 抢占算法尝试找到一个最小的工作负载集来
// 抢占以适应待处理的工作负载，优先抢占优先级较低的工作负载。
// +kubebuilder:validation:XValidation:rule="!(self.reclaimWithinCohort == 'Never' && has(self.borrowWithinCohort) &&  self.borrowWithinCohort.policy != 'Never')", message="reclaimWithinCohort=Never and borrowWithinCohort.Policy!=Never"
type ClusterQueuePreemption struct {
	// reclaimWithinCohort 确定一个待处理的工作负载是否可以抢占
	// cohort 中使用超过其配额的 ClusterQueues 的工作负载。可能的值是：
	//
	// - `Never` (default): 不要抢占 cohort 中的工作负载。
	// - `LowerPriority`: **经典抢占** 如果待处理的工作负载
	//   适合其 ClusterQueue 的配额，仅抢占 cohort 中优先级低于待处理工作负载的工作负载。**公平共享** 仅抢占 cohort 中优先级低于待处理工作负载且满足公平共享抢占策略的工作负载。
	// - `Any`: **经典抢占** 如果待处理的工作负载适合其 ClusterQueue 的配额，抢占 cohort 中的任何工作负载，不论优先级。**公平共享** 抢占满足公平共享抢占策略的工作负载。
	//
	// +kubebuilder:default=Never
	// +kubebuilder:validation:Enum=Never;LowerPriority;Any
	ReclaimWithinCohort PreemptionPolicy `json:"reclaimWithinCohort,omitempty"`

	// +kubebuilder:default={}
	BorrowWithinCohort *BorrowWithinCohort `json:"borrowWithinCohort,omitempty"`

	// withinClusterQueue 确定一个待处理的工作负载是否不适合其 ClusterQueue 的配额，可以抢占 ClusterQueue 中的活跃工作负载。可能的值是：
	//
	// - `Never` (default): 不要抢占 ClusterQueue 中的工作负载。
	// - `LowerPriority`: 仅抢占 ClusterQueue 中优先级低于待处理工作负载的工作负载。
	// - `LowerOrNewerEqualPriority`: 仅抢占 ClusterQueue 中优先级低于待处理工作负载或优先级相等且比待处理工作负载新的工作负载。
	//
	// +kubebuilder:default=Never
	// +kubebuilder:validation:Enum=Never;LowerPriority;LowerOrNewerEqualPriority
	WithinClusterQueue PreemptionPolicy `json:"withinClusterQueue,omitempty"`
}

type BorrowWithinCohortPolicy string

const (
	BorrowWithinCohortPolicyNever         BorrowWithinCohortPolicy = "Never"
	BorrowWithinCohortPolicyLowerPriority BorrowWithinCohortPolicy = "LowerPriority"
)

// BorrowWithinCohort 包含在借用时抢占工作负载的配置。它仅与经典抢占一起工作，
// __不__ 与公平共享一起工作。
type BorrowWithinCohort struct {
	// policy 确定在借用时从 cohort 中抢占工作负载的策略。
	// 可能的值是：
	// - `Never` (default): 不允许抢占，在 cohort 中的其他 ClusterQueues 中，对于一个借用工作负载。
	// - `LowerPriority`: 允许抢占，在 cohort 中的其他 ClusterQueues 中，对于一个借用工作负载，但仅当抢占的工作负载优先级较低时。
	//
	// +kubebuilder:default=Never
	// +kubebuilder:validation:Enum=Never;LowerPriority
	Policy BorrowWithinCohortPolicy `json:"policy,omitempty"`

	// maxPriorityThreshold 允许限制被借用工作负载抢占的工作负载集，仅限于优先级小于或等于指定阈值优先级的工作负载。
	// 当阈值未指定时，则任何满足策略的工作负载都可以被借用工作负载抢占。
	//
	// +optional
	MaxPriorityThreshold *int32 `json:"maxPriorityThreshold,omitempty"`
}

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Cluster,shortName={cq}
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Cohort",JSONPath=".spec.cohort",type=string,description="ClusterQueue 所属的 cohort"
// +kubebuilder:printcolumn:name="Strategy",JSONPath=".spec.queueingStrategy",type=string,description="用于优先级工作负载的排队策略",priority=1
// +kubebuilder:printcolumn:name="Pending Workloads",JSONPath=".status.pendingWorkloads",type=integer,description="等待工作负载数量"
// +kubebuilder:printcolumn:name="Admitted Workloads",JSONPath=".status.admittedWorkloads",type=integer,description="尚未完成的已接纳工作负载数量",priority=1

// ClusterQueue 是 ClusterQueue API 的 Schema。
type ClusterQueue struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterQueueSpec   `json:"spec,omitempty"`
	Status ClusterQueueStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterQueueList 包含 ClusterQueue 列表。
type ClusterQueueList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterQueue `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterQueue{}, &ClusterQueueList{})
}
