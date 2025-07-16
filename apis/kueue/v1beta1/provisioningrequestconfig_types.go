package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ProvisioningRequestConfigPodSetMergePolicy string

const (
	// ProvisioningRequestControllerName 是 Provisioning Request 准入检查控制器使用的名称。
	ProvisioningRequestControllerName = "kueue.x-k8s.io/provisioning-request"

	IdenticalWorkloadSchedulingRequirements ProvisioningRequestConfigPodSetMergePolicy = "IdenticalWorkloadSchedulingRequirements"
	IdenticalPodTemplates                   ProvisioningRequestConfigPodSetMergePolicy = "IdenticalPodTemplates"
)

// ProvisioningRequestConfigSpec 定义了 ProvisioningRequestConfig 的期望状态
type ProvisioningRequestConfigSpec struct {
	// ProvisioningClassName 描述了资源调度的不同模式。
	// 详情请参考 autoscaling.x-k8s.io ProvisioningRequestSpec.ProvisioningClassName。
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +kubebuilder:validation:MaxLength=253
	ProvisioningClassName string `json:"provisioningClassName"`

	// Parameters 包含类可能需要的所有其他参数。
	//
	// +optional
	// +kubebuilder:validation:MaxProperties=100
	Parameters map[string]Parameter `json:"parameters,omitempty"`

	// managedResources 包含由自动扩缩容管理的资源列表。
	//
	// 如果为空，则认为所有资源都被管理。
	//
	// 如果不为空，ProvisioningRequest 只会包含请求了其中至少一个资源的 podset。
	//
	// 如果所有 workload 的 podset 都没有请求任何被管理的资源，则认为该 workload 已就绪。
	//
	// +optional
	// +listType=set
	// +kubebuilder:validation:MaxItems=100
	ManagedResources []corev1.ResourceName `json:"managedResources,omitempty"`

	// retryStrategy 定义了重试 ProvisioningRequest 的策略。
	// 如果为 null，则应用默认配置，参数如下：
	// backoffLimitCount:  3
	// backoffBaseSeconds: 60 - 1 分钟
	// backoffMaxSeconds:  1800 - 30 分钟
	//
	// 若要关闭重试机制，将 retryStrategy.backoffLimitCount 设为 0。
	//
	// +optional
	// +kubebuilder:default={backoffLimitCount:3,backoffBaseSeconds:60,backoffMaxSeconds:1800}
	RetryStrategy *ProvisioningRequestRetryStrategy `json:"retryStrategy,omitempty"`

	// podSetUpdates 指定 workload 的 PodSetUpdates 更新，用于定位已调度节点。
	//
	// +optional
	PodSetUpdates *ProvisioningRequestPodSetUpdates `json:"podSetUpdates,omitempty"`

	// podSetMergePolicy 指定在传递给集群自动扩缩容器前合并 PodSet 的策略。
	//
	// +optional
	// +kubebuilder:validation:Enum=IdenticalPodTemplates;IdenticalWorkloadSchedulingRequirements
	PodSetMergePolicy *ProvisioningRequestConfigPodSetMergePolicy `json:"podSetMergePolicy,omitempty"`
}

type ProvisioningRequestPodSetUpdates struct {
	// nodeSelector 指定 NodeSelector 的更新列表。
	//
	// +optional
	// +kubebuilder:validation:MaxItems=8
	NodeSelector []ProvisioningRequestPodSetUpdatesNodeSelector `json:"nodeSelector,omitempty"`
}

type ProvisioningRequestPodSetUpdatesNodeSelector struct {
	// key 指定 NodeSelector 的键。
	//
	// +required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=317
	// +kubebuilder:validation:Pattern=`^([a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*/)?(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])$`
	Key string `json:"key"`

	// valueFromProvisioningClassDetail 指定 ProvisioningRequest.status.provisioningClassDetails 的键，
	// 其值用于更新。
	//
	// +required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=32768
	ValueFromProvisioningClassDetail string `json:"valueFromProvisioningClassDetail"`
}

type ProvisioningRequestRetryStrategy struct {
	// BackoffLimitCount 定义了最大重试次数。
	// 达到该次数后，workload 会被停用（`.spec.activate`=`false`）。
	//
	// 每次退避时间大约为 "b*2^(n-1)+Rand"，其中：
	// - "b" 由 "BackoffBaseSeconds" 参数设定，
	// - "n" 为 "workloadStatus.requeueState.count"，
	// - "Rand" 为随机抖动。
	// 在此期间，workload 被视为不可接受，其他 workload 有机会被调度。
	// 默认连续重排队延迟约为：(60s, 120s, 240s, ...)。
	//
	// 默认值为 3。
	// +optional
	// +kubebuilder:default=3
	BackoffLimitCount *int32 `json:"backoffLimitCount,omitempty"`

	// BackoffBaseSeconds 定义了被驱逐 workload 重新排队的指数退避基数。
	//
	// 默认值为 60。
	// +optional
	// +kubebuilder:default=60
	BackoffBaseSeconds *int32 `json:"backoffBaseSeconds,omitempty"`

	// BackoffMaxSeconds 定义了被驱逐 workload 重新排队的最大退避时间。
	//
	// 默认值为 1800。
	// +optional
	// +kubebuilder:default=1800
	BackoffMaxSeconds *int32 `json:"backoffMaxSeconds,omitempty"`
}

// Parameter 限制为 255 个字符。
// +kubebuilder:validation:MaxLength=255
type Parameter string

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Cluster

// ProvisioningRequestConfig 是 provisioningrequestconfig API 的 Schema
type ProvisioningRequestConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ProvisioningRequestConfigSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// ProvisioningRequestConfigList 包含 ProvisioningRequestConfig 的列表
type ProvisioningRequestConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProvisioningRequestConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProvisioningRequestConfig{}, &ProvisioningRequestConfigList{})
}
