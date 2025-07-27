package constants

import kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"

const (
	// QueueLabel 是工作负载中保存队列名称的标签键。
	QueueLabel = "kueue.x-k8s.io/queue-name"

	// DefaultLocalQueueName 是默认 LocalQueue 的名称，当启用 LocalQueueDefaulting 特性且未指定 QueueLabel 时应用。
	DefaultLocalQueueName kueue.LocalQueueName = "default"

	// QueueAnnotation 是工作负载中保存队列名称的注解键。
	//
	// 已弃用：请使用 QueueLabel 作为标签键。
	QueueAnnotation = QueueLabel

	// PrebuiltWorkloadLabel 是作业的标签键，保存要使用的预构建工作负载的名称。
	PrebuiltWorkloadLabel = "kueue.x-k8s.io/prebuilt-workload-name" // 提前准备好的   workload

	// JobUIDLabel 是工作负载资源中的标签键，保存所有者作业的 UID。
	JobUIDLabel = "kueue.x-k8s.io/job-uid"

	// WorkloadPriorityClassLabel 是工作负载中保存 workloadPriorityClass 名称的标签键。
	// 该标签始终可变，因为它可能对抢占有用。
	WorkloadPriorityClassLabel = "kueue.x-k8s.io/priority-class"

	// ProvReqAnnotationPrefix 是应作为参数传递给 ProvisioningRequest 的注解前缀。
	ProvReqAnnotationPrefix = "provreq.kueue.x-k8s.io/"

	// MaxExecTimeSecondsLabel 是作业中保存最大执行时间的标签键。
	MaxExecTimeSecondsLabel = `kueue.x-k8s.io/max-exec-time-seconds`

	// PodSetLabel 是设置在作业 PodTemplate 上的标签，用于指示与 PodTemplate 对应的已接纳 Workload 的 PodSet 名称。
	// 启动作业时设置该标签，停止作业时移除。
	PodSetLabel = "kueue.x-k8s.io/podset"
)
