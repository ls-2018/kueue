package queue

import kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"

// StatusChecker checks status of clusterQueue.
type StatusChecker interface {
	// ClusterQueueActive returns whether the clusterQueue is active.
	ClusterQueueActive(name kueue.ClusterQueueReference) bool
}
