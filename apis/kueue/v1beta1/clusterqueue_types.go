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
// 必须为 DNS（RFC 1123）格式，最大长度为 253 个字符。
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

// ClusterQueueSpec 定义 ClusterQueue 的期望状态
// +kubebuilder:validation:XValidation:rule="!has(self.cohort) && has(self.resourceGroups) ? self.resourceGroups.all(rg, rg.flavors.all(f, f.resources.all(r, !has(r.borrowingLimit)))) : true", message="borrowingLimit must be nil when cohort is empty"
type ClusterQueueSpec struct {
	// resourceGroups 描述资源组。
	// 每个资源组定义资源列表和为这些资源提供配额的 flavor 列表。
	// 每个资源和每个 flavor 只能属于一个资源组。
	// resourceGroups 最多可有 16 个。
	// +listType=atomic
	// +kubebuilder:validation:MaxItems=16
	ResourceGroups []ResourceGroup `json:"resourceGroups,omitempty"`

	// cohort 表示该 ClusterQueue 所属的 cohort。属于同一 cohort 的 CQ 可以相互借用未使用的资源。
	//
	// CQ 只能属于一个借用 cohort。提交到引用该 CQ 的队列的工作负载可以从 cohort 中的任何 CQ 借用配额。
	// 只有 CQ 中列出的 [resource, flavor] 对的配额可以被借用。
	// 如果为空，则该 ClusterQueue 不能从其他 ClusterQueue 借用，也不能被其他 ClusterQueue 借用。
	//
	// cohort 是将 CQ 关联在一起的名称，但不引用任何对象。
	Cohort CohortReference `json:"cohort,omitempty"`

	// QueueingStrategy 表示该 ClusterQueue 下队列中工作负载的排队策略。
	// 当前支持的策略：
	//
	// - StrictFIFO：工作负载严格按创建时间排序。无法接收的旧工作负载会阻塞新工作负载的接收，即使新工作负载适合现有配额。
	// - BestEffortFIFO：工作负载按创建时间排序，但无法接收的旧工作负载不会阻塞适合现有配额的新工作负载的接收。
	//
	// +kubebuilder:default=BestEffortFIFO
	// +kubebuilder:validation:Enum=StrictFIFO;BestEffortFIFO
	QueueingStrategy QueueingStrategy `json:"queueingStrategy,omitempty"`

	// namespaceSelector 定义哪些命名空间可以向该 clusterQueue 提交工作负载。除了基本策略支持外，应使用策略代理（如 Gatekeeper）来实施更高级的策略。
	// 默认为 null，即无选择器（无命名空间有资格）。如果设置为空选择器 `{}`，则所有命名空间都有资格。
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`

	// flavorFungibility 定义在评估 flavor 时，工作负载是否应在借用或抢占前尝试下一个 flavor。
	// +kubebuilder:default={}
	FlavorFungibility *FlavorFungibility `json:"flavorFungibility,omitempty"`

	// +kubebuilder:default={}
	Preemption *ClusterQueuePreemption `json:"preemption,omitempty"`

	// admissionChecks 列出该 ClusterQueue 所需的 AdmissionChecks。不能与 AdmissionCheckStrategy 同时使用。
	// +optional
	AdmissionChecks []AdmissionCheckReference `json:"admissionChecks,omitempty"`

	// admissionCheckStrategy 定义确定哪些 ResourceFlavors 需要 AdmissionChecks 的策略列表。该属性不能与 'admissionChecks' 属性同时使用。
	// +optional
	AdmissionChecksStrategy *AdmissionChecksStrategy `json:"admissionChecksStrategy,omitempty"`

	// stopPolicy - 如果设置为非 None，则该 ClusterQueue 被视为非活动状态，不会进行新的配额预留。
	//
	// 根据其值，相关工作负载将：
	//
	// - None - 工作负载被接收
	// - HoldAndDrain - 已接收的工作负载被驱逐，预留中的工作负载将取消预留。
	// - Hold - 已接收的工作负载将运行至完成，预留中的工作负载将取消预留。
	//
	// +optional
	// +kubebuilder:validation:Enum=None;Hold;HoldAndDrain
	// +kubebuilder:default="None"
	StopPolicy *StopPolicy `json:"stopPolicy,omitempty"`

	// fairSharing 定义该 ClusterQueue 参与公平共享时的属性。仅在 Kueue 配置中启用公平共享时才相关。
	// +optional
	FairSharing *FairSharing `json:"fairSharing,omitempty"`

	// admissionScope 表示 ClusterQueue 是否使用公平共享资源配额
	// +optional
	AdmissionScope *AdmissionScope `json:"admissionScope,omitempty"`
}

// AdmissionChecksStrategy defines a strategy for a AdmissionCheck.
// AdmissionChecksStrategy 定义 AdmissionCheck 的策略。
type AdmissionChecksStrategy struct {
	// admissionChecks is a list of strategies for AdmissionChecks
	// admissionChecks 是 AdmissionChecks 的策略列表
	AdmissionChecks []AdmissionCheckStrategyRule `json:"admissionChecks,omitempty"`
}

// AdmissionCheckStrategyRule defines rules for a single AdmissionCheck
// AdmissionCheckStrategyRule 定义单个 AdmissionCheck 的规则
type AdmissionCheckStrategyRule struct {
	// name is an AdmissionCheck's name.
	// name 是 AdmissionCheck 的名称。
	Name AdmissionCheckReference `json:"name"`

	// onFlavors is a list of ResourceFlavors' names that this AdmissionCheck should run for.
	// onFlavors 是此 AdmissionCheck 应运行的 ResourceFlavors 名称列表。
	// If empty, the AdmissionCheck will run for all workloads submitted to the ClusterQueue.
	// 如果为空，则 AdmissionCheck 将对提交到 ClusterQueue 的所有工作负载运行。
	// +optional
	OnFlavors []ResourceFlavorReference `json:"onFlavors,omitempty"`
}

type QueueingStrategy string

const (
	// StrictFIFO means that workloads of the same priority are ordered strictly by creation time.
	// Older workloads that can't be admitted will block admitting newer
	// workloads even if they fit available quota.
	StrictFIFO QueueingStrategy = "StrictFIFO"

	// BestEffortFIFO means that workloads of the same priority are ordered by creation time,
	// however older workloads that can't be admitted will not block
	// admitting newer workloads that fit existing quota.
	BestEffortFIFO QueueingStrategy = "BestEffortFIFO"
)

// +kubebuilder:validation:XValidation:rule="self.flavors.all(x, size(x.resources) == size(self.coveredResources))", message="flavors must have the same number of resources as the coveredResources"
type ResourceGroup struct {
	// coveredResources 是此组中 flavors 覆盖的资源列表。
	// 例如：cpu、memory、vendor.com/gpu。
	// 此列表不能为空，且最多包含 16 个资源。
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	CoveredResources []corev1.ResourceName `json:"coveredResources"`

	// flavors 是为此组资源提供配额的 flavor 列表。
	// 通常，不同的 flavor 代表不同的硬件型号（如 gpu 型号、cpu 架构）或定价模式（按需 vs 竞价 cpu）。
	// 每个 flavor 必须以与 .resources 字段相同的顺序列出此组的所有资源。
	// 此列表不能为空，且最多包含 16 个 flavor。
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	Flavors []FlavorQuotas `json:"flavors"`
}

type FlavorQuotas struct {
	// name of this flavor. The name should match the .metadata.name of a
	// ResourceFlavor. If a matching ResourceFlavor does not exist, the
	// ClusterQueue will have an Active condition set to False.
	// 此 flavor 的名称。名称应与 ResourceFlavor 的 .metadata.name 匹配。如果不存在匹配的 ResourceFlavor，则 ClusterQueue 的 Active 状态将为 False。
	Name ResourceFlavorReference `json:"name"`

	// resources is the list of quotas for this flavor per resource.
	// There could be up to 16 resources.
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	Resources []ResourceQuota `json:"resources"`
}

type ResourceQuota struct {
	// name of this resource.
	// 此资源的名称。
	Name corev1.ResourceName `json:"name"`

	// nominalQuota 必须为非负数。
	// nominalQuota 应代表集群中可用于运行作业的资源（扣除系统组件和非 kueue 管理的 pod 所消耗的资源后）。在自动扩缩的集群中，nominalQuota 应考虑如 Kubernetes cluster-autoscaler 等组件可提供的资源。
	//
	// 如果 ClusterQueue 属于 cohort，则每个（flavor, resource）组合的配额总和定义了 cohort 中 ClusterQueue 可分配的最大数量。
	NominalQuota resource.Quantity `json:"nominalQuota"`

	// borrowingLimit 是该 ClusterQueue 允许从同一 cohort 中其他 ClusterQueue 未使用配额中借用的 [flavor, resource] 组合的最大配额。
	// 总体上，在某一时刻，ClusterQueue 中的工作负载可以消耗等于 nominalQuota+borrowingLimit 的配额，前提是 cohort 中其他 ClusterQueue 有足够的未使用配额。
	// 如果为 null，表示没有借用上限。
	// 如果不为 null，则必须为非负数。
	// 如果 spec.cohort 为空，则 borrowingLimit 必须为 null。
	// +optional
	BorrowingLimit *resource.Quantity `json:"borrowingLimit,omitempty"` // 借别人的最大值

	// lendingLimit 是该 ClusterQueue 可以借给同一 cohort 中其他 ClusterQueue 的 [flavor, resource] 组合的最大未使用配额。
	// 总体上，在某一时刻，ClusterQueue 为其专用保留的配额等于 nominalQuota - lendingLimit。
	// 如果为 null，表示没有出借上限，意味着所有 nominalQuota 都可以被 cohort 中其他 ClusterQueue 借用。
	// 如果不为 null，则必须为非负数。
	// 如果 spec.cohort 为空，则 lendingLimit 必须为 null。
	// 此字段为 beta 阶段，默认启用。
	// +optional
	LendingLimit *resource.Quantity `json:"lendingLimit,omitempty"` // 借出最大值
}

// ResourceFlavorReference 是 ResourceFlavor 的名称。
// +kubebuilder:validation:MaxLength=253
// +kubebuilder:validation:Pattern="^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$"
type ResourceFlavorReference string

// ClusterQueueStatus 定义 ClusterQueue 的观测状态
type ClusterQueueStatus struct {
	// flavorsReservation 是当前被分配到此 ClusterQueue 的工作负载所使用的按 flavor 划分的预留配额。
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=16
	// +optional
	FlavorsReservation []FlavorUsage `json:"flavorsReservation"`

	// flavorsUsage 是当前被此 ClusterQueue 接收的工作负载所使用的按 flavor 划分的已用配额。
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=16
	// +optional
	FlavorsUsage []FlavorUsage `json:"flavorsUsage"`

	// pendingWorkloads 是当前等待被此 clusterQueue 接收的工作负载数量。
	// +optional
	PendingWorkloads int32 `json:"pendingWorkloads"`

	// reservingWorkloads 是当前在此 clusterQueue 中预留配额的工作负载数量。
	// +optional
	ReservingWorkloads int32 `json:"reservingWorkloads"`

	// admittedWorkloads 是当前已被此 clusterQueue 接收且尚未完成的工作负载数量。
	// +optional
	AdmittedWorkloads int32 `json:"admittedWorkloads"`

	// conditions 保存 ClusterQueue 当前状态的最新可用观测信息。
	// +optional
	// +listType=map
	// +listMapKey=type
	// +patchStrategy=merge
	// +patchMergeKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// PendingWorkloadsStatus 包含关于 cluster queue 中当前待处理工作负载状态的信息。
	// 已弃用：此字段将在 v1beta2 移除，请改用 VisibilityOnDemand。
	// +optional
	PendingWorkloadsStatus *ClusterQueuePendingWorkloadsStatus `json:"pendingWorkloadsStatus"`

	// fairSharing 包含此 ClusterQueue 参与公平共享时的当前状态。
	// 仅在 Kueue 配置中启用公平共享时记录。
	// +optional
	FairSharing *FairSharingStatus `json:"fairSharing,omitempty"`
}

type ClusterQueuePendingWorkloadsStatus struct {
	// Head 包含最前面的待处理工作负载列表。
	// +listType=atomic
	// +optional
	Head []ClusterQueuePendingWorkload `json:"clusterQueuePendingWorkload"`

	// LastChangeTime 表示结构最后一次变更的时间。
	LastChangeTime metav1.Time `json:"lastChangeTime"`
}

// ClusterQueuePendingWorkload contains the information identifying a pending workload
// in the cluster queue.
// ClusterQueuePendingWorkload 包含标识 cluster queue 中待处理工作负载的信息。
type ClusterQueuePendingWorkload struct {
	// Name 表示待处理工作负载的名称。
	Name string `json:"name"`

	// Namespace 表示待处理工作负载的命名空间。
	Namespace string `json:"namespace"`
}

type FlavorUsage struct {
	// flavor 的名称。
	Name ResourceFlavorReference `json:"name"`

	// resources 列出此 flavor 中资源的配额使用情况。
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=16
	Resources []ResourceUsage `json:"resources"`
}

type ResourceUsage struct {
	// 资源的名称
	Name corev1.ResourceName `json:"name"`

	// total 是已用配额的总量，包括从 cohort 借用的数量。
	Total resource.Quantity `json:"total,omitempty"`

	// Borrowed 是从 cohort 借用的配额数量。换句话说，是超出 nominalQuota 的已用配额。
	Borrowed resource.Quantity `json:"borrowed,omitempty"`
}

const (
	// ClusterQueueActive indicates that the ClusterQueue can admit new workloads and its quota
	// can be borrowed by other ClusterQueues in the same cohort.
	ClusterQueueActive string = "Active"
)

// PreemptionPolicy defines the preemption policy.
// PreemptionPolicy 定义抢占策略。
type PreemptionPolicy string

const (
	PreemptionPolicyNever                     PreemptionPolicy = "Never"
	PreemptionPolicyAny                       PreemptionPolicy = "Any"
	PreemptionPolicyLowerPriority             PreemptionPolicy = "LowerPriority"
	PreemptionPolicyLowerOrNewerEqualPriority PreemptionPolicy = "LowerOrNewerEqualPriority"
)

// FlavorFungibilityPolicy defines the policy for a flavor.
// FlavorFungibilityPolicy 定义 flavor 的策略。
type FlavorFungibilityPolicy string

const (
	Borrow        FlavorFungibilityPolicy = "Borrow"
	Preempt       FlavorFungibilityPolicy = "Preempt"
	TryNextFlavor FlavorFungibilityPolicy = "TryNextFlavor"
)

// FlavorFungibility 决定工作负载在当前 flavor 借用或抢占前是否应尝试下一个 flavor。
type FlavorFungibility struct {
	// whenCanBorrow 决定工作负载在当前 flavor 借用前是否应尝试下一个 flavor。可选值：
	//
	// - `Borrow`（默认）：如果可以借用，则在当前 flavor 分配。
	// - `TryNextFlavor`：即使当前 flavor 有足够资源可借用，也尝试下一个 flavor。
	//
	// +kubebuilder:validation:Enum={Borrow,TryNextFlavor}
	// +kubebuilder:default="Borrow"
	WhenCanBorrow FlavorFungibilityPolicy `json:"whenCanBorrow,omitempty"`
	// whenCanPreempt 决定工作负载在当前 flavor 抢占前是否应尝试下一个 flavor。可选值：
	//
	// - `Preempt`：如果可以抢占一些工作负载，则在当前 flavor 分配。
	// - `TryNextFlavor`（默认）：即使当前 flavor 有足够可抢占的候选，也尝试下一个 flavor。
	//
	// +kubebuilder:validation:Enum={Preempt,TryNextFlavor}
	// +kubebuilder:default="TryNextFlavor"
	WhenCanPreempt FlavorFungibilityPolicy `json:"whenCanPreempt,omitempty"`
}

// ClusterQueuePreemption 包含从本 ClusterQueue 或其 cohort 抢占工作负载的策略。
//
// 抢占可配置在以下场景下工作：
//
//   - 当工作负载适合 ClusterQueue 的 nominal quota，但该配额当前被 cohort 中其他 ClusterQueue 借用时。我们会抢占其他 ClusterQueue 的工作负载，以允许本 ClusterQueue 回收其 nominal quota。通过 reclaimWithinCohort 配置。
//   - 当工作负载不适合 ClusterQueue 的 nominal quota，且 ClusterQueue 中有更低优先级的已接收工作负载时。通过 withinClusterQueue 配置。
//   - 当工作负载可以通过借用和抢占 cohort 中低优先级工作负载来适配时。通过 borrowWithinCohort 配置。
//   - 启用 FairSharing 时，为保持未用资源的公平分配。详见 FairSharing 文档。
//
// 抢占算法会尝试找到最小集合的工作负载进行抢占，以容纳待处理工作负载，优先抢占优先级较低的工作负载。
// +kubebuilder:validation:XValidation:rule="!(self.reclaimWithinCohort == 'Never' && has(self.borrowWithinCohort) &&  self.borrowWithinCohort.policy != 'Never')", message="reclaimWithinCohort=Never and borrowWithinCohort.Policy!=Never"
type ClusterQueuePreemption struct {
	// reclaimWithinCohort 决定待处理工作负载是否可以抢占 cohort 中超出 nominal quota 的其他 ClusterQueue 的工作负载。可选值：
	//
	// - `Never`（默认）：不抢占 cohort 中的工作负载。
	// - `LowerPriority`：**经典抢占**：如果待处理工作负载适合其 ClusterQueue 的 nominal quota，仅抢占 cohort 中优先级较低的工作负载。**公平共享**：仅抢占 cohort 中优先级较低且满足公平共享抢占策略的工作负载。
	// - `Any`：**经典抢占**：如果待处理工作负载适合其 ClusterQueue 的 nominal quota，可抢占 cohort 中任何工作负载，无论优先级。**公平共享**：抢占 cohort 中满足公平共享抢占策略的工作负载。
	//
	// +kubebuilder:default=Never
	// +kubebuilder:validation:Enum=Never;LowerPriority;Any
	ReclaimWithinCohort PreemptionPolicy `json:"reclaimWithinCohort,omitempty"`

	// +kubebuilder:default={}
	BorrowWithinCohort *BorrowWithinCohort `json:"borrowWithinCohort,omitempty"`

	// withinClusterQueue 决定不适合其 ClusterQueue nominal quota 的待处理工作负载是否可以抢占 ClusterQueue 中的活动工作负载。可选值：
	//
	// - `Never`（默认）：不抢占 ClusterQueue 中的工作负载。
	// - `LowerPriority`：仅抢占 ClusterQueue 中优先级低于待处理工作负载的工作负载。
	// - `LowerOrNewerEqualPriority`：仅抢占 ClusterQueue 中优先级低于待处理工作负载，或优先级相同但比待处理工作负载新的工作负载。
	//
	// +kubebuilder:default=Never
	// +kubebuilder:validation:Enum=Never;LowerPriority;LowerOrNewerEqualPriority
	WithinClusterQueue PreemptionPolicy `json:"withinClusterQueue,omitempty"`
}

// BorrowWithinCohort 包含允许在借用时抢占 cohort 内工作负载的配置。仅适用于经典抢占，不适用于公平共享。
type BorrowWithinCohort struct {
	// policy 决定借用时在 cohort 内回收配额的抢占策略。
	// 可选值：
	// - `Never`（默认）：不允许借用工作负载在 cohort 内其他 ClusterQueue 抢占。
	// - `LowerPriority`：允许借用工作负载在 cohort 内其他 ClusterQueue 抢占，但仅限于被抢占工作负载优先级较低时。
	//
	// +kubebuilder:default=Never
	// +kubebuilder:validation:Enum=Never;LowerPriority
	Policy BorrowWithinCohortPolicy `json:"policy,omitempty"`

	// maxPriorityThreshold 允许将可被借用工作负载抢占的工作负载范围限制为优先级小于等于指定阈值的工作负载。
	// 如果未指定阈值，则任何满足策略的工作负载都可以被借用工作负载抢占。
	//
	// +optional
	MaxPriorityThreshold *int32 `json:"maxPriorityThreshold,omitempty"`
}

// BorrowWithinCohortPolicy defines the policy for borrowing within cohort.
// BorrowWithinCohortPolicy 定义 cohort 内借用时的策略。
type BorrowWithinCohortPolicy string

const (
	BorrowWithinCohortPolicyNever         BorrowWithinCohortPolicy = "Never"
	BorrowWithinCohortPolicyLowerPriority BorrowWithinCohortPolicy = "LowerPriority"
)

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Cluster,shortName={cq}
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Cohort",JSONPath=".spec.cohort",type=string,description="Cohort that this ClusterQueue belongs to"
// +kubebuilder:printcolumn:name="Strategy",JSONPath=".spec.queueingStrategy",type=string,description="The queueing strategy used to prioritize workloads",priority=1
// +kubebuilder:printcolumn:name="Pending Workloads",JSONPath=".status.pendingWorkloads",type=integer,description="Number of pending workloads"
// +kubebuilder:printcolumn:name="Admitted Workloads",JSONPath=".status.admittedWorkloads",type=integer,description="Number of admitted workloads that haven't finished yet",priority=1

// ClusterQueue is the Schema for the clusterQueue API.
type ClusterQueue struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterQueueSpec   `json:"spec,omitempty"`
	Status ClusterQueueStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterQueueList contains a list of ClusterQueue
type ClusterQueueList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterQueue `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterQueue{}, &ClusterQueueList{})
}
