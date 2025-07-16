package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Cluster,shortName={flavor,flavors,rf}

// ResourceFlavor is the Schema for the resourceflavors API.
// ResourceFlavor 是 resourceflavors API 的架构。
type ResourceFlavor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ResourceFlavorSpec `json:"spec,omitempty"`
}

// TopologyReference is the name of the Topology.
// TopologyReference 是拓扑的名称。
// +kubebuilder:validation:MaxLength=253
// +kubebuilder:validation:Pattern="^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$"
type TopologyReference string

// ResourceFlavorSpec defines the desired state of the ResourceFlavor
// ResourceFlavorSpec 定义了 ResourceFlavor 的期望状态
// +kubebuilder:validation:XValidation:rule="!has(self.topologyName) || self.nodeLabels.size() >= 1", message="at least one nodeLabel is required when topology is set"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.topologyName) || self == oldSelf", message="resourceFlavorSpec are immutable when topologyName is set"
type ResourceFlavorSpec struct {
	// nodeLabels are labels that associate the ResourceFlavor with Nodes that
	// have the same labels.
	// 当 Workload 被接纳时，其 podsets 只能分配到 nodeLabels 匹配 nodeSelector 和 nodeAffinity 字段的 ResourceFlavors。
	// 一旦 ResourceFlavor 被分配给 podSet，集成 Workload 对象的控制器应将 ResourceFlavor 的 nodeLabels 注入到 Workload 的 pods 中。
	//
	// nodeLabels 最多可以有 8 个元素。
	// +optional
	// +mapType=atomic
	// +kubebuilder:validation:MaxProperties=8
	NodeLabels map[string]string `json:"nodeLabels,omitempty"`

	// nodeTaints are taints that the nodes associated with this ResourceFlavor
	// have.
	// Workloads 的 podsets 必须对这些 nodeTaints 有容忍（tolerations），才能在接纳时分配到该 ResourceFlavor。
	// 当此 ResourceFlavor 也设置了匹配的容忍（在 .spec.tolerations 中），则在接纳时不会考虑 nodeTaints。
	// 只评估 'NoSchedule' 和 'NoExecute' 污点效果，忽略 'PreferNoSchedule'。
	//
	// nodeTaints 示例：
	// cloud.provider.com/preemptible="true":NoSchedule
	//
	// nodeTaints 最多可以有 8 个元素。
	//
	// +optional
	// +listType=atomic
	// +kubebuilder:validation:MaxItems=8
	// +kubebuilder:validation:XValidation:rule="self.all(x, x.effect in ['NoSchedule', 'PreferNoSchedule', 'NoExecute'])", message="supported taint effect values: 'NoSchedule', 'PreferNoSchedule', 'NoExecute'"
	NodeTaints []corev1.Taint `json:"nodeTaints,omitempty"`

	// tolerations are extra tolerations that will be added to the pods admitted in
	// the quota associated with this resource flavor.
	//
	// 容忍（toleration）示例：
	// cloud.provider.com/preemptible="true":NoSchedule
	//
	// tolerations 最多可以有 8 个元素。
	//
	// +optional
	// +listType=atomic
	// +kubebuilder:validation:MaxItems=8
	// +kubebuilder:validation:XValidation:rule="self.all(x, !has(x.key) ? x.operator == 'Exists' : true)", message="operator must be Exists when 'key' is empty, which means 'match all values and all keys'"
	// +kubebuilder:validation:XValidation:rule="self.all(x, has(x.tolerationSeconds) ? x.effect == 'NoExecute' : true)", message="effect must be 'NoExecute' when 'tolerationSeconds' is set"
	// +kubebuilder:validation:XValidation:rule="self.all(x, !has(x.operator) || x.operator in ['Equal', 'Exists'])", message="supported toleration values: 'Equal'(default), 'Exists'"
	// +kubebuilder:validation:XValidation:rule="self.all(x, has(x.operator) && x.operator == 'Exists' ? !has(x.value) : true)", message="a value must be empty when 'operator' is 'Exists'"
	// +kubebuilder:validation:XValidation:rule="self.all(x, !has(x.effect) || x.effect in ['NoSchedule', 'PreferNoSchedule', 'NoExecute'])", message="supported taint effect values: 'NoSchedule', 'PreferNoSchedule', 'NoExecute'"
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// topologyName indicates topology for the TAS ResourceFlavor.
	// topologyName 表示 TAS ResourceFlavor 的拓扑。
	// 指定后，会从与 Resource Flavor nodeLabels 匹配的节点中抓取拓扑信息。
	//
	// +optional
	TopologyName *TopologyReference `json:"topologyName,omitempty"`
}

// +kubebuilder:object:root=true

// ResourceFlavorList contains a list of ResourceFlavor
// ResourceFlavorList 包含 ResourceFlavor 的列表
type ResourceFlavorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourceFlavor `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ResourceFlavor{}, &ResourceFlavorList{})
}
