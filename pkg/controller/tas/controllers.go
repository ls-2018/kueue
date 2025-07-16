package tas

import (
	ctrl "sigs.k8s.io/controller-runtime"

	configapi "sigs.k8s.io/kueue/apis/config/v1beta1"
	"sigs.k8s.io/kueue/pkg/cache"
	"sigs.k8s.io/kueue/pkg/features"
	"sigs.k8s.io/kueue/pkg/queue"
)

func SetupControllers(mgr ctrl.Manager, queues *queue.Manager, cache *cache.Cache, cfg *configapi.Configuration) (string, error) {
	recorder := mgr.GetEventRecorderFor(TASResourceFlavorController)
	topologyRec := newTopologyReconciler(mgr.GetClient(), queues, cache)
	if ctrlName, err := topologyRec.setupWithManager(mgr, cfg); err != nil {
		return ctrlName, err
	}
	rfRec := newRfReconciler(mgr.GetClient(), queues, cache, recorder)
	if ctrlName, err := rfRec.setupWithManager(mgr, cache, cfg); err != nil {
		return ctrlName, err
	}
	topologyUngater := newTopologyUngater(mgr.GetClient())
	if ctrlName, err := topologyUngater.setupWithManager(mgr, cfg); err != nil {
		return ctrlName, err
	}
	if features.Enabled(features.TASFailedNodeReplacement) {
		nodeFailureReconciler := newNodeFailureReconciler(mgr.GetClient(), recorder)
		if ctrlName, err := nodeFailureReconciler.SetupWithManager(mgr, cfg); err != nil {
			return ctrlName, err
		}
	}
	return "", nil
}
