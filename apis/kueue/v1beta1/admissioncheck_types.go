package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CheckState string

const (
	// CheckStateRetry 表示该检查当前无法通过，需回退（可能允许其他尝试，解除配额阻塞）并重试。
	// 如果某个工作负载至少有一个检查处于此状态，则在被接收后会被驱逐，并且在检查处于此状态时不会被考虑接收。
	CheckStateRetry CheckState = "Retry"

	// CheckStateRejected 表示该检查在近期内不会通过，不值得重试。
	// 如果某个工作负载至少有一个检查处于此状态，则在被接收后会被驱逐并被停用。
	CheckStateRejected CheckState = "Rejected"

	// CheckStatePending 表示该检查尚未执行，状态可能为：
	// 1. Unknown，该条件由 kueue 添加，其控制器无法评估。
	// 2. 由其控制器设置，并在配额保留后重新评估。
	CheckStatePending CheckState = "Pending"

	// CheckStateReady 表示该检查已通过。
	// 如果所有检查都为 ready 且配额已保留，工作负载即可开始执行。
	CheckStateReady CheckState = "Ready"
)

// AdmissionCheckSpec 定义 AdmissionCheck 的期望状态
type AdmissionCheckSpec struct {
	// controllerName 标识处理 AdmissionCheck 的控制器，不一定是 Kubernetes Pod 或 Deployment 名称。不能为空。
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf", message="field is immutable"
	ControllerName string `json:"controllerName"`

	// RetryDelayMinutes 指定检查失败（转为 False）后，工作负载保持挂起的时间。延迟期过后，检查状态变为 "Unknown"。默认 15 分钟。
	// +optional
	// +kubebuilder:default=15
	// 已废弃：retryDelayMinutes 自 v0.8 起已废弃，将在 v1beta2 移除。
	RetryDelayMinutes *int64 `json:"retryDelayMinutes,omitempty"`

	// Parameters 标识带有附加参数的检查配置。
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

// AdmissionCheckStatus 定义 AdmissionCheck 的观测状态
type AdmissionCheckStatus struct {
	// conditions 保存 AdmissionCheck 当前状态的最新可用观测信息。
	// +optional
	// +listType=map
	// +listMapKey=type
	// +patchStrategy=merge
	// +patchMergeKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

const (
	// AdmissionCheckActive 表示 admission check 的控制器已准备好评估检查状态
	AdmissionCheckActive string = "Active"
)

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

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
