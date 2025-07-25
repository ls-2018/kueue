package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ProvisioningRequestConfigPodSetMergePolicy string

const (
	// ProvisioningRequestControllerName 是 Provisioning 请求准入检查控制器使用的名称。
	ProvisioningRequestControllerName                                                  = "kueue.x-k8s.io/provisioning-request"
	IdenticalWorkloadSchedulingRequirements ProvisioningRequestConfigPodSetMergePolicy = "IdenticalWorkloadSchedulingRequirements"
	IdenticalPodTemplates                   ProvisioningRequestConfigPodSetMergePolicy = "IdenticalPodTemplates"
)

// ProvisioningRequestConfigSpec 定义 ProvisioningRequestConfig 的期望状态。
type ProvisioningRequestConfigSpec struct {
	// ProvisioningClassName 描述资源预配的不同模式。
	// 请参阅 autoscaling.x-k8s.io ProvisioningRequestSpec.ProvisioningClassName 了解更多详情。
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +kubebuilder:validation:MaxLength=253
	ProvisioningClassName string `json:"provisioningClassName"`

	// Parameters 包含其他参数类可能需要的所有参数。
	//
	// +optional
	// +kubebuilder:validation:MaxProperties=100
	Parameters map[string]Parameter `json:"parameters,omitempty"`

	// managedResources 包含由自动扩缩控制器管理的资源列表。
	// 如果为空，则认为所有资源都受管理。
	// 如果非空，则 ProvisioningRequest 仅包含请求至少一个受管理资源的 podset。
	// 如果工作负载的 podset 都没有请求受管理资源，则认为工作负载已就绪。
	//
	// +optional
	// +listType=set
	// +kubebuilder:validation:MaxItems=100
	ManagedResources []corev1.ResourceName `json:"managedResources,omitempty"`

	// retryStrategy 定义重试 ProvisioningRequest 的策略。
	// 如果为 null，则应用默认配置，并使用以下参数值：
	// backoffLimitCount:  3
	// backoffBaseSeconds: 60 - 1 min
	// backoffMaxSeconds:  1800 - 30 mins
	//
	// 要关闭重试机制
	// 将 retryStrategy.backoffLimitCount 设置为 0。
	//
	// +optional
	// +kubebuilder:default={backoffLimitCount:3,backoffBaseSeconds:60,backoffMaxSeconds:1800}
	RetryStrategy *ProvisioningRequestRetryStrategy `json:"retryStrategy,omitempty"`

	// podSetUpdates 指定工作负载的 PodSetUpdates 更新，
	// 这些更新用于定位预配的节点。
	//
	// +optional
	PodSetUpdates *ProvisioningRequestPodSetUpdates `json:"podSetUpdates,omitempty"`

	// podSetMergePolicy 指定合并 PodSets 的策略，以便传递给集群自动扩缩器。
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

	// valueFromProvisioningClassDetail 指定用于更新值的
	// ProvisioningRequest.status.provisioningClassDetails 的键。
	//
	// +required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=32768
	ValueFromProvisioningClassDetail string `json:"valueFromProvisioningClassDetail"`
}

type ProvisioningRequestRetryStrategy struct {
	// BackoffLimitCount 定义最大重试次数。
	// 达到此数量后，工作负载将被停用（`.spec.activate`=`false`）。
	//
	// 每次回退持续时间大约为 "b*2^(n-1)+Rand"，其中：
	// - "b" 表示由 "BackoffBaseSeconds" 参数设置的基础，
	// - "n" 表示 "workloadStatus.requeueState.count"，
	// - "Rand" 表示随机抖动。
	// 在此期间，工作负载被视为不可接受，
	// 其他工作负载将有机会被接纳。
	// 默认情况下，连续请求延迟约为：(60s, 120s, 240s, ...)。
	//
	// 默认值为 3。
	// +optional
	// +kubebuilder:default=3
	BackoffLimitCount *int32 `json:"backoffLimitCount,omitempty"`

	// BackoffBaseSeconds 定义工作负载驱逐后重试的指数回退基础。
	//
	// 默认值为 60。
	// +optional
	// +kubebuilder:default=60
	BackoffBaseSeconds *int32 `json:"backoffBaseSeconds,omitempty"`

	// BackoffMaxSeconds 定义驱逐工作负载的最大回退时间。
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

// ProvisioningRequestConfig 是 ProvisioningRequestConfig API 的 Schema。
type ProvisioningRequestConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ProvisioningRequestConfigSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// ProvisioningRequestConfigList 包含 ProvisioningRequestConfig 的列表。
type ProvisioningRequestConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProvisioningRequestConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProvisioningRequestConfig{}, &ProvisioningRequestConfigList{})
}
