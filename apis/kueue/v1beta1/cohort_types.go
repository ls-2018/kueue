package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CohortSpec 定义了 Cohort 的期望状态
type CohortSpec struct {
	// ParentName 引用该 Cohort 父级的名称（如果有）。它满足以下三种情况之一：
	// 1) 未设置。本 Cohort 是其 Cohort 树的根。
	// 2) 引用了一个不存在的 Cohort。我们使用默认 Cohort（无借用/出借限制）。
	// 3) 引用了一个存在的 Cohort。
	//
	// 如果创建了循环，我们会禁用该 Cohort 的所有成员，包括 ClusterQueues，直到循环被移除。在循环存在期间，阻止进一步的准入。
	ParentName CohortReference `json:"parentName,omitempty"`

	// ResourceGroups 描述了资源和风味的分组。每个 ResourceGroup 定义了一组资源和一组为这些资源提供配额的风味。每个资源和每个风味只能属于一个 ResourceGroup。一个 Cohort 最多可以有 16 个 ResourceGroup。
	//
	// BorrowingLimit 限制该 Cohort 子树成员可以从父子树借用的资源量。
	//
	// LendingLimit 限制该 Cohort 子树成员可以借给父子树的资源量。
	//
	// 只有当 Cohort 有父级时，才能设置借用和出借限制。否则，webhook 会拒绝 Cohort 的创建/更新。
	//
	//+listType=atomic
	//+kubebuilder:validation:MaxItems=16
	ResourceGroups []ResourceGroup `json:"resourceGroups,omitempty"`

	// fairSharing 定义了 Cohort 参与公平共享时的属性。仅当 Kueue 配置中启用公平共享时，这些值才有意义。
	// +optional
	FairSharing *FairSharing `json:"fairSharing,omitempty"`
}

// CohortStatus 定义了 Cohort 的观测状态。
type CohortStatus struct {
	// fairSharing 包含该 Cohort 参与公平共享时的当前状态。
	// 仅当 Kueue 配置中启用公平共享时才会记录。
	// +optional
	FairSharing *FairSharingStatus `json:"fairSharing,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status

// Cohort 定义了 Cohorts API。
//
// 分层 Cohort（有父级的 Cohort）自 v0.11 起兼容公平共享。在 v0.9 和 v0.10 中同时使用这些特性是不支持的，会导致未定义行为。
type Cohort struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CohortSpec   `json:"spec,omitempty"`
	Status CohortStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CohortList 包含 Cohort 的列表
type CohortList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cohort `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cohort{}, &CohortList{})
}
