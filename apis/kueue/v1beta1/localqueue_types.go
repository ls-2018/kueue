package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LocalQueueName is the name of the LocalQueue.
// 它必须是 DNS（RFC 1123）格式，最大长度为 253 个字符。
//
// +kubebuilder:validation:MaxLength=253
// +kubebuilder:validation:Pattern="^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$"
// 它必须是 DNS（RFC 1123）格式，最大长度为 253 个字符。
type LocalQueueName string

// LocalQueueSpec 定义了 LocalQueue 的期望状态
type LocalQueueSpec struct {
	// clusterQueue 是指向支持此 localQueue 的 clusterQueue 的引用。
	// +kubebuilder:validation:XValidation:rule="self == oldSelf", message="field is immutable"
	ClusterQueue ClusterQueueReference `json:"clusterQueue,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum=None;Hold;HoldAndDrain
	// +kubebuilder:default="None"
	// stopPolicy - 如果设置为非 None 的值，则 LocalQueue 被视为非活动状态，不会进行新的预留。
	// 根据其值，相关的工作负载将：
	// - None - 工作负载被接纳
	// - HoldAndDrain - 已接纳的工作负载会被驱逐，预留中的工作负载会取消预留
	// - Hold - 已接纳的工作负载会运行至完成，预留中的工作负载会取消预留
	StopPolicy *StopPolicy `json:"stopPolicy,omitempty"`

	// +optional
	// fairSharing 定义了 LocalQueue 在参与 AdmissionFairSharing 时的属性。仅当 Kueue 配置中启用 AdmissionFairSharing 时，这些值才有意义。
	FairSharing *FairSharing `json:"fairSharing,omitempty"`
}

type LocalQueueFlavorStatus struct {
	// name of the flavor.
	// flavor 的名称。
	Name ResourceFlavorReference `json:"name"`

	// resources used in the flavor.
	// +listType=set
	// +kubebuilder:validation:MaxItems=16
	// +optional
	// flavor 中使用的资源。
	Resources []corev1.ResourceName `json:"resources,omitempty"`

	// nodeLabels are labels that associate the ResourceFlavor with Nodes that
	// have the same labels.
	// +mapType=atomic
	// +kubebuilder:validation:MaxProperties=8
	// +optional
	// nodeLabels 是将 ResourceFlavor 与具有相同标签的节点关联的标签。
	NodeLabels map[string]string `json:"nodeLabels,omitempty"`

	// nodeTaints are taints that the nodes associated with this ResourceFlavor
	// have.
	// +listType=atomic
	// +kubebuilder:validation:MaxItems=8
	// +optional
	// nodeTaints 是与此 ResourceFlavor 关联的节点所具有的污点。
	NodeTaints []corev1.Taint `json:"nodeTaints,omitempty"`

	// topology is the topology that associated with this ResourceFlavor.
	//
	// This is an alpha field and requires enabling the TopologyAwareScheduling
	// feature gate.
	//
	// +optional
	// topology 是与此 ResourceFlavor 关联的拓扑。
	// 这是一个 alpha 字段，需要启用 TopologyAwareScheduling 特性门控。
	Topology *TopologyInfo `json:"topology,omitempty"`
}

type TopologyInfo struct {
	// name is the name of the topology.
	//
	// +required
	// +kubebuilder:validation:Required
	// name 是拓扑的名称。
	Name TopologyReference `json:"name"`

	// levels define the levels of topology.
	//
	// +required
	// +listType=atomic
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=8
	// levels 定义了拓扑的层级。
	Levels []string `json:"levels"`
}

// LocalQueueStatus defines the observed state of LocalQueue
// LocalQueueStatus 定义了 LocalQueue 的观测状态
type LocalQueueStatus struct {
	// PendingWorkloads is the number of Workloads in the LocalQueue not yet admitted to a ClusterQueue
	// +optional
	// PendingWorkloads 是 LocalQueue 中尚未被接纳到 ClusterQueue 的工作负载数量
	PendingWorkloads int32 `json:"pendingWorkloads"`

	// reservingWorkloads is the number of workloads in this LocalQueue
	// reserving quota in a ClusterQueue and that haven't finished yet.
	// +optional
	// reservingWorkloads 是此 LocalQueue 中正在 ClusterQueue 预留配额且尚未完成的工作负载数量。
	ReservingWorkloads int32 `json:"reservingWorkloads"`

	// admittedWorkloads is the number of workloads in this LocalQueue
	// admitted to a ClusterQueue and that haven't finished yet.
	// +optional
	// admittedWorkloads 是此 LocalQueue 中已被接纳到 ClusterQueue 且尚未完成的工作负载数量。
	AdmittedWorkloads int32 `json:"admittedWorkloads"`

	// Conditions hold the latest available observations of the LocalQueue
	// current state.
	// +optional
	// +listType=map
	// +listMapKey=type
	// +patchStrategy=merge
	// +patchMergeKey=type
	// Conditions 保存了 LocalQueue 当前状态的最新可用观测信息。
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// flavorsReservation are the reserved quotas, by flavor currently in use by the
	// workloads assigned to this LocalQueue.
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=16
	// +optional
	// flavorsReservation 是分配给此 LocalQueue 的工作负载当前使用的 flavor 的预留配额。
	FlavorsReservation []LocalQueueFlavorUsage `json:"flavorsReservation"`

	// flavorsUsage are the used quotas, by flavor currently in use by the
	// workloads assigned to this LocalQueue.
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=16
	// +optional
	// flavorsUsage 是分配给此 LocalQueue 的工作负载当前使用的 flavor 的已用配额。
	FlavorUsage []LocalQueueFlavorUsage `json:"flavorUsage"`

	// flavors lists all currently available ResourceFlavors in specified ClusterQueue.
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=16
	// +optional
	// flavors 列出了指定 ClusterQueue 中当前可用的所有 ResourceFlavor。
	Flavors []LocalQueueFlavorStatus `json:"flavors,omitempty"`

	// FairSharing contains the information about the current status of fair sharing.
	// +optional
	// FairSharing 包含有关当前公平共享状态的信息。
	FairSharing *FairSharingStatus `json:"fairSharing,omitempty"`
}

const (
	// LocalQueueActive 表示支持 LocalQueue 的 ClusterQueue 处于活动状态，LocalQueue 可以向其 ClusterQueue 提交新的工作负载。
	LocalQueueActive string = "Active"
)

type LocalQueueFlavorUsage struct {
	// name of the flavor.
	// flavor 的名称。
	Name ResourceFlavorReference `json:"name"`

	// resources lists the quota usage for the resources in this flavor.
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=16
	// resources 列出了此 flavor 中资源的配额使用情况。
	Resources []LocalQueueResourceUsage `json:"resources"`
}

type LocalQueueResourceUsage struct {
	// name of the resource.
	// 资源的名称。
	Name corev1.ResourceName `json:"name"`

	// total is the total quantity of used quota.
	// total 是已用配额的总量。
	Total resource.Quantity `json:"total,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ClusterQueue",JSONPath=".spec.clusterQueue",type=string,description="Backing ClusterQueue"
// +kubebuilder:printcolumn:name="Pending Workloads",JSONPath=".status.pendingWorkloads",type=integer,description="Number of pending workloads"
// +kubebuilder:printcolumn:name="Admitted Workloads",JSONPath=".status.admittedWorkloads",type=integer,description="Number of admitted workloads that haven't finished yet."
// +kubebuilder:resource:shortName={queue,queues,lq}

// LocalQueue is the Schema for the localQueues API
// LocalQueue 是 localQueues API 的架构
type LocalQueue struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LocalQueueSpec   `json:"spec,omitempty"`
	Status LocalQueueStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LocalQueueList contains a list of LocalQueue
// LocalQueueList 包含 LocalQueue 的列表
type LocalQueueList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LocalQueue `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LocalQueue{}, &LocalQueueList{})
}
