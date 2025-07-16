package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/kueue/apis/kueue/v1beta1"
)

// +genclient
// +kubebuilder:object:root=true
// +k8s:openapi-gen=true
// +genclient:nonNamespaced
// +genclient:method=GetPendingWorkloadsSummary,verb=get,subresource=pendingworkloads,result=sigs.k8s.io/kueue/apis/visibility/v1beta1.PendingWorkloadsSummary
type ClusterQueue struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Summary PendingWorkloadsSummary `json:"pendingWorkloadsSummary"`
}

// +kubebuilder:object:root=true
type ClusterQueueList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ClusterQueue `json:"items"`
}

// +genclient
// +kubebuilder:object:root=true
// +k8s:openapi-gen=true
// +genclient:method=GetPendingWorkloadsSummary,verb=get,subresource=pendingworkloads,result=sigs.k8s.io/kueue/apis/visibility/v1beta1.PendingWorkloadsSummary
type LocalQueue struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Summary PendingWorkloadsSummary `json:"pendingWorkloadsSummary"`
}

// +kubebuilder:object:root=true
type LocalQueueList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []LocalQueue `json:"items"`
}

// PendingWorkload 是面向用户的待处理工作负载表示，用于汇总在集群队列中的相关位置信息。
type PendingWorkload struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Priority 表示工作负载的优先级
	Priority int32 `json:"priority"`

	// LocalQueueName 表示该工作负载提交到的 LocalQueue 的名称
	LocalQueueName v1beta1.LocalQueueName `json:"localQueueName"`

	// PositionInClusterQueue 表示工作负载在 ClusterQueue 中的位置，从 0 开始
	PositionInClusterQueue int32 `json:"positionInClusterQueue"`

	// PositionInLocalQueue 表示工作负载在 LocalQueue 中的位置，从 0 开始
	PositionInLocalQueue int32 `json:"positionInLocalQueue"`
}

// +k8s:openapi-gen=true
// +kubebuilder:object:root=true

// PendingWorkloadsSummary 包含在查询上下文（LocalQueue 或 ClusterQueue 内）中的待处理工作负载列表。
type PendingWorkloadsSummary struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Items []PendingWorkload `json:"items"`
}

// +kubebuilder:object:root=true
// +k8s:openapi-gen=true
// +k8s:conversion-gen:explicit-from=net/url.Values
// +k8s:defaulter-gen=true

// PendingWorkloadOptions 是可见性查询中使用的查询参数
type PendingWorkloadOptions struct {
	metav1.TypeMeta `json:",inline"`

	// Offset 表示应获取的第一个待处理工作负载的位置，从 0 开始。默认值为 0
	Offset int64 `json:"offset"`

	// Limit 表示应获取的待处理工作负载的最大数量。默认值为 1000
	Limit int64 `json:"limit,omitempty"`
}

func init() {
	SchemeBuilder.Register(
		&PendingWorkloadsSummary{},
		&PendingWorkloadOptions{},
	)
}
