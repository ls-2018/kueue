package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LocalQueueName 是 LocalQueue 的名称。
// 它必须是 DNS（RFC 1123）格式，最大长度为 253 个字符。
//
// +kubebuilder:validation:MaxLength=253
// +kubebuilder:validation:Pattern="^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$"
type LocalQueueName string

// LocalQueueSpec 定义 LocalQueue 的期望状态
type LocalQueueSpec struct {
	// clusterQueue 是对支持此 localQueue 的 clusterQueue 的引用。
	// +kubebuilder:validation:XValidation:rule="self == oldSelf", message="field is immutable"
	ClusterQueue ClusterQueueReference `json:"clusterQueue,omitempty"`

	// stopPolicy - 如果设置为非 None 的值，则 LocalQueue 被视为非活跃状态，
	// 不会进行新的资源预留。
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

	// fairSharing 定义参与准入公平共享时 LocalQueue 的属性。
	// 这些值仅在 Kueue 配置中启用准入公平共享时相关。
	// +optional
	FairSharing *FairSharing `json:"fairSharing,omitempty"`
}

type LocalQueueFlavorStatus struct {
	// flavor 的名称。
	// +required
	// +kubebuilder:validation:Required
	Name ResourceFlavorReference `json:"name"`

	// flavor 中使用的资源。
	// +listType=set
	// +kubebuilder:validation:MaxItems=16
	// +optional
	Resources []corev1.ResourceName `json:"resources,omitempty"`

	// nodeLabels 是将 ResourceFlavor 与具有相同标签的节点关联的标签。
	// +mapType=atomic
	// +kubebuilder:validation:MaxProperties=8
	// +optional
	NodeLabels map[string]string `json:"nodeLabels,omitempty"`

	// nodeTaints 是与此 ResourceFlavor 关联的节点所具有的污点。
	// +listType=atomic
	// +kubebuilder:validation:MaxItems=8
	// +optional
	NodeTaints []corev1.Taint `json:"nodeTaints,omitempty"`

	// topology 是与此 ResourceFlavor 关联的拓扑。
	//
	// 这是一个 alpha 字段，需要启用 TopologyAwareScheduling 功能门控。
	//
	// +optional
	Topology *TopologyInfo `json:"topology,omitempty"`
}

type TopologyInfo struct {
	// name 是拓扑的名称。
	//
	// +required
	// +kubebuilder:validation:Required
	Name TopologyReference `json:"name"`

	// levels 定义拓扑的级别。
	//
	// +required
	// +listType=atomic
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=8
	Levels []string `json:"levels"`
}

// LocalQueueStatus 定义 LocalQueue 的观察状态
type LocalQueueStatus struct {
	// PendingWorkloads 是 LocalQueue 中尚未被接纳到 ClusterQueue 的工作负载数量
	// +optional
	PendingWorkloads int32 `json:"pendingWorkloads"`

	// reservingWorkloads 是此 LocalQueue 中在 ClusterQueue 中预留配额且尚未完成的工作负载数量。
	// +optional
	ReservingWorkloads int32 `json:"reservingWorkloads"`

	// admittedWorkloads 是此 LocalQueue 中已被接纳到 ClusterQueue 且尚未完成的工作负载数量。
	// +optional
	AdmittedWorkloads int32 `json:"admittedWorkloads"`

	// Conditions 持有 LocalQueue 当前状态的最新可用观察结果。
	// +optional
	// +listType=map
	// +listMapKey=type
	// +patchStrategy=merge
	// +patchMergeKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// flavorsReservation 是按 flavor 分组的预留配额，当前由分配给此 LocalQueue 的工作负载使用。
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=16
	// +optional
	FlavorsReservation []LocalQueueFlavorUsage `json:"flavorsReservation"`

	// flavorsUsage 是按 flavor 分组的已使用配额，当前由分配给此 LocalQueue 的工作负载使用。
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=16
	// +optional
	FlavorUsage []LocalQueueFlavorUsage `json:"flavorUsage"`

	// flavors 列出指定 ClusterQueue 中当前可用的所有 ResourceFlavors。
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=16
	// +optional
	Flavors []LocalQueueFlavorStatus `json:"flavors,omitempty"`

	// FairSharing 包含有关公平共享当前状态的信息。
	// +optional
	FairSharing *FairSharingStatus `json:"fairSharing,omitempty"`
}

const (
	// LocalQueueActive 表示支持 LocalQueue 的 ClusterQueue 处于活跃状态，
	// LocalQueue 可以向其 ClusterQueue 提交新的工作负载。
	LocalQueueActive string = "Active"
)

type LocalQueueFlavorUsage struct {
	// flavor 的名称。
	Name ResourceFlavorReference `json:"name"`

	// resources 列出此 flavor 中资源的配额使用情况。
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=16
	Resources []LocalQueueResourceUsage `json:"resources"`
}

type LocalQueueResourceUsage struct {
	// 资源的名称。
	Name corev1.ResourceName `json:"name"`

	// total 是已使用配额的总量。
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

// LocalQueue 是 localQueues API 的 Schema
type LocalQueue struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LocalQueueSpec   `json:"spec,omitempty"`
	Status LocalQueueStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LocalQueueList 包含 LocalQueue 的列表
type LocalQueueList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LocalQueue `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LocalQueue{}, &LocalQueueList{})
}
