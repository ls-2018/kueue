package constants

import "sigs.k8s.io/kueue/pkg/constants"

const (
	// PodFinalizer 用于标记由 Kueue 管理的 Pod 的 Finalizer，确保资源被正确清理。
	PodFinalizer = constants.ManagedByKueueLabelKey
	// SchedulingGateName 是调度门的名称，用于控制 Pod 是否可以被调度。
	SchedulingGateName = "kueue.x-k8s.io/admission"
	// SuspendedByParentAnnotation 标记 Pod 是否被其父对象（如 Job）挂起。
	SuspendedByParentAnnotation = "kueue.x-k8s.io/pod-suspending-parent"
	// GroupNameLabel 表示 Pod 所属的组名，用于分组调度。
	GroupNameLabel = "kueue.x-k8s.io/pod-group-name"
	// GroupTotalCountAnnotation 表示该组中 Pod 的总数。
	GroupTotalCountAnnotation = "kueue.x-k8s.io/pod-group-total-count"
	// GroupFastAdmissionAnnotationKey 用于标记该组是否支持快速准入。
	GroupFastAdmissionAnnotationKey = "kueue.x-k8s.io/pod-group-fast-admission"
	// GroupFastAdmissionAnnotationValue 表示快速准入的值，通常为 "true"。
	GroupFastAdmissionAnnotationValue = "true"
	// GroupServingAnnotationKey 用于标记该组是否为服务型工作负载。
	GroupServingAnnotationKey = "kueue.x-k8s.io/pod-group-serving"
	// GroupServingAnnotationValue 表示服务型工作负载的值，通常为 "true"。
	GroupServingAnnotationValue = "true"
	// RoleHashAnnotation 用于标记 Pod 的角色哈希值，便于区分不同角色的 Pod。
	RoleHashAnnotation = "kueue.x-k8s.io/role-hash"
	// RetriableInGroupAnnotationKey 标记该 Pod 是否在组内可重试。
	RetriableInGroupAnnotationKey = "kueue.x-k8s.io/retriable-in-group"
	// RetriableInGroupAnnotationValue 表示不可重试的值，通常为 "false"。
	RetriableInGroupAnnotationValue = "false"
	// IsGroupWorkloadAnnotationKey 标记该 Pod 是否为组工作负载。
	IsGroupWorkloadAnnotationKey = "kueue.x-k8s.io/is-group-workload"
	// IsGroupWorkloadAnnotationValue 表示是组工作负载的值，通常为 "true"。
	IsGroupWorkloadAnnotationValue = "true"
)
