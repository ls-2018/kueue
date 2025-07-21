package scheduler

import (
	"github.com/go-logr/logr"
	"k8s.io/klog/v2"

	"sigs.k8s.io/kueue/pkg/cache"
	"sigs.k8s.io/kueue/pkg/scheduler/preemption"
	"sigs.k8s.io/kueue/pkg/util/slices"
)

func logAdmissionAttemptIfVerbose(log logr.Logger, e *entry) {
	logV := log.V(3)
	if !logV.Enabled() {
		return
	}
	args := []any{
		"workload", klog.KObj(e.Obj),
		"clusterQueue", klog.KRef("", string(e.ClusterQueue)),
		"status", e.status,
		"reason", e.inadmissibleMsg,
	}
	if log.V(4).Enabled() {
		args = append(args, "nominatedAssignment", e.assignment)
		args = append(args, "preempted", getWorkloadReferences(e.preemptionTargets))
	}
	logV.Info("Workload evaluated for admission", args...)
}

func logSnapshotIfVerbose(log logr.Logger, s *cache.Snapshot) {
	if logV := log.V(6); logV.Enabled() {
		s.Log(logV)
	}
}

func getWorkloadReferences(targets []*preemption.Target) []klog.ObjectRef {
	return slices.Map(targets, func(t **preemption.Target) klog.ObjectRef { return klog.KObj((*t).WorkloadInfo.Obj) })
}
