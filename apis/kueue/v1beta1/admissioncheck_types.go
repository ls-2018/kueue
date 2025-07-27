package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CheckState string

const (
	// CheckStateRetry 表示此时检查无法通过，需退避（可能允许其他尝试，解除配额阻塞）并重试。
	// 如果工作负载中至少有一个检查处于此状态，若已被接纳则会被驱逐，并且在该检查处于此状态时不会被考虑接纳。
	CheckStateRetry CheckState = "Retry"

	// CheckStateRejected 如果工作负载中至少有一个检查处于此状态，若已被接纳则会被驱逐并失效。
	CheckStateRejected CheckState = "Rejected"

	// CheckStatePending 1. Unknown，条件由 kueue 添加，控制器无法评估。
	// 2. 由控制器设置，并在配额预留后重新评估。
	CheckStatePending CheckState = "Pending"

	// CheckStateReady 表示检查已通过。
	// 如果工作负载的所有检查都为 ready，且已预留配额，则可开始执行。
	CheckStateReady CheckState = "Ready"
)

// AdmissionCheckSpec 定义了 AdmissionCheck 的期望状态
type AdmissionCheckSpec struct {
	// controllerName 标识处理 AdmissionCheck 的控制器，不一定是 Kubernetes 的 Pod 或 Deployment 名称。不能为空。
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf", message="field is immutable"
	ControllerName string `json:"controllerName"`

	// Parameters 标识检查的附加参数配置。
	// +optional
	Parameters *AdmissionCheckParametersReference `json:"parameters,omitempty"`
}

type AdmissionCheckParametersReference struct {
	// ApiGroup 是被引用资源的组。
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern="^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$"
	APIGroup string `json:"apiGroup"`
	// Kind 是被引用资源的类型。
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern="^(?i)[a-z]([-a-z0-9]*[a-z0-9])?$"
	Kind string `json:"kind"`
	// Name 是被引用资源的名称。
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern="^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"
	Name string `json:"name"`
}

// AdmissionCheckStatus 定义了 AdmissionCheck 的当前状态
type AdmissionCheckStatus struct {
	// conditions 保存 AdmissionCheck 的最新可用状态。
	// current state.
	// +optional
	// +listType=map
	// +listMapKey=type
	// +patchStrategy=merge
	// +patchMergeKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

const (
	// AdmissionCheckActive 表示 admission check 的控制器已准备好评估检查状态。
	AdmissionCheckActive string = "Active"
)

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// AdmissionCheck is the Schema for the admissionchecks API
// AdmissionCheck 是 admissionchecks API 的 Schema
type AdmissionCheck struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AdmissionCheckSpec   `json:"spec,omitempty"`
	Status AdmissionCheckStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AdmissionCheckList 包含 AdmissionCheck 的列表
type AdmissionCheckList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AdmissionCheck `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AdmissionCheck{}, &AdmissionCheckList{})
}
