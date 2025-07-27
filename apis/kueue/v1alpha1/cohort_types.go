package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kueuebeta "sigs.k8s.io/kueue/apis/kueue/v1beta1"
)

// CohortSpec 定义了 Cohort 的期望状态
type CohortSpec struct {
	// Parent 引用 Cohort 父级的名称（如果有）。有三种情况：
	// 1）未设置。本 Cohort 是其 Cohort 树的根。
	// 2）引用了不存在的 Cohort。此时使用默认 Cohort（无借用/出借限制）。
	// 3）引用了已存在的 Cohort。
	//
	// 如果形成了环，则会禁用该 Cohort 的所有成员（包括 ClusterQueue），直到环被移除。在环存在期间，禁止进一步接纳。
	Parent kueuebeta.CohortReference `json:"parent,omitempty"`

	// ResourceGroups 描述了资源和风味的分组。每个 ResourceGroup 定义了一组资源和一组为这些资源提供配额的风味。每个资源和风味只能属于一个 ResourceGroup。每个 Cohort 最多可有 16 个 ResourceGroup。
	//
	// BorrowingLimit 限制该 Cohort 子树成员可从父子树借用的资源量。
	//
	// LendingLimit 限制该 Cohort 子树成员可向父子树出借的资源量。
	//
	// 借用和出借限制仅在 Cohort 有父级时设置，否则 Cohort 的创建/更新会被 webhook 拒绝。
	//
	//+listType=atomic
	//+kubebuilder:validation:MaxItems=16
	ResourceGroups []kueuebeta.ResourceGroup `json:"resourceGroups,omitempty"`

	// fairSharing 定义了 Cohort 参与公平共享时的属性。仅当 Kueue 配置中启用公平共享时这些值才有效。
	// +optional
	FairSharing *kueuebeta.FairSharing `json:"fairSharing,omitempty"`
}

// CohortStatus 定义了 Cohort 的观测状态。
type CohortStatus struct {
	// fairSharing 包含该 Cohort 参与公平共享时的当前状态。
	// 仅在 Kueue 配置中启用公平共享时记录。
	// +optional
	FairSharing *kueuebeta.FairSharingStatus `json:"fairSharing,omitempty"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:subresource:status

// Cohort 定义了 Cohorts API。
//
// 分层 Cohort（有父级的 Cohort）自 v0.11 起兼容公平共享。在 v0.9 和 v0.10 版本中同时使用这些特性不受支持，可能导致未定义行为。
type Cohort struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CohortSpec   `json:"spec,omitempty"`
	Status CohortStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CohortList 包含 Cohort 的列表
type CohortList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cohort `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cohort{}, &CohortList{})
}
