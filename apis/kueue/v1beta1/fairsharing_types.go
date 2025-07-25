package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FairSharing 包含参与公平共享时 ClusterQueue 或 Cohort 的属性。
//
// 公平共享与分层 Cohort（任何有父级的 Cohort）兼容，自 v0.11 起。
// 在 V0.9 和 V0.10 中一起使用这些功能不受支持，会导致未定义的行为。
type FairSharing struct {
	// weight 在 Cohort 中竞争未使用资源时，为该 ClusterQueue 或 Cohort 提供比较优势。
	// 份额基于每种资源的名义配额之上的主导资源使用量，除以权重。
	// 准入优先调度来自份额最低的 ClusterQueues 和 Cohorts 的工作负载，
	// 并抢占来自份额最高的 ClusterQueues 和 Cohorts 的工作负载。
	// 零权重意味着无限份额值，表示此节点将始终处于相对于其他 ClusterQueues 和 Cohorts 的劣势。
	// +kubebuilder:default=1
	Weight *resource.Quantity `json:"weight,omitempty"`
}

// FairSharingStatus 包含有关公平共享当前状态的信息。
type FairSharingStatus struct {
	// WeightedShare 表示节点提供的所有资源中，使用量超过名义配额与 Cohort 中可出借资源的比率的最大值，
	// 并除以权重。如果为零，表示节点的使用量低于名义配额。
	// 如果节点的权重为零且正在借用，这将返回 9223372036854775807，即最大可能的份额值。
	WeightedShare int64 `json:"weightedShare"`

	// admissionFairSharingStatus 表示与准入公平共享相关的信息
	// +optional
	AdmissionFairSharingStatus *AdmissionFairSharingStatus `json:"admissionFairSharingStatus,omitempty"`
}

type AdmissionFairSharingStatus struct {
	// ConsumedResources 表示随时间聚合的资源使用量，应用了衰减函数。
	// 如果在 Kueue 配置中启用了使用量消耗功能，则会填充此值。
	// +required
	ConsumedResources corev1.ResourceList `json:"consumedResources"`

	// LastUpdate 是份额和已消耗资源更新的时间。
	// +required
	LastUpdate metav1.Time `json:"lastUpdate"`
}

type AdmissionScope struct {
	// AdmissionMode 指示在 AdmissionScope 中应使用哪种准入公平共享模式。可能的值是：
	// - UsageBasedAdmissionFairSharing
	// - NoAdmissionFairSharing
	//
	// +required
	AdmissionMode AdmissionMode `json:"admissionMode"`
}

type AdmissionMode string

const (
	// 基于使用量的准入公平共享，使用 CQ 中定义的排队策略。
	UsageBasedAdmissionFairSharing AdmissionMode = "UsageBasedAdmissionFairSharing"

	// 为此 CQ 禁用准入公平共享
	NoAdmissionFairSharing AdmissionMode = "NoAdmissionFairSharing"
)
