package cache

import (
	"sync"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/kueue/pkg/over_queue"
	"sigs.k8s.io/kueue/pkg/over_resources"
)

// LocalQueue 结构体用于描述本地队列。
// GetAdmittedUsage 获取已接收的资源使用量。
// updateAdmittedUsage 更新已接收的资源使用量。
type LocalQueue struct {
	sync.RWMutex
	key                over_queue.LocalQueueReference
	reservingWorkloads int
	admittedWorkloads  int
	totalReserved      over_resources.FlavorResourceQuantities
	admittedUsage      over_resources.FlavorResourceQuantities
}

func (lq *LocalQueue) GetAdmittedUsage() corev1.ResourceList {
	lq.RLock()
	defer lq.RUnlock()
	return lq.admittedUsage.FlattenFlavors().ToResourceList()
}

func (lq *LocalQueue) updateAdmittedUsage(usage over_resources.FlavorResourceQuantities, op usageOp) {
	lq.Lock()
	defer lq.Unlock()
	updateFlavorUsage(usage, lq.admittedUsage, op)
}
