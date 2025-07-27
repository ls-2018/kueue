package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// PodSetRequiredTopologyAnnotation 表示 PodSet 需要拓扑感知调度，并要求所有 pod 调度到注解值所指示的拓扑级别（例如机架或区块）内的同一拓扑域的节点上。
	PodSetRequiredTopologyAnnotation = "kueue.x-k8s.io/podset-required-topology"

	// PodSetPreferredTopologyAnnotation 表示 PodSet 需要拓扑感知调度，但所有 pod 调度到同一拓扑域的节点只是偏好而非强制要求。
	//
	// 这些级别会从注解所指示的级别向上逐级评估。如果 PodSet 无法适应某个拓扑域，则会考虑下一个更高的拓扑级别。如果在最高拓扑级别仍无法适应，则会分布在多个拓扑域中。
	PodSetPreferredTopologyAnnotation = "kueue.x-k8s.io/podset-preferred-topology"

	// PodSetUnconstrainedTopologyAnnotation 表示 PodSet 没有任何拓扑要求。
	// 如果有足够的空闲容量，Kueue 会接纳该 PodSet。
	// 推荐用于不需要低延迟或高吞吐量 pod 间通信，但希望利用 TAS 能力提升作业接纳准确性的 PodSet。
	//
	// +kubebuilder:validation:Type=boolean
	PodSetUnconstrainedTopologyAnnotation = "kueue.x-k8s.io/podset-unconstrained-topology"

	// TopologySchedulingGate 用于延迟 Pod 的调度，直到分配的拓扑域对应的 nodeSelector 注入到 Pod 中。对于基于 Pod 的集成，gate 会在 Pod 创建时通过 webhook 添加。
	TopologySchedulingGate = "kueue.x-k8s.io/topology"

	// WorkloadAnnotation 是设置在 Job 的 PodTemplate 上的注解，用于指示与 Job 对应的已接纳 Workload 的名称。该注解在启动 Job 时设置，在停止 Job 时移除。
	WorkloadAnnotation = "kueue.x-k8s.io/workload"

	// TASLabel 是设置在 Job 的 PodTemplate 上的标签，表示 PodSet 是通过拓扑感知调度接纳的，且所有由该 PodTemplate 创建的 Pod 也有该标签。对于基于 Pod 的集成，标签会在 Pod 创建时通过 webhook 添加。
	TASLabel = "kueue.x-k8s.io/tas"

	// PodGroupPodIndexLabel 是设置在属于 Pod 组的 Pod 元数据上的标签，表示该 Pod 在组内的索引。
	PodGroupPodIndexLabel = "kueue.x-k8s.io/pod-group-pod-index"

	// PodGroupPodIndexLabelAnnotation 是设置在属于 Pod 组的 Pod 元数据上的注解，表示用于获取该 Pod 在组内索引的标签名。
	PodGroupPodIndexLabelAnnotation = "kueue.x-k8s.io/pod-group-pod-index-label"

	// NodeToReplaceAnnotation 是设置在 Workload 上的注解，记录运行该 workload 至少一个 pod 的故障节点名称。
	NodeToReplaceAnnotation = "alpha.kueue.x-k8s.io/node-to-replace"
)

// TopologySpec 定义了 Topology 的期望状态
type TopologySpec struct {
	// levels 定义了拓扑的各级。
	//
	// +required
	// +listType=atomic
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=8
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="field is immutable"
	// +kubebuilder:validation:XValidation:rule="size(self.filter(i, size(self.filter(j, j == i)) > 1)) == 0",message="must be unique"
	// +kubebuilder:validation:XValidation:rule="size(self.filter(i, i.nodeLabel == 'kubernetes.io/hostname')) == 0 || self[size(self) - 1].nodeLabel == 'kubernetes.io/hostname'",message="the kubernetes.io/hostname label can only be used at the lowest level of topology"
	Levels []TopologyLevel `json:"levels,omitempty"`
}

// TopologyLevel 定义了 TopologyLevel 的期望状态
type TopologyLevel struct {
	// “nodeLabel”表示特定拓扑级别下节点标签的名称。
	//
	// 示例：
	// - cloud.provider.com/topology-block
	// - cloud.provider.com/topology-rack
	//
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=316
	// +kubebuilder:validation:Pattern=`^([a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*/)?(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])$`
	NodeLabel string `json:"nodeLabel"`
}

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Cluster

// Topology 是 topology API 的 Schema
type Topology struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:validation:Required
	Spec TopologySpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// TopologyList 包含 Topology 的列表
type TopologyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Topology `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Topology{}, &TopologyList{})
}
