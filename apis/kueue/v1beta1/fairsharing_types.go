package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FairSharing 包含 ClusterQueue 或 Cohort 参与公平共享时的属性。
//
// 公平共享自 v0.11 起兼容分层 Cohort（任何有父级的 Cohort）。在 V0.9 和 V0.10 中同时使用这些特性是不支持的，会导致未定义行为。
type FairSharing struct {
	// weight 为该 ClusterQueue 或 Cohort 在 Cohort 中争夺未使用资源时提供比较优势。
	// 份额基于每种资源超出名义配额的主导资源使用量，除以权重。
	// 调度优先考虑来自份额最低的 ClusterQueue 和 Cohort 的工作负载，并抢占份额最高的。
	// 权重为零意味着份额值为无穷大，这意味着该节点在与其他 ClusterQueue 和 Cohort 竞争时总是处于劣势。
	// +kubebuilder:default=1
	Weight *resource.Quantity `json:"weight,omitempty"`
}

// FairSharingStatus 包含有关当前公平共享状态的信息。
type FairSharingStatus struct {
	// WeightedShare 表示节点所提供的所有资源中，超出名义配额的使用量与 Cohort 可借用资源的比值的最大值，再除以权重。
	// 如果为零，表示节点的使用量低于名义配额。如果节点权重为零且正在借用，则返回 9223372036854775807，即最大可能的份额值。
	WeightedShare int64 `json:"weightedShare"`

	// admissionFairSharingStatus 表示与 Admission Fair Sharing 相关的信息
	// +optional
	AdmissionFairSharingStatus *AdmissionFairSharingStatus `json:"admissionFairSharingStatus,omitempty"`
}

type AdmissionFairSharingStatus struct {
	// ConsumedResources 表示资源随时间的聚合使用量，应用了衰减函数。
	// 如果在 Kueue 配置中启用了使用量消耗功能，则会填充值。
	// +required
	ConsumedResources corev1.ResourceList `json:"consumedResources"`

	// LastUpdate 是份额和已消耗资源被更新的时间。
	// +required
	LastUpdate metav1.Time `json:"lastUpdate"`
}

type AdmissionScope struct {
	// AdmissionMode 表示 AdmissionScope 中 AdmissionFairSharing 应使用的模式。
	// 可能的值有：
	// - UsageBasedAdmissionFairSharing
	// - NoAdmissionFairSharing
	//
	// +required
	AdmissionMode AdmissionMode `json:"admissionMode"`
}

type AdmissionMode string

const (
	// 基于使用量的 AdmissionFairSharing，QueuingStrategy 由 CQ 定义。
	UsageBasedAdmissionFairSharing AdmissionMode = "UsageBasedAdmissionFairSharing"

	// 对于该 CQ 禁用 AdmissionFairSharing
	NoAdmissionFairSharing AdmissionMode = "NoAdmissionFairSharing"
)
