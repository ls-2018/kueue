package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	MultiKueueConfigSecretKey = "kubeconfig"
	MultiKueueClusterActive   = "Active"

	// MultiKueueOriginLabel 是用于追踪 multikueue 远程对象创建者的标签。
	MultiKueueOriginLabel = "kueue.x-k8s.io/multikueue-origin"

	// MultiKueueControllerName 是 MultiKueue 准入检查控制器使用的名称。
	MultiKueueControllerName = "kueue.x-k8s.io/multikueue"
)

type LocationType string

const (
	// PathLocationType 表示 kueue-controller-manager 磁盘上的路径。
	PathLocationType LocationType = "Path"

	// SecretLocationType 表示 kueue 控制器管理器所在命名空间中的 Secret 名称。配置应存储在 "kubeconfig" 键中。
	SecretLocationType LocationType = "Secret"
)

type KubeConfig struct {
	// KubeConfig 的位置。
	//
	// 如果 LocationType 为 Secret，则 Location 是 kueue 控制器管理器所在命名空间中的 Secret 名称。配置应存储在 "kubeconfig" 键中。
	Location string `json:"location"`

	// KubeConfig 位置的类型。
	//
	// +kubebuilder:default=Secret
	// +kubebuilder:validation:Enum=Secret;Path
	LocationType LocationType `json:"locationType"`
}

type MultiKueueClusterSpec struct {
	// 连接集群的信息。
	KubeConfig KubeConfig `json:"kubeConfig"`
}

type MultiKueueClusterStatus struct {
	// +optional
	// +listType=map
	// +listMapKey=type
	// +patchStrategy=merge
	// +patchMergeKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// +kubebuilder:printcolumn:name="Connected",JSONPath=".status.conditions[?(@.type=='Active')].status",type="string",description="MultiKueueCluster is connected"
// +kubebuilder:printcolumn:name="Age",JSONPath=".metadata.creationTimestamp",type="date",description="Time this workload was created"
// MultiKueueCluster 是 multikueue API 的 Schema。
type MultiKueueCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MultiKueueClusterSpec   `json:"spec,omitempty"`
	Status MultiKueueClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MultiKueueClusterList 包含 MultiKueueCluster 的列表。
type MultiKueueClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MultiKueueCluster `json:"items"`
}

// MultiKueueConfigSpec 定义了 MultiKueueConfig 的期望状态。
type MultiKueueConfigSpec struct {
	// ClusterQueue 中的工作负载应分发到的 MultiKueueCluster 名称列表。
	//
	// +listType=set
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=10
	Clusters []string `json:"clusters"`
}

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Cluster

// MultiKueueConfig 是 multikueue API 的 Schema。
type MultiKueueConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec MultiKueueConfigSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// MultiKueueConfigList 包含 MultiKueueConfig 的列表。
type MultiKueueConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MultiKueueConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MultiKueueConfig{}, &MultiKueueConfigList{}, &MultiKueueCluster{}, &MultiKueueClusterList{})
}
