package preemption

import (
	"sigs.k8s.io/kueue/pkg/workload"
)

type PreemptedWorkloads map[workload.Reference]*workload.Info

func (p PreemptedWorkloads) HasAny(newTargets []*Target) bool {
	for _, target := range newTargets {
		if _, found := p[workload.Key(target.WorkloadInfo.Obj)]; found {
			return true
		}
	}
	return false
}

func (p PreemptedWorkloads) Insert(newTargets []*Target) {
	for _, target := range newTargets {
		p[workload.Key(target.WorkloadInfo.Obj)] = target.WorkloadInfo
	}
}
