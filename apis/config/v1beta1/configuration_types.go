package v1beta1

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	configv1alpha1 "k8s.io/component-base/config/v1alpha1"
)

// +k8s:defaulter-gen=true
// +kubebuilder:object:root=true

// Configuration 是 kueueconfigurations API 的 Schema
type Configuration struct {
	metav1.TypeMeta `json:",inline"`

	// Namespace 是 kueue 部署所在的命名空间。它作为 webhook Service 的 DNSName 的一部分。
	// 如果未设置，则从文件 /var/run/secrets/kubernetes.io/serviceaccount/namespace 获取值。
	// 如果该文件不存在，默认值为 kueue-system。
	Namespace *string `json:"namespace,omitempty"`

	// ControllerManager 返回控制器的配置
	ControllerManager `json:",inline"`

	// ManageJobsWithoutQueueName 控制 Kueue 是否管理未设置注解 kueue.x-k8s.io/queue-name 的作业。
	// 如果设置为 true，则这些作业将被挂起，除非分配了队列并最终被接纳，否则永远不会启动。这也适用于启动 kueue 控制器之前创建的作业。
	// 默认为 false；因此，这些作业不会被管理，如果它们在创建时未挂起，将会立即启动。
	ManageJobsWithoutQueueName bool `json:"manageJobsWithoutQueueName"`

	// ManagedJobsNamespaceSelector 提供基于命名空间的机制，以使作业不被 Kueue 管理。
	//
	// 对于基于 Pod 的集成（pod、deployment、statefulset 等），只有命名空间匹配 ManagedJobsNamespaceSelector 的作业才有资格被 Kueue 管理。
	// 在不匹配的命名空间中的 Pod、deployment 等，即使有 kueue.x-k8s.io/queue-name 标签，也永远不会被 Kueue 管理。
	// 这种强豁免确保 Kueue 不会干扰系统命名空间的基本操作。
	//
	// 对于所有其他集成，ManagedJobsNamespaceSelector 仅通过调节 ManageJobsWithoutQueueName 的效果提供较弱的豁免。
	// 对于这些集成，具有 kueue.x-k8s.io/queue-name 标签的作业始终由 Kueue 管理。只有当 ManageJobsWithoutQueueName 为 true 且作业的命名空间匹配 ManagedJobsNamespaceSelector 时，未设置该标签的作业才会被 Kueue 管理。
	ManagedJobsNamespaceSelector *metav1.LabelSelector `json:"managedJobsNamespaceSelector,omitempty"`

	// InternalCertManagement 是 internalCertManagement 的配置
	InternalCertManagement *InternalCertManagement `json:"internalCertManagement,omitempty"`

	// WaitForPodsReady 是为作业提供基于时间的全有或全无调度语义的配置，通过确保所有 pod 在指定时间内就绪（运行并通过就绪探针）。如果超时，则驱逐该工作负载。
	WaitForPodsReady *WaitForPodsReady `json:"waitForPodsReady,omitempty"`

	// ClientConnection 提供 Kubernetes API server 客户端的其他配置选项。
	ClientConnection *ClientConnection `json:"clientConnection,omitempty"`

	// Integrations 提供 AI/ML/批处理框架集成（包括 K8S job）的配置选项。
	Integrations *Integrations `json:"integrations,omitempty"`

	// QueueVisibility 用于暴露关于队列中最顶层待处理工作负载的信息的配置。
	// 已废弃：该字段将在 v1beta2 移除，请使用 VisibilityOnDemand（https://kueue.sigs.k8s.io/docs/tasks/manage/monitor_pending_workloads/pending_workloads_on_demand/）代替。
	QueueVisibility *QueueVisibility `json:"queueVisibility,omitempty"`

	// MultiKueue 控制 MultiKueue AdmissionCheck Controller 的行为。
	MultiKueue *MultiKueue `json:"multiKueue,omitempty"`

	// FairSharing 控制集群范围内的公平共享语义。
	FairSharing *FairSharing `json:"fairSharing,omitempty"`

	// admissionFairSharing 表示启用 `AdmissionTime` 模式下的 FairSharing 配置
	AdmissionFairSharing *AdmissionFairSharing `json:"admissionFairSharing,omitempty"`

	// Resources 提供处理资源的其他配置选项。
	Resources *Resources `json:"resources,omitempty"`

	// FeatureGates 是特性名称到布尔值的映射，用于覆盖特性的默认启用状态。不能与通过 Kueue Deployment 的命令行参数 "--feature-gates" 传递特性列表同时使用。
	FeatureGates map[string]bool `json:"featureGates,omitempty"`

	// ObjectRetentionPolicies 提供 Kueue 管理对象自动删除的配置选项。为 nil 时禁用所有自动删除。
	// +optional
	ObjectRetentionPolicies *ObjectRetentionPolicies `json:"objectRetentionPolicies,omitempty"`
}

type ControllerManager struct {
	// Webhook 包含控制器 webhook 的配置
	// +optional
	Webhook ControllerWebhook `json:"webhook,omitempty"`

	// LeaderElection 是配置 manager.Manager 选举主节点的 LeaderElection 配置
	// +optional
	LeaderElection *configv1alpha1.LeaderElectionConfiguration `json:"leaderElection,omitempty"`

	// Metrics 包含控制器指标的配置
	// +optional
	Metrics ControllerMetrics `json:"metrics,omitempty"`

	// Health 包含控制器健康检查的配置
	// +optional
	Health ControllerHealth `json:"health,omitempty"`

	// PprofBindAddress 是控制器用于 pprof 服务绑定的 TCP 地址。
	// 可以设置为 "" 或 "0" 以禁用 pprof 服务。
	// 由于 pprof 可能包含敏感信息，公开前请确保保护好。
	// +optional
	PprofBindAddress string `json:"pprofBindAddress,omitempty"`

	// Controller 包含在此管理器中注册的控制器的全局配置选项。
	// +optional
	Controller *ControllerConfigurationSpec `json:"controller,omitempty"`
}

// ControllerWebhook 定义控制器的 webhook 服务器。
type ControllerWebhook struct {
	// Port 是 webhook 服务器监听的端口。
	// 用于设置 webhook.Server.Port。
	// +optional
	Port *int `json:"port,omitempty"`

	// Host 是 webhook 服务器绑定的主机名。
	// 用于设置 webhook.Server.Host。
	// +optional
	Host string `json:"host,omitempty"`

	// CertDir 是包含服务器密钥和证书的目录。
	// 如果未设置，webhook 服务器会在 {TempDir}/k8s-webhook-server/serving-certs 查找密钥和证书。
	// 密钥和证书文件名分别为 tls.key 和 tls.crt。
	// +optional
	CertDir string `json:"certDir,omitempty"`
}

// ControllerMetrics 定义指标配置。
type ControllerMetrics struct {
	// BindAddress 是控制器用于 prometheus 指标服务绑定的 TCP 地址。
	// 可以设置为 "0" 以禁用指标服务。
	// +optional
	BindAddress string `json:"bindAddress,omitempty"`

	// EnableClusterQueueResources，如果为 true，则会报告集群队列资源使用和配额指标。
	// +optional
	EnableClusterQueueResources bool `json:"enableClusterQueueResources,omitempty"`
}

// ControllerHealth 定义健康检查配置。
type ControllerHealth struct {
	// HealthProbeBindAddress 是控制器用于健康探针服务绑定的 TCP 地址。
	// 可以设置为 "0" 或 "" 以禁用健康探针服务。
	// +optional
	HealthProbeBindAddress string `json:"healthProbeBindAddress,omitempty"`

	// ReadinessEndpointName，默认为 "readyz"
	// +optional
	ReadinessEndpointName string `json:"readinessEndpointName,omitempty"`

	// LivenessEndpointName，默认为 "healthz"
	// +optional
	LivenessEndpointName string `json:"livenessEndpointName,omitempty"`
}

// ControllerConfigurationSpec 定义在管理器中注册的控制器的全局配置。
type ControllerConfigurationSpec struct {
	// GroupKindConcurrency 是 Kind 到该控制器允许的并发调和数的映射。
	//
	// 当控制器使用 builder 工具在此管理器中注册时，用户需在 For(...) 调用中指定控制器调和的类型。
	// 如果对象的 kind 匹配此映射中的某个键，则该控制器的并发数设置为指定的值。
	//
	// 键的格式应与 GroupKind.String() 一致，例如 apps 组下的 ReplicaSet（无论版本）为 `ReplicaSet.apps`。
	//
	// +optional
	GroupKindConcurrency map[string]int `json:"groupKindConcurrency,omitempty"`

	// CacheSyncTimeout 指定等待缓存同步的时间限制。
	// 如果未设置，默认为 2 分钟。
	// +optional
	CacheSyncTimeout *time.Duration `json:"cacheSyncTimeout,omitempty"`
}

// WaitForPodsReady 定义“等待 Pod 就绪”功能的配置，用于确保所有 Pod 在指定时间内就绪。
type WaitForPodsReady struct {
	// Enable 表示是否启用“等待 Pod 就绪”功能。
	// 默认为 false。
	Enable bool `json:"enable,omitempty"`

	// Timeout 定义已接纳工作负载达到 PodsReady=true 条件的时间。
	// 超时后，工作负载会被驱逐并重新排队到同一个集群队列。
	// 默认为 5 分钟。
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// BlockAdmission 为 true 时，集群队列会阻止所有后续作业的接纳，直到这些作业达到 PodsReady=true 条件。
	// 仅当 Enable 为 true 时，此设置才生效。
	BlockAdmission *bool `json:"blockAdmission,omitempty"`

	// RequeuingStrategy 定义工作负载重新排队的策略。
	// +optional
	RequeuingStrategy *RequeuingStrategy `json:"requeuingStrategy,omitempty"`

	// RecoveryTimeout 定义自上次转为 PodsReady=false 状态后（工作负载已接纳并运行）起的超时时间。
	// 例如 Pod 失败并等待替换 Pod 调度时。
	// 超时后，相应作业会再次挂起，并在退避延迟后重新排队。仅当 waitForPodsReady.enable=true 时才强制执行。
	// 如果未设置，则无超时。
	// +optional
	RecoveryTimeout *metav1.Duration `json:"recoveryTimeout,omitempty"`
}

type MultiKueue struct {
	// GCInterval 定义两次连续垃圾回收运行之间的时间间隔。
	// 默认为 1 分钟。如果为 0，则禁用垃圾回收。
	// +optional
	GCInterval *metav1.Duration `json:"gcInterval"`

	// Origin 定义用于跟踪工作负载创建者的标签值。
	// 这用于 multikueue 在组件（如垃圾收集器）中识别远程对象，并删除它们，如果本地对应对象不再存在。
	// +optional
	Origin *string `json:"origin,omitempty"`

	// WorkerLostTimeout 定义本地工作负载的 multikueue 准入检查状态在连接丢失时保持 Ready 的时间。
	//
	// 默认为 15 分钟。
	// +optional
	WorkerLostTimeout *metav1.Duration `json:"workerLostTimeout,omitempty"`
}

type RequeuingStrategy struct {
	// Timestamp 定义工作负载重新排队的标记。
	//
	// - `Eviction`（默认）表示从 Workload 的 `Evicted` 条件和 `PodsReadyTimeout` 原因。
	// - `Creation` 表示从 Workload .metadata.creationTimestamp。
	//
	// +optional
	Timestamp *RequeuingTimestamp `json:"timestamp,omitempty"`

	// BackoffLimitCount 定义重新排队尝试的最大次数。
	// 达到此数量后，工作负载将被停用（`.spec.activate`=`false`）。
	// 当为 null 时，工作负载将不断重复重新排队。
	//
	// 每次退避持续时间为 "b*2^(n-1)+Rand"，其中：
	// - "b" 表示由 "BackoffBaseSeconds" 参数设置的基础，
	// - "n" 表示 "workloadStatus.requeueState.count"，
	// - "Rand" 表示随机抖动。
	// 在此期间，工作负载被视为不可接纳，
	// 其他工作负载将有机会被接纳。
	// 默认情况下，连续的重新排队延迟约为：(60s, 120s, 240s, ...)。
	//
	// 默认为 null。
	// +optional
	BackoffLimitCount *int32 `json:"backoffLimitCount,omitempty"`

	// BackoffBaseSeconds 定义重新排队已驱逐工作负载的指数退避基础。
	//
	// 默认为 60。
	// +optional
	BackoffBaseSeconds *int32 `json:"backoffBaseSeconds,omitempty"`

	// BackoffMaxSeconds 定义重新排队已驱逐工作负载的最大退避时间。
	//
	// 默认为 3600。
	// +optional
	BackoffMaxSeconds *int32 `json:"backoffMaxSeconds,omitempty"`
}

type RequeuingTimestamp string

const (
	// CreationTimestamp 标记（从 Workload .metadata.creationTimestamp）。
	CreationTimestamp RequeuingTimestamp = "Creation"

	// EvictionTimestamp 标记（从 Workload .status.conditions）。
	EvictionTimestamp RequeuingTimestamp = "Eviction"
)

type InternalCertManagement struct {
	// Enable 控制是否对 webhook 和指标端点使用内部证书管理。
	// 启用后，Kueue 使用库生成和自签名证书。
	// 禁用后，您需要通过第三方证书提供 webhook 和指标证书。
	// 此 secret 挂载到 kueue 控制器管理器 pod。webhook 的挂载路径为 /tmp/k8s-webhook-server/serving-certs，指标端点的预期路径为 `/etc/kueue/metrics/certs`。
	// 密钥和证书文件名为 tls.key 和 tls.crt。
	Enable *bool `json:"enable,omitempty"`

	// WebhookServiceName 是作为 DNSName 一部分使用的 Service 名称。
	// 默认为 kueue-webhook-service。
	WebhookServiceName *string `json:"webhookServiceName,omitempty"`

	// WebhookSecretName 是用于存储 CA 和服务器证书的 Secret 名称。
	// 默认为 kueue-webhook-server-cert。
	WebhookSecretName *string `json:"webhookSecretName,omitempty"`
}

type ClientConnection struct {
	// QPS 控制允许 K8S api server 连接的每秒查询数。
	QPS *float32 `json:"qps,omitempty"`

	// Burst 允许客户端在超出其速率时累积额外的查询。
	Burst *int32 `json:"burst,omitempty"`
}

type Integrations struct {
	// 启用的框架名称列表。
	// 可能选项：
	//  - "batch/job"
	//  - "pod"
	//  - "deployment"（需要启用 pod 集成）
	//  - "statefulset"（需要启用 pod 集成）
	//  - "leaderworkerset.x-k8s.io/leaderworkerset"（需要启用 pod 集成）
	Frameworks []string `json:"frameworks,omitempty"`
	// 由外部控制器管理的 GroupVersionKind 列表；
	// 预期格式为 `Kind.version.group.com`。
	ExternalFrameworks []string `json:"externalFrameworks,omitempty"`
	// PodOptions 定义 kueue 控制器对 pod 对象的行为。
	// 已废弃：此字段将在 v1beta2 移除，请使用 ManagedJobsNamespaceSelector（https://kueue.sigs.k8s.io/docs/tasks/run/plain_pods/）代替。
	PodOptions *PodIntegrationOptions `json:"podOptions,omitempty"`

	// labelKeysToCopy 是从作业复制到工作负载对象的标签键列表。
	// 作业不需要包含此列表中的所有标签。如果作业缺少此列表中某个键的标签，则构造的工作负载对象将不包含该标签。
	// 在复合作业（pod 组）从多个对象创建工作负载时，如果多个对象具有此列表中某个键的标签，则这些标签的值必须匹配，否则工作负载创建将失败。
	// 标签仅在创建工作负载时复制，并且不会在底层作业的标签更改时更新。
	LabelKeysToCopy []string `json:"labelKeysToCopy,omitempty"`
}

type PodIntegrationOptions struct {
	// NamespaceSelector 可用于排除一些命名空间中的 pod 重新同步
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`
	// PodSelector 可用于选择要重新同步的 pod
	PodSelector *metav1.LabelSelector `json:"podSelector,omitempty"`
}

type QueueVisibility struct {
	// ClusterQueues 是配置，用于暴露集群队列中最顶层待处理工作负载的信息。
	ClusterQueues *ClusterQueueVisibility `json:"clusterQueues,omitempty"`

	// UpdateIntervalSeconds 指定更新队列中最顶层待处理工作负载结构的时间间隔。
	// 最小值为 1。
	// 默认为 5。
	UpdateIntervalSeconds int32 `json:"updateIntervalSeconds,omitempty"`
}

type ClusterQueueVisibility struct {
	// MaxCount 指示在集群队列状态中暴露的待处理工作负载的最大数量。
	// 当值设置为 0 时，则禁用集群队列可见性更新。
	// 最大值为 4000。
	// 默认为 10。
	MaxCount int32 `json:"maxCount,omitempty"`
}

type Resources struct {
	// ExcludedResourcePrefixes 定义 Kueue 应忽略的资源。
	ExcludeResourcePrefixes []string `json:"excludeResourcePrefixes,omitempty"`

	// Transformations 定义如何将 PodSpec 资源转换为 Workload 资源请求。
	// 这旨在是一个键值对，其中键是输入资源名称（由验证代码强制执行）。
	Transformations []ResourceTransformation `json:"transformations,omitempty"`
}

type ResourceTransformationStrategy string

const Retain ResourceTransformationStrategy = "Retain"
const Replace ResourceTransformationStrategy = "Replace"

type ResourceTransformation struct {
	// Input 是输入资源名称。
	Input corev1.ResourceName `json:"input"`

	// Strategy 指定输入资源是否应替换或保留。
	// 默认为 Retain
	Strategy *ResourceTransformationStrategy `json:"strategy,omitempty"`

	// Outputs 指定输入资源每单位数量的输出资源和数量。
	// 空 Outputs 与 `Replace` 策略结合使用会导致 Kueue 忽略输入资源。
	Outputs corev1.ResourceList `json:"outputs,omitempty"`
}

type PreemptionStrategy string

const (
	LessThanOrEqualToFinalShare PreemptionStrategy = "LessThanOrEqualToFinalShare"
	LessThanInitialShare        PreemptionStrategy = "LessThanInitialShare"
)

type FairSharing struct {
	// enable 表示是否启用所有 cohort 的公平共享。
	// 默认为 false。
	Enable bool `json:"enable"`

	// preemptionStrategies 表示预抢占应满足哪些约束。
	// 预抢占算法仅在预抢占工作负载（抢占者）不满足后续策略时才使用下一个策略。
	// 可能值为：
	// - LessThanOrEqualToFinalShare：仅当抢占者 CQ 与抢占者工作负载的共享小于或等于预抢占者 CQ 与未被抢占工作负载的共享时才抢占工作负载。
	//   这可能会优先抢占较小的工作负载，无论优先级或开始时间如何，以保持 CQ 的共享尽可能高。
	// - LessThanInitialShare：仅当抢占者 CQ 与传入工作负载的共享严格小于预抢占者 CQ 时才抢占工作负载。
	//   此策略不依赖于被抢占工作负载的共享使用情况。
	//   因此，策略会选择优先级最低且开始时间最新的工作负载进行抢占。
	// 默认策略为 ["LessThanOrEqualToFinalShare", "LessThanInitialShare"]。
	PreemptionStrategies []PreemptionStrategy `json:"preemptionStrategies,omitempty"`
}

type AdmissionFairSharing struct {
	// usageHalfLifeTime 表示当前使用量在达到一半后衰减的时间。
	// 如果设置为 0，则使用量将立即重置为 0。
	UsageHalfLifeTime metav1.Duration `json:"usageHalfLifeTime"`

	// usageSamplingInterval 表示 Kueue 更新 consumedResources 的时间间隔。
	// 默认为 5 分钟。
	UsageSamplingInterval metav1.Duration `json:"usageSamplingInterval"`

	// resourceWeights 为资源分配权重，然后用于计算 LocalQueue 的资源使用情况和排序工作负载。
	// 默认为 1。
	ResourceWeights map[corev1.ResourceName]float64 `json:"resourceWeights,omitempty"`
}

// ObjectRetentionPolicies 持有不同对象类型的保留设置。
type ObjectRetentionPolicies struct {
	// Workloads 配置 Workloads 的保留。
	// 为 nil 时禁用 Workloads 的自动删除。
	// +optional
	Workloads *WorkloadRetentionPolicy `json:"workloads,omitempty"`
}

// WorkloadRetentionPolicy 定义 Workloads 应删除的时间。
type WorkloadRetentionPolicy struct {
	// AfterFinished 是 Workload 完成后等待删除的时间。
	// 持续时间为 0 时立即删除。
	// 为 nil 时禁用自动删除。
	// 使用 metav1.Duration（例如 "10m", "1h30m"）表示。
	// +optional
	AfterFinished *metav1.Duration `json:"afterFinished,omitempty"`

	// AfterDeactivatedByKueue 是 *任何* Kueue 管理的 Workload（如 Job、JobSet 或其他自定义工作负载类型）被 Kueue 标记为停用后自动删除的等待时间。
	// 停用工作负载的删除可能会级联到不是 Kueue 创建的对象，因为删除父 Workload 所有者（例如 JobSet）可以触发依赖资源的垃圾回收。
	// 持续时间为 0 时立即删除。
	// 为 nil 时禁用自动删除。
	// 使用 metav1.Duration（例如 "10m", "1h30m"）表示。
	// +optional
	AfterDeactivatedByKueue *metav1.Duration `json:"afterDeactivatedByKueue,omitempty"`
}
