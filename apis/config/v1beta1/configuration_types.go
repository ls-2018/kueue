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
	// 如果文件不存在，默认值为 kueue-system。
	Namespace *string `json:"namespace,omitempty"`

	// ControllerManager 返回控制器的配置
	ControllerManager `json:",inline"`

	// ManageJobsWithoutQueueName 控制 Kueue 是否管理未设置标签 kueue.x-k8s.io/queue-name 的作业。
	// 如果设置为 true，则这些作业将被挂起，除非分配了队列并最终被接收，否则永远不会启动。这也适用于启动 kueue 控制器之前创建的作业。
	// 默认为 false，因此这些作业不受管理，如果创建时未挂起，将立即启动。
	ManageJobsWithoutQueueName bool `json:"manageJobsWithoutQueueName"`

	// ManagedJobsNamespaceSelector 提供基于命名空间的机制以使作业不被 Kueue 管理。
	//
	// 对于基于 Pod 的集成（pod、deployment、statefulset 等），只有命名空间匹配 ManagedJobsNamespaceSelector 的作业才有资格被 Kueue 管理。
	// 在不匹配命名空间中的 Pod、deployment 等永远不会被 Kueue 管理，即使它们有 kueue.x-k8s.io/queue-name 标签。
	// 这种强豁免确保 Kueue 不会干扰系统命名空间的基本操作。
	//
	// 对于所有其他集成，ManagedJobsNamespaceSelector 仅通过调节 ManageJobsWithoutQueueName 的效果提供较弱的豁免。
	// 对于这些集成，具有 kueue.x-k8s.io/queue-name 标签的作业将始终由 Kueue 管理。没有该标签的作业仅在 ManageJobsWithoutQueueName 为 true 且命名空间匹配 ManagedJobsNamespaceSelector 时才由 Kueue 管理。
	ManagedJobsNamespaceSelector *metav1.LabelSelector `json:"managedJobsNamespaceSelector,omitempty"`

	// InternalCertManagement 是 internalCertManagement 的配置
	InternalCertManagement *InternalCertManagement `json:"internalCertManagement,omitempty"`

	// WaitForPodsReady 是为作业提供基于时间的全有或全无调度语义的配置，
	// 通过确保所有 pod 在指定时间内就绪（运行并通过就绪探针）。如果超时，则驱逐该工作负载。
	WaitForPodsReady *WaitForPodsReady `json:"waitForPodsReady,omitempty"`

	// ClientConnection 提供 Kubernetes API server 客户端的其他配置选项。
	ClientConnection *ClientConnection `json:"clientConnection,omitempty"`

	// Integrations 提供 AI/ML/Batch 框架集成（包括 K8S job）的配置选项。
	Integrations *Integrations `json:"integrations,omitempty"`

	// QueueVisibility 用于暴露关于顶部待处理工作负载的信息。
	// 已废弃：该字段将在 v1beta2 移除，请使用 VisibilityOnDemand
	// (https://kueue.sigs.k8s.io/docs/tasks/manage/monitor_pending_workloads/pending_workloads_on_demand/)
	// 代替。
	QueueVisibility *QueueVisibility `json:"queueVisibility,omitempty"`

	// MultiKueue 控制 MultiKueue AdmissionCheck Controller 的行为。
	MultiKueue *MultiKueue `json:"multiKueue,omitempty"`

	// FairSharing 控制集群范围内的公平共享语义。
	FairSharing *FairSharing `json:"fairSharing,omitempty"`

	// admissionFairSharing 表示开启 `AdmissionTime` 模式下的 FairSharing 配置
	AdmissionFairSharing *AdmissionFairSharing `json:"admissionFairSharing,omitempty"`

	// Resources 提供处理资源的其他配置选项。
	Resources *Resources `json:"resources,omitempty"`

	// FeatureGates 是特性名称到布尔值的映射，允许覆盖特性的默认启用状态。该映射不能与通过命令行参数 "--feature-gates" 传递特性列表同时使用。
	FeatureGates map[string]bool `json:"featureGates,omitempty"`

	// ObjectRetentionPolicies 提供 Kueue 管理对象自动删除的配置选项。nil 值禁用所有自动删除。
	// +optional
	ObjectRetentionPolicies *ObjectRetentionPolicies `json:"objectRetentionPolicies,omitempty"`
}

type ControllerManager struct {
	// Webhook 包含控制器 webhook 的配置
	// +optional
	Webhook ControllerWebhook `json:"webhook,omitempty"`

	// LeaderElection 是配置 manager.Manager 选举的 LeaderElection 配置
	// +optional
	LeaderElection *configv1alpha1.LeaderElectionConfiguration `json:"leaderElection,omitempty"`

	// Metrics 包含控制器指标的配置
	// +optional
	Metrics ControllerMetrics `json:"metrics,omitempty"`

	// Health 包含控制器健康检查的配置
	// +optional
	Health ControllerHealth `json:"health,omitempty"`

	// PprofBindAddress 是控制器用于绑定 pprof 服务的 TCP 地址。
	// 可以设置为 "" 或 "0" 以禁用 pprof 服务。
	// 由于 pprof 可能包含敏感信息，公开前请确保保护好。
	// +optional
	PprofBindAddress string `json:"pprofBindAddress,omitempty"`

	// Controller 包含在此 manager 注册的控制器的全局配置选项。
	// +optional
	Controller *ControllerConfigurationSpec `json:"controller,omitempty"`
}

// ControllerWebhook 定义控制器的 webhook 服务器。
type ControllerWebhook struct {
	// Port 是 webhook 服务器服务的端口。
	// 用于设置 webhook.Server.Port。
	// +optional
	Port *int `json:"port,omitempty"`

	// Host 是 webhook 服务器绑定的主机名。
	// 用于设置 webhook.Server.Host。
	// +optional
	Host string `json:"host,omitempty"`

	// CertDir 是包含服务器密钥和证书的目录。
	// 如果未设置，webhook 服务器会在 {TempDir}/k8s-webhook-server/serving-certs 查找服务器密钥和证书。
	// 服务器密钥和证书必须分别命名为 tls.key 和 tls.crt。
	// +optional
	CertDir string `json:"certDir,omitempty"`
}

// ControllerMetrics 定义指标配置。
type ControllerMetrics struct {
	// BindAddress 是控制器用于绑定 prometheus 指标服务的 TCP 地址。
	// 可以设置为 "0" 以禁用指标服务。
	// +optional
	BindAddress string `json:"bindAddress,omitempty"`

	// EnableClusterQueueResources 如果为 true，则会报告集群队列资源使用和配额指标。
	// +optional
	EnableClusterQueueResources bool `json:"enableClusterQueueResources,omitempty"`
}

// ControllerHealth 定义健康检查配置。
type ControllerHealth struct {
	// HealthProbeBindAddress 是控制器用于健康探针服务的 TCP 地址。
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

// ControllerConfigurationSpec 定义注册到 manager 的控制器的全局配置。
type ControllerConfigurationSpec struct {
	// GroupKindConcurrency 是从 Kind 到控制器并发重调度的映射。
	//
	// 当在 manager 中使用 builder 工具注册控制器时，用户必须在 For(...) 调用中指定控制器重调度的类型。
	// 如果对象的种类与该映射中的某个键匹配，则该控制器的并发设置为指定的数量。
	//
	// 键的格式应与 GroupKind.String() 一致，例如 ReplicaSet 在 apps 组（无论版本如何）中将是 `ReplicaSet.apps`。
	//
	// +optional
	GroupKindConcurrency map[string]int `json:"groupKindConcurrency,omitempty"`

	// CacheSyncTimeout 是指设置缓存同步超时的时间限制。
	// 如果未设置，则默认为 2 分钟。
	// +optional
	CacheSyncTimeout *time.Duration `json:"cacheSyncTimeout,omitempty"`
}

// WaitForPodsReady 是为作业提供基于时间的全有或全无调度语义的配置，
// 通过确保所有 pod 在指定时间内就绪（运行并通过就绪探针）。如果超时，则驱逐该工作负载。
type WaitForPodsReady struct {
	// Enable 表示是否启用等待 Pods 就绪功能。
	// 默认为 false。
	Enable bool `json:"enable,omitempty"`

	// Timeout 定义了已接收工作负载达到 PodsReady=true 条件的时间。
	// 当超时时，工作负载被驱逐并重新排队到同一个集群队列。
	// 默认为 5min。
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// BlockAdmission 当为 true 时，集群队列将阻止所有后续作业的准入，直到作业达到 PodsReady=true 条件。
	// 此设置仅在 `Enable` 设置为 true 时才生效。
	BlockAdmission *bool `json:"blockAdmission,omitempty"`

	// RequeuingStrategy 定义了重新排队 Workload 的策略。
	// +optional
	RequeuingStrategy *RequeuingStrategy `json:"requeuingStrategy,omitempty"`

	// RecoveryTimeout 定义了一个可选的超时，从 Workload 被 Admitted 并运行后的最后一次 PodsReady=false 状态开始测量。
	// 这种状态可能发生在 Pod 失败且替换 Pod 等待调度时。
	// 超时后，相应的作业将被暂停，并在回退延迟后重新排队。超时仅在 waitForPodsReady.enable=true 时强制执行。
	// 如果未设置，则没有超时。
	// +optional
	RecoveryTimeout *metav1.Duration `json:"recoveryTimeout,omitempty"`
}

type MultiKueue struct {
	// GCInterval 定义了两次连续垃圾回收运行之间的时间间隔。
	// 默认为 1min。如果为 0，则禁用垃圾回收。
	// +optional
	GCInterval *metav1.Duration `json:"gcInterval"`

	// Origin 定义了一个标签值，用于跟踪 worker 集群中工作负载的创建者。
	// 这用于 multikueue 在其组件（如垃圾收集器）中识别远程对象，
	// 如果本地对应对象不再存在，则删除它们。
	// +optional
	Origin *string `json:"origin,omitempty"`

	// WorkerLostTimeout 定义了本地工作负载的 multikueue 准入检查状态在与其预留 worker 集群断开连接后保持 Ready 的时间。
	//
	// 默认为 15 分钟。
	// +optional
	WorkerLostTimeout *metav1.Duration `json:"workerLostTimeout,omitempty"`
}

type RequeuingStrategy struct {
	// Timestamp 定义了重新排队 Workload 的时间戳。
	// 可能的值是：
	//
	// - `Eviction`（默认）表示从 Workload `Evicted` 条件获取时间，原因 `PodsReadyTimeout`。
	// - `Creation` 表示从 Workload .metadata.creationTimestamp 获取时间。
	//
	// +optional
	Timestamp *RequeuingTimestamp `json:"timestamp,omitempty"`

	// BackoffLimitCount 定义了最大重试次数。
	// 达到此数量后，工作负载将被停用（`.spec.activate`=`false`）。
	// 当为 null 时，工作负载将不断重复重试。
	//
	// 每次回退持续时间约为 "b*2^(n-1)+Rand"，其中：
	// - "b" 表示由 "BackoffBaseSeconds" 参数设置的基础，
	// - "n" 表示 "workloadStatus.requeueState.count"，
	// - "Rand" 表示随机抖动。
	// 在此期间，工作负载被视为不合格，
	// 其他工作负载将有机会被接收。
	// 默认情况下，连续重试延迟约为：(60s, 120s, 240s, ...)。
	//
	// 默认为 null。
	// +optional
	BackoffLimitCount *int32 `json:"backoffLimitCount,omitempty"`

	// BackoffBaseSeconds 定义了重新排队已驱逐工作负载的指数回退基础。
	//
	// 默认为 60。
	// +optional
	BackoffBaseSeconds *int32 `json:"backoffBaseSeconds,omitempty"`

	// BackoffMaxSeconds 定义了重新排队已驱逐工作负载的最大回退时间。
	//
	// 默认为 3600。
	// +optional
	BackoffMaxSeconds *int32 `json:"backoffMaxSeconds,omitempty"`
}

type RequeuingTimestamp string

const (
	// CreationTimestamp 时间戳（从 Workload .metadata.creationTimestamp 获取）。
	CreationTimestamp RequeuingTimestamp = "Creation"

	// EvictionTimestamp 时间戳（从 Workload .status.conditions 获取）。
	EvictionTimestamp RequeuingTimestamp = "Eviction"
)

type InternalCertManagement struct {
	// Enable 控制是否使用内部证书管理。
	// 启用时，Kueue 使用库来生成和自签名证书。
	// 禁用时，您需要通过第三方证书提供 webhook 和指标端点的证书。
	// 此 secret 挂载到 kueue 控制器管理器 pod 中。webhook 的挂载路径为 /tmp/k8s-webhook-server/serving-certs，
	// 指标端点的预期路径为 `/etc/kueue/metrics/certs`。密钥和证书分别命名为 tls.key 和 tls.crt。
	Enable *bool `json:"enable,omitempty"`

	// WebhookServiceName 是作为 DNSName 一部分使用的 Service 名称。
	// 默认为 kueue-webhook-service。
	WebhookServiceName *string `json:"webhookServiceName,omitempty"`

	// WebhookSecretName 是用于存储 CA 和服务器证书的 Secret 名称。
	// 默认为 kueue-webhook-server-cert。
	WebhookSecretName *string `json:"webhookSecretName,omitempty"`
}

type ClientConnection struct {
	// QPS 控制允许 K8S API server 连接的每秒查询数。
	QPS *float32 `json:"qps,omitempty"`

	// Burst 允许客户端在超出其速率时积累额外的查询。
	Burst *int32 `json:"burst,omitempty"`
}

type Integrations struct {
	// List of framework names to be enabled.
	// Possible options:
	//  - "batch/job"
	//  - "pod"
	//  - "deployment" (requires enabling pod integration)
	//  - "statefulset" (requires enabling pod integration)
	//  - "leaderworkerset.x-k8s.io/leaderworkerset" (requires enabling pod integration)
	Frameworks []string `json:"frameworks,omitempty"`
	// List of GroupVersionKinds that are managed for Kueue by external controllers;
	// the expected format is `Kind.version.group.com`.
	ExternalFrameworks []string `json:"externalFrameworks,omitempty"`
	// PodOptions 定义 kueue 控制器对 pod 对象的行为
	// 已废弃：此字段将在 v1beta2 移除，请使用 ManagedJobsNamespaceSelector
	// (https://kueue.sigs.k8s.io/docs/tasks/run/plain_pods/)
	// 代替。
	PodOptions *PodIntegrationOptions `json:"podOptions,omitempty"`

	// labelKeysToCopy 是从作业复制到工作负载对象的标签键列表。
	// 作业不必具有此列表中的所有标签。如果作业缺少此列表中某个键的标签，则构造的工作负载对象将不包含该标签。
	// 在从可组合作业（pod 组）创建工作负载时，如果多个对象具有此列表中某个键的标签，则这些标签的值必须匹配，
	// 否则工作负载创建将失败。标签仅在创建工作负载时复制，即使底层作业的标签已更改，也不会更新。
	LabelKeysToCopy []string `json:"labelKeysToCopy,omitempty"`
}

type PodIntegrationOptions struct {
	// NamespaceSelector 可用于排除一些命名空间从 pod 重调度的范围。
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`
	// PodSelector 可用于选择要重调度的 pod。
	PodSelector *metav1.LabelSelector `json:"podSelector,omitempty"`
}

type QueueVisibility struct {
	// ClusterQueues 是配置，用于暴露集群队列中顶部待处理工作负载的信息。
	ClusterQueues *ClusterQueueVisibility `json:"clusterQueues,omitempty"`

	// UpdateIntervalSeconds 指定更新队列中顶部待处理工作负载结构的时间间隔。
	// 最小值为 1。
	// 默认为 5。
	UpdateIntervalSeconds int32 `json:"updateIntervalSeconds,omitempty"`
}

type ClusterQueueVisibility struct {
	// MaxCount 表示在集群队列状态中暴露的最大待处理工作负载数量。
	// 当值设置为 0 时，则禁用集群队列可见性更新。
	// 最大值为 4000。
	// 默认为 10。
	MaxCount int32 `json:"maxCount,omitempty"`
}

type Resources struct {
	// ExcludedResourcePrefixes 定义哪些资源应被 Kueue 忽略。
	ExcludeResourcePrefixes []string `json:"excludeResourcePrefixes,omitempty"`

	// Transformations 定义如何将 PodSpec 资源转换为 Workload 资源请求。
	// 这旨在是一个键为 Input 的映射（由验证代码强制执行）
	Transformations []ResourceTransformation `json:"transformations,omitempty"`
}

type ResourceTransformationStrategy string

const Retain ResourceTransformationStrategy = "Retain"
const Replace ResourceTransformationStrategy = "Replace"

type ResourceTransformation struct {
	// Input 是输入资源的名称。
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

	// preemptionStrategies 表示哪些约束应由抢占满足。
	// 抢占算法仅在抢占者（preemptor）不满足下一个策略时才使用列表中的下一个策略。
	// 可能的值是：
	// - LessThanOrEqualToFinalShare：仅当抢占者 CQ 与抢占者工作负载的共享小于或等于被抢占 CQ 的共享时才抢占工作负载。
	//   这可能会优先抢占较小的工作负载，无论优先级或开始时间如何，以保持 CQ 的共享尽可能高。
	// - LessThanInitialShare：仅当抢占者 CQ 与传入工作负载的共享严格小于被抢占 CQ 的共享时才抢占工作负载。
	//   此策略不依赖于被抢占工作负载的共享使用情况。
	//   结果是，策略选择优先抢占具有最低优先级和最新开始时间的作业。
	// 默认策略是 ["LessThanOrEqualToFinalShare", "LessThanInitialShare"]。
	PreemptionStrategies []PreemptionStrategy `json:"preemptionStrategies,omitempty"`
}

type AdmissionFairSharing struct {
	// usageHalfLifeTime 表示当前使用量在达到一半后衰减的时间。
	// 如果设置为 0，则使用量将立即重置为 0。
	UsageHalfLifeTime metav1.Duration `json:"usageHalfLifeTime"`

	// usageSamplingInterval 表示 Kueue 更新 FairSharingStatus 中 consumedResources 的频率。
	// 默认为 5min。
	UsageSamplingInterval metav1.Duration `json:"usageSamplingInterval"`

	// resourceWeights 为资源分配权重，然后用于计算 LocalQueue 的资源使用情况和排序 Workloads。
	// 默认为 1。
	ResourceWeights map[corev1.ResourceName]float64 `json:"resourceWeights,omitempty"`
}

// ObjectRetentionPolicies 提供 Kueue 管理对象自动删除的配置选项。nil 值禁用所有自动删除。
type ObjectRetentionPolicies struct {
	// Workloads 配置 Workloads 的保留。
	// 一个 nil 值禁用 Workloads 的自动删除。
	// +optional
	Workloads *WorkloadRetentionPolicy `json:"workloads,omitempty"`
}

// WorkloadRetentionPolicy 定义 Workloads 应在何时删除的策略。
type WorkloadRetentionPolicy struct {
	// AfterFinished 是 Workload 完成后等待删除的时间。
	// 持续时间为 0 将立即删除。
	// 一个 nil 值禁用自动删除。
	// 使用 metav1.Duration（例如 "10m", "1h30m"）表示。
	// +optional
	AfterFinished *metav1.Duration `json:"afterFinished,omitempty"`

	// AfterDeactivatedByKueue 是 *任何* Kueue 管理的 Workload（例如 Job、JobSet 或其他自定义工作负载类型）
	// 被 Kueue 标记为 deactivated 后自动删除的时间。
	// 已停用工作负载的删除可能会级联到不是由 Kueue 创建的对象，
	// 因为删除父 Workload 所有者（例如 JobSet）可以触发依赖资源的垃圾回收。
	// 持续时间为 0 将立即删除。
	// 一个 nil 值禁用自动删除。
	// 使用 metav1.Duration（例如 "10m", "1h30m"）表示。
	// +optional
	AfterDeactivatedByKueue *metav1.Duration `json:"afterDeactivatedByKueue,omitempty"`
}
