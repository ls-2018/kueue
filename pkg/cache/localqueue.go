package cache

import (
	"sync"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/kueue/pkg/queue"
	"sigs.k8s.io/kueue/pkg/resources"
)

type LocalQueue struct {
	sync.RWMutex
	key                queue.LocalQueueReference
	reservingWorkloads int
	admittedWorkloads  int
	totalReserved      resources.FlavorResourceQuantities
	admittedUsage      resources.FlavorResourceQuantities
}

func (lq *LocalQueue) GetAdmittedUsage() corev1.ResourceList {
	lq.RLock()
	defer lq.RUnlock()
	return lq.admittedUsage.FlattenFlavors().ToResourceList()
}

func (lq *LocalQueue) updateAdmittedUsage(usage resources.FlavorResourceQuantities, op usageOp) {
	lq.Lock()
	defer lq.Unlock()
	updateFlavorUsage(usage, lq.admittedUsage, op)
}
