package constants

import (
	"time"
)

const (
	KueueName              = "kueue"
	JobControllerName      = KueueName + "-job-controller"
	WorkloadControllerName = KueueName + "-workload-controller"
	AdmissionName          = KueueName + "-admission"
	ReclaimablePodsMgr     = KueueName + "-reclaimable-pods"

	// UpdatesBatchPeriod is the batch period to hold workload updates
	// before syncing a Queue and ClusterQueue objects.
	UpdatesBatchPeriod = time.Second

	// DefaultPriority is used to set priority of workloads
	// that do not specify any priority class and there is no priority class
	// marked as default.
	DefaultPriority = 0

	WorkloadPriorityClassSource = "kueue.x-k8s.io/workloadpriorityclass"
	PodPriorityClassSource      = "scheduling.k8s.io/priorityclass"

	DefaultPendingWorkloadsLimit = 1000

	// ManagedByKueueLabelKey label that signalize that an object is managed by Kueue
	ManagedByKueueLabelKey   = "kueue.x-k8s.io/managed"
	ManagedByKueueLabelValue = "true"
)
