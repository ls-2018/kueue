package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Value",JSONPath=".value",type=integer,description="Value of workloadPriorityClass's Priority"

// WorkloadPriorityClass is the Schema for the workloadPriorityClass API
// WorkloadPriorityClass 是 workloadPriorityClass API 的模式定义
type WorkloadPriorityClass struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// value represents the integer value of this workloadPriorityClass. This is the actual priority that workloads
	// receive when jobs have the name of this class in their workloadPriorityClass label.
	// Changing the value of workloadPriorityClass doesn't affect the priority of workloads that were already created.
	// value 表示此 workloadPriorityClass 的整数值。这是当作业在其 workloadPriorityClass 标签中具有此类名称时，工作负载实际获得的优先级。
	// 更改 workloadPriorityClass 的 value 不会影响已创建工作负载的优先级。
	Value int32 `json:"value"`

	// description is an arbitrary string that usually provides guidelines on
	// when this workloadPriorityClass should be used.
	// +optional
	// description 是一个任意字符串，通常用于提供何时应使用此 workloadPriorityClass 的指导。
	Description string `json:"description,omitempty"`
}

// +kubebuilder:object:root=true

// WorkloadPriorityClassList contains a list of WorkloadPriorityClass
// WorkloadPriorityClassList 包含 WorkloadPriorityClass 的列表
type WorkloadPriorityClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkloadPriorityClass `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WorkloadPriorityClass{}, &WorkloadPriorityClassList{})
}
