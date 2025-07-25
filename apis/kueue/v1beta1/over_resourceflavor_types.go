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

// ResourceFlavor 是 resourceflavors API 的 Schema。
type ResourceFlavor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ResourceFlavorSpec `json:"spec,omitempty"`
}

// TopologyReference 是拓扑的名称。
// +kubebuilder:validation:MaxLength=253
// +kubebuilder:validation:Pattern="^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$"
type TopologyReference string

// ResourceFlavorSpec 定义 ResourceFlavor 的期望状态
// +kubebuilder:validation:XValidation:rule="!has(self.topologyName) || self.nodeLabels.size() >= 1", message="at least one nodeLabel is required when topology is set"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.topologyName) || self == oldSelf", message="resourceFlavorSpec are immutable when topologyName is set"
type ResourceFlavorSpec struct {
	// nodeLabels 是将 ResourceFlavor 与具有相同标签的节点关联的标签。
	// 当工作负载被接纳时，其 podset 只能分配给 nodeLabels 与 nodeSelector 和 nodeAffinity 字段匹配的 ResourceFlavors。
	// 一旦 ResourceFlavor 被分配给 podSet，ResourceFlavor 的 nodeLabels 应该由与工作负载对象集成的控制器注入到工作负载的 pod 中。
	//
	// nodeLabels 最多可以有 8 个元素。
	// +optional
	// +mapType=atomic
	// +kubebuilder:validation:MaxProperties=8
	NodeLabels map[string]string `json:"nodeLabels,omitempty"`

	// nodeTaints 是与此 ResourceFlavor 关联的节点所具有的污点。
	// 工作负载的 podset 必须对这些 nodeTaints 有容忍度，才能在准入期间被分配此 ResourceFlavor。
	// 当此 ResourceFlavor 也设置了匹配的容忍度（在 .spec.tolerations 中）时，则在准入期间不考虑 nodeTaints。
	// 仅评估 'NoSchedule' 和 'NoExecute' 污点效果，而忽略 'PreferNoSchedule'。
	//
	// nodeTaint 的示例是
	// cloud.provider.com/preemptible="true":NoSchedule
	//
	// nodeTaints 最多可以有 8 个元素。
	//
	// +optional
	// +listType=atomic
	// +kubebuilder:validation:MaxItems=8
	// +kubebuilder:validation:XValidation:rule="self.all(x, x.effect in ['NoSchedule', 'PreferNoSchedule', 'NoExecute'])", message="supported taint effect values: 'NoSchedule', 'PreferNoSchedule', 'NoExecute'"
	NodeTaints []corev1.Taint `json:"nodeTaints,omitempty"`

	// tolerations 是额外的容忍度，将添加到与此资源类型关联的配额中接纳的 pod 中。
	//
	// 容忍度的示例是
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

	// topologyName 指定了 TAS 资源类型所对应的拓扑结构。
	// 若指定了该参数，则能够从与资源类型标签相匹配的节点中抓取拓扑信息。
	// +optional
	TopologyName *TopologyReference `json:"topologyName,omitempty"`
}

// +kubebuilder:object:root=true

// ResourceFlavorList 包含 ResourceFlavor 的列表
type ResourceFlavorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourceFlavor `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ResourceFlavor{}, &ResourceFlavorList{})
}
