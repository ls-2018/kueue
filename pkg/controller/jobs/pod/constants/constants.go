package constants

import "sigs.k8s.io/kueue/pkg/over_constants"

const (
	PodFinalizer = over_constants.ManagedByKueueLabelKey

	SchedulingGateName = "kueue.x-k8s.io/admission"

	SuspendedByParentAnnotation       = "kueue.x-k8s.io/pod-suspending-parent"
	GroupNameLabel                    = "kueue.x-k8s.io/pod-group-name"
	GroupTotalCountAnnotation         = "kueue.x-k8s.io/pod-group-total-count"
	GroupFastAdmissionAnnotationKey   = "kueue.x-k8s.io/pod-group-fast-admission"
	GroupFastAdmissionAnnotationValue = "true"
	GroupServingAnnotationKey         = "kueue.x-k8s.io/pod-group-serving"
	GroupServingAnnotationValue       = "true"
	RoleHashAnnotation                = "kueue.x-k8s.io/role-hash"
	RetriableInGroupAnnotationKey     = "kueue.x-k8s.io/retriable-in-group"
	RetriableInGroupAnnotationValue   = "false"
	IsGroupWorkloadAnnotationKey      = "kueue.x-k8s.io/is-group-workload"
	IsGroupWorkloadAnnotationValue    = "true"
)
