package v1beta1

import (
	"k8s.io/apiserver/pkg/registry/rest"

	"sigs.k8s.io/kueue/pkg/queue"
)

func NewStorage(mgr *queue.Manager) map[string]rest.Storage {
	return map[string]rest.Storage{
		"clusterqueues":                  NewCqREST(),
		"clusterqueues/pendingworkloads": NewPendingWorkloadsInCqREST(mgr),
		"localqueues":                    NewLqREST(),
		"localqueues/pendingworkloads":   NewPendingWorkloadsInLqREST(mgr),
	}
}
