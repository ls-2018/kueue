package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterQueue Active condition reasons.
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

// ClusterQueueReference is the name of the ClusterQueue.
// ClusterQueueReference 是 ClusterQueue 的名称。
// 它必须是 DNS（RFC 1123）格式，最大长度为 253 个字符。
//
// +kubebuilder:validation:MaxLength=253
// +kubebuilder:validation:Pattern="^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$"
type ClusterQueueReference string

// CohortReference is the name of the Cohort.
// CohortReference 是 Cohort 的名称。
//
// Cohort 名称的校验等同于对象名称的校验：DNS（RFC 1123）中的子域名。
// +kubebuilder:validation:MaxLength=253
// +kubebuilder:validation:Pattern="^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$"
type CohortReference string

// ClusterQueueSpec defines the desired state of ClusterQueue
// ClusterQueueSpec 定义了 ClusterQueue 的期望状态
// +kubebuilder:validation:XValidation:rule="!has(self.cohort) && has(self.resourceGroups) ? self.resourceGroups.all(rg, rg.flavors.all(f, f.resources.all(r, !has(r.borrowingLimit)))) : true", message="borrowingLimit must be nil when cohort is empty"
type ClusterQueueSpec struct {
	// resourceGroups describes groups of resources.
	// resourceGroups 描述资源组。
	// 每个资源组定义了资源列表和为这些资源提供配额的 flavor 列表。
	// 每个资源和每个 flavor 只能属于一个资源组。
	// resourceGroups 最多可有 16 个。
	// +listType=atomic
	// +kubebuilder:validation:MaxItems=16
	ResourceGroups []ResourceGroup `json:"resourceGroups,omitempty"`

	// cohort that this ClusterQueue belongs to. CQs that belong to the
	// same cohort can borrow unused resources from each other.
	// cohort 表示该 ClusterQueue 所属的组。属于同一 cohort 的 CQ 可以相互借用未使用的资源。
	//
	// 一个 CQ 只能属于一个借用 cohort。提交到引用此 CQ 的队列的工作负载可以从 cohort 中的任何 CQ 借用配额。
	// 只有 CQ 中列出的 [resource, flavor] 对的配额可以被借用。
	// 如果为空，则该 ClusterQueue 不能从其他 ClusterQueue 借用，也不能被借用。
	//
	// cohort 是将 CQ 关联在一起的名称，但它不引用任何对象。
	Cohort CohortReference `json:"cohort,omitempty"`

	// QueueingStrategy indicates the queueing strategy of the workloads
	// across the queues in this ClusterQueue.
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

	// namespaceSelector defines which namespaces are allowed to submit workloads to
	// this clusterQueue. Beyond this basic support for policy, a policy agent like
	// Gatekeeper should be used to enforce more advanced policies.
	// namespaceSelector 定义哪些命名空间可以向该 clusterQueue 提交工作负载。更高级的策略建议使用 Gatekeeper 等策略代理实现。
	// 默认为 null，即无命名空间可用。
	// 如果设置为空选择器 `{}`，则所有命名空间都可用。
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`

	// flavorFungibility defines whether a workload should try the next flavor
	// before borrowing or preempting in the flavor being evaluated.
	// flavorFungibility 定义在借用或抢占当前 flavor 前，工作负载是否应尝试下一个 flavor。
	// +kubebuilder:default={}
	FlavorFungibility *FlavorFungibility `json:"flavorFungibility,omitempty"`

	// +kubebuilder:default={}
	Preemption *ClusterQueuePreemption `json:"preemption,omitempty"`

	// admissionChecks lists the AdmissionChecks required by this ClusterQueue.
	// admissionChecks 列出该 ClusterQueue 所需的 AdmissionChecks。
	// 不能与 AdmissionCheckStrategy 同时使用。
	// +optional
	AdmissionChecks []AdmissionCheckReference `json:"admissionChecks,omitempty"`

	// admissionCheckStrategy defines a list of strategies to determine which ResourceFlavors require AdmissionChecks.
	// admissionCheckStrategy 定义用于确定哪些 ResourceFlavors 需要 AdmissionChecks 的策略列表。
	// 此属性不能与 'admissionChecks' 属性同时使用。
	// +optional
	AdmissionChecksStrategy *AdmissionChecksStrategy `json:"admissionChecksStrategy,omitempty"`

	// stopPolicy - if set to a value different from None, the ClusterQueue is considered Inactive, no new reservation being
	// made.
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

	// fairSharing defines the properties of the ClusterQueue when
	// participating in FairSharing.  The values are only relevant
	// if FairSharing is enabled in the Kueue configuration.
	// fairSharing 定义该 ClusterQueue 参与公平共享时的属性。仅在 Kueue 配置启用公平共享时有效。
	// +optional
	FairSharing *FairSharing `json:"fairSharing,omitempty"`

	// admissionScope indicates whether ClusterQueue uses the Admission Fair Sharing
	// admissionScope 指示 ClusterQueue 是否使用 Admission Fair Sharing。
	// +optional
	AdmissionScope *AdmissionScope `json:"admissionScope,omitempty"`
}

// AdmissionChecksStrategy defines a strategy for a AdmissionCheck.
type AdmissionChecksStrategy struct {
	// admissionChecks is a list of strategies for AdmissionChecks
	AdmissionChecks []AdmissionCheckStrategyRule `json:"admissionChecks,omitempty"`
}

// AdmissionCheckStrategyRule defines rules for a single AdmissionCheck
type AdmissionCheckStrategyRule struct {
	// name is an AdmissionCheck's name.
	Name AdmissionCheckReference `json:"name"`

	// onFlavors is a list of ResourceFlavors' names that this AdmissionCheck should run for.
	// If empty, the AdmissionCheck will run for all workloads submitted to the ClusterQueue.
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
	// coveredResources is the list of resources covered by the flavors in this
	// group.
	// Examples: cpu, memory, vendor.com/gpu.
	// The list cannot be empty and it can contain up to 16 resources.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	CoveredResources []corev1.ResourceName `json:"coveredResources"`

	// flavors is the list of flavors that provide the resources of this group.
	// Typically, different flavors represent different hardware models
	// (e.g., gpu models, cpu architectures) or pricing models (on-demand vs spot
	// cpus).
	// Each flavor MUST list all the resources listed for this group in the same
	// order as the .resources field.
	// The list cannot be empty and it can contain up to 16 flavors.
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	Flavors []FlavorQuotas `json:"flavors"`
}

type FlavorQuotas struct {
	// name of this flavor. The name should match the .metadata.name of a
	// 此 flavor 的名称。名称应与 ResourceFlavor 的 .metadata.name 匹配。
	// 如果不存在匹配的 ResourceFlavor，ClusterQueue 的 Active 条件将被设置为 False。
	Name ResourceFlavorReference `json:"name"`

	// resources is the list of quotas for this flavor per resource.
	// resources 是该 flavor 针对每种资源的配额列表。
	// 最多可有 16 种资源。
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

	// nominalQuota is the quantity of this resource that is available for
	// Workloads admitted by this ClusterQueue at a point in time.
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

// ResourceFlavorReference is the name of the ResourceFlavor.
// ResourceFlavorReference 是 ResourceFlavor 的名称。
// +kubebuilder:validation:MaxLength=253
// +kubebuilder:validation:Pattern="^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$"
type ResourceFlavorReference string

// ClusterQueueStatus defines the observed state of ClusterQueue
// ClusterQueueStatus 定义了 ClusterQueue 的当前状态。
type ClusterQueueStatus struct {
	// flavorsReservation are the reserved quotas, by flavor, currently in use by the
	// workloads assigned to this ClusterQueue.
	// flavorsReservation 是当前分配给该 ClusterQueue 的工作负载的预留配额，按 flavor 分组。
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=16
	// +optional
	FlavorsReservation []FlavorUsage `json:"flavorsReservation"`

	// flavorsUsage are the used quotas, by flavor, currently in use by the
	// workloads admitted in this ClusterQueue.
	// flavorsUsage 是当前分配给该 ClusterQueue 的工作负载的已使用配额，按 flavor 分组。
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=16
	// +optional
	FlavorsUsage []FlavorUsage `json:"flavorsUsage"`

	// pendingWorkloads is the number of workloads currently waiting to be
	// admitted to this clusterQueue.
	// pendingWorkloads 是当前等待被接纳到该 clusterQueue 的工作负载数量。
	// +optional
	PendingWorkloads int32 `json:"pendingWorkloads"`

	// reservingWorkloads is the number of workloads currently reserving quota in this
	// clusterQueue.
	// reservingWorkloads 是当前预留配额的工作负载数量。
	// +optional
	ReservingWorkloads int32 `json:"reservingWorkloads"`

	// admittedWorkloads is the number of workloads currently admitted to this
	// clusterQueue and haven't finished yet.
	// admittedWorkloads 是当前已接纳到该 clusterQueue 且尚未完成的工作负载数量。
	// +optional
	AdmittedWorkloads int32 `json:"admittedWorkloads"`

	// conditions hold the latest available observations of the ClusterQueue
	// current state.
	// conditions 持有该 ClusterQueue 的最新可用观察结果。
	// +optional
	// +listType=map
	// +listMapKey=type
	// +patchStrategy=merge
	// +patchMergeKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// PendingWorkloadsStatus contains the information exposed about the current
	// status of the pending workloads in the cluster queue.
	// PendingWorkloadsStatus 包含关于集群队列中等待工作负载状态的信息。
	// 弃用：此字段将在 v1beta2 中移除，请改用 VisibilityOnDemand
	// (https://kueue.sigs.k8s.io/docs/tasks/manage/monitor_pending_workloads/pending_workloads_on_demand/)
	// 替代。
	// +optional
	PendingWorkloadsStatus *ClusterQueuePendingWorkloadsStatus `json:"pendingWorkloadsStatus"`

	// fairSharing contains the current state for this ClusterQueue
	// when participating in Fair Sharing.
	// fairSharing 仅在 Kueue 配置启用公平共享时记录该 ClusterQueue 的当前状态。
	// +optional
	FairSharing *FairSharingStatus `json:"fairSharing,omitempty"`
}

type ClusterQueuePendingWorkloadsStatus struct {
	// Head contains the list of top pending workloads.
	// +listType=atomic
	// +optional
	Head []ClusterQueuePendingWorkload `json:"clusterQueuePendingWorkload"`

	// LastChangeTime indicates the time of the last change of the structure.
	LastChangeTime metav1.Time `json:"lastChangeTime"`
}

// ClusterQueuePendingWorkload contains the information identifying a pending workload
// in the cluster queue.
type ClusterQueuePendingWorkload struct {
	// Name indicates the name of the pending workload.
	Name string `json:"name"`

	// Namespace indicates the name of the pending workload.
	Namespace string `json:"namespace"`
}

type FlavorUsage struct {
	// name of the flavor.
	Name ResourceFlavorReference `json:"name"`

	// resources lists the quota usage for the resources in this flavor.
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=16
	Resources []ResourceUsage `json:"resources"`
}

type ResourceUsage struct {
	// name of the resource
	Name corev1.ResourceName `json:"name"`

	// total is the total quantity of used quota, including the amount borrowed
	// from the cohort.
	Total resource.Quantity `json:"total,omitempty"`

	// Borrowed is quantity of quota that is borrowed from the cohort. In other
	// words, it's the used quota that is over the nominalQuota.
	Borrowed resource.Quantity `json:"borrowed,omitempty"`
}

const (
	// ClusterQueueActive indicates that the ClusterQueue can admit new workloads and its quota
	// can be borrowed by other ClusterQueues in the same cohort.
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

// FlavorFungibility determines whether a workload should try the next flavor
// before borrowing or preempting in current flavor.
type FlavorFungibility struct {
	// whenCanBorrow determines whether a workload should try the next flavor
	// before borrowing in current flavor. The possible values are:
	//
	// - `Borrow` (default): allocate in current flavor if borrowing
	//   is possible.
	// - `TryNextFlavor`: try next flavor even if the current
	//   flavor has enough resources to borrow.
	//
	// +kubebuilder:validation:Enum={Borrow,TryNextFlavor}
	// +kubebuilder:default="Borrow"
	WhenCanBorrow FlavorFungibilityPolicy `json:"whenCanBorrow,omitempty"`
	// whenCanPreempt determines whether a workload should try the next flavor
	// before borrowing in current flavor. The possible values are:
	//
	// - `Preempt`: allocate in current flavor if it's possible to preempt some workloads.
	// - `TryNextFlavor` (default): try next flavor even if there are enough
	//   candidates for preemption in the current flavor.
	//
	// +kubebuilder:validation:Enum={Preempt,TryNextFlavor}
	// +kubebuilder:default="TryNextFlavor"
	WhenCanPreempt FlavorFungibilityPolicy `json:"whenCanPreempt,omitempty"`
}

// ClusterQueuePreemption contains policies to preempt Workloads from this
// ClusterQueue or the ClusterQueue's cohort.
//
// Preemption may be configured to work in the following scenarios:
//
//   - When a Workload fits within the nominal quota of the ClusterQueue, but
//     the quota is currently borrowed by other ClusterQueues in the cohort.
//     We preempt workloads in other ClusterQueues to allow this ClusterQueue to
//     reclaim its nominal quota. Configured using reclaimWithinCohort.
//   - When a Workload doesn't fit within the nominal quota of the ClusterQueue
//     and there are admitted Workloads in the ClusterQueue with lower priority.
//     Configured using withinClusterQueue.
//   - When a Workload may fit while both borrowing and preempting
//     low priority workloads in the Cohort. Configured using borrowWithinCohort.
//   - When FairSharing is enabled, to maintain fair distribution of
//     unused resources. See FairSharing documentation.
//
// The preemption algorithm tries to find a minimal set of Workloads to
// preempt to accomomdate the pending Workload, preempting Workloads with
// lower priority first.
// +kubebuilder:validation:XValidation:rule="!(self.reclaimWithinCohort == 'Never' && has(self.borrowWithinCohort) &&  self.borrowWithinCohort.policy != 'Never')", message="reclaimWithinCohort=Never and borrowWithinCohort.Policy!=Never"
type ClusterQueuePreemption struct {
	// reclaimWithinCohort determines whether a pending Workload can preempt
	// Workloads from other ClusterQueues in the cohort that are using more than
	// their nominal quota. The possible values are:
	//
	// - `Never` (default): do not preempt Workloads in the cohort.
	// - `LowerPriority`: **Classic Preemption** if the pending Workload
	//   fits within the nominal quota of its ClusterQueue, only preempt
	//   Workloads in the cohort that have lower priority than the pending
	//   Workload. **Fair Sharing** only preempt Workloads in the cohort that
	//   have lower priority than the pending Workload and that satisfy the
	//   Fair Sharing preemptionStategies.
	// - `Any`: **Classic Preemption** if the pending Workload fits within
	//    the nominal quota of its ClusterQueue, preempt any Workload in the
	//    cohort, irrespective of priority. **Fair Sharing** preempt Workloads
	//    in the cohort that satisfy the Fair Sharing preemptionStrategies.
	//
	// +kubebuilder:default=Never
	// +kubebuilder:validation:Enum=Never;LowerPriority;Any
	ReclaimWithinCohort PreemptionPolicy `json:"reclaimWithinCohort,omitempty"`

	// +kubebuilder:default={}
	BorrowWithinCohort *BorrowWithinCohort `json:"borrowWithinCohort,omitempty"`

	// withinClusterQueue determines whether a pending Workload that doesn't fit
	// within the nominal quota for its ClusterQueue, can preempt active Workloads in
	// the ClusterQueue. The possible values are:
	//
	// - `Never` (default): do not preempt Workloads in the ClusterQueue.
	// - `LowerPriority`: only preempt Workloads in the ClusterQueue that have
	//   lower priority than the pending Workload.
	// - `LowerOrNewerEqualPriority`: only preempt Workloads in the ClusterQueue that
	//   either have a lower priority than the pending workload or equal priority
	//   and are newer than the pending workload.
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

// BorrowWithinCohort contains configuration which allows to preempt workloads
// within cohort while borrowing. It only works with Classical Preemption,
// __not__ with Fair Sharing.
type BorrowWithinCohort struct {
	// policy determines the policy for preemption to reclaim quota within cohort while borrowing.
	// policy 确定在借用时从 cohort 中抢占工作负载的策略。
	// 可能的值是：
	// - `Never` (default): 不允许抢占，在 cohort 中的其他 ClusterQueues 中，对于一个借用工作负载。
	// - `LowerPriority`: 允许抢占，在 cohort 中的其他 ClusterQueues 中，对于一个借用工作负载，但仅当抢占的工作负载优先级较低时。
	//
	// +kubebuilder:default=Never
	// +kubebuilder:validation:Enum=Never;LowerPriority
	Policy BorrowWithinCohortPolicy `json:"policy,omitempty"`

	// maxPriorityThreshold allows to restrict the set of workloads which
	// might be preempted by a borrowing workload, to only workloads with
	// priority less than or equal to the specified threshold priority.
	// When the threshold is not specified, then any workload satisfying the
	// policy can be preempted by the borrowing workload.
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
