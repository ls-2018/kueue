package preemption

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/sets"

	"sigs.k8s.io/kueue/pkg/cache"
	"sigs.k8s.io/kueue/pkg/resources"
	preemptioncommon "sigs.k8s.io/kueue/pkg/scheduler/preemption/common"
	"sigs.k8s.io/kueue/pkg/workload"
)

func NewOracle(preemptor *Preemptor, snapshot *cache.Snapshot) *PreemptionOracle {
	return &PreemptionOracle{preemptor, snapshot}
}

type PreemptionOracle struct {
	preemptor *Preemptor
	snapshot  *cache.Snapshot
}

// SimulatePreemption runs the preemption algorithm for a given flavor resource to check if
// preemption and reclaim are possible in this flavor resource.
func (p *PreemptionOracle) SimulatePreemption(log logr.Logger, cq *cache.ClusterQueueSnapshot, wl workload.Info, fr resources.FlavorResource, quantity int64) preemptioncommon.PreemptionPossibility {
	candidates := p.preemptor.getTargets(&preemptionCtx{
		log:               log,
		preemptor:         wl,
		preemptorCQ:       p.snapshot.ClusterQueue(wl.ClusterQueue),
		snapshot:          p.snapshot,
		frsNeedPreemption: sets.New(fr),
		workloadUsage:     workload.Usage{Quota: resources.FlavorResourceQuantities{fr: quantity}},
	})
	if len(candidates) == 0 {
		return preemptioncommon.NoCandidates
	}
	for _, candidate := range candidates {
		if candidate.WorkloadInfo.ClusterQueue == cq.Name {
			return preemptioncommon.Preempt
		}
	}
	return preemptioncommon.Reclaim
}
