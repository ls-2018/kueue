package cache

import (
	"sync"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/kueue/pkg/queue"
	"sigs.k8s.io/kueue/pkg/resources"
)

// LocalQueue 结构体用于描述本地队列。
// GetAdmittedUsage 获取已接收的资源使用量。
// updateAdmittedUsage 更新已接收的资源使用量。
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
