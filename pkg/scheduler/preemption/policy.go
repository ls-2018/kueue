package preemption

import (
	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/cache"
)

// CanAlwaysReclaim indicates that the CQ is guaranteed to
// be able to reclaim the capacity of workloads borrowing
// its capacity.
func CanAlwaysReclaim(cq *cache.ClusterQueueSnapshot) bool {
	return cq.Preemption.ReclaimWithinCohort == kueue.PreemptionPolicyAny
}
