package queue

import (
	"context"
	"sort"
	"sync"

	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	config "sigs.k8s.io/kueue/apis/config/v1beta1"
	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/features"
	"sigs.k8s.io/kueue/pkg/hierarchy"
	"sigs.k8s.io/kueue/pkg/util/heap"
	utilpriority "sigs.k8s.io/kueue/pkg/util/priority"
	"sigs.k8s.io/kueue/pkg/workload"
)

type RequeueReason string

const (
	RequeueReasonFailedAfterNomination RequeueReason = "FailedAfterNomination"
	RequeueReasonNamespaceMismatch     RequeueReason = "NamespaceMismatch"
	RequeueReasonGeneric               RequeueReason = ""
	RequeueReasonPendingPreemption     RequeueReason = "PendingPreemption"
)

var (
	realClock = clock.RealClock{}
)

type ClusterQueue struct {
	hierarchy.ClusterQueue[*cohort]
	name              kueue.ClusterQueueReference
	heap              heap.Heap[workload.Info] // 即将处理的
	namespaceSelector labels.Selector
	active            bool

	inadmissibleWorkloads map[string]*workload.Info // 是至少尝试过一次但无法被接纳的工作负载

	// popCycle 标识最后一次调用Pop的周期。调用Pop时会递增。
	// popCycle 和 queueInadmissibleCycle 用于跟踪在工作负载被调度时
	// 是否有不可接纳工作负载的重新排队。
	popCycle int64

	// inflight 表示调度器最后弹出的工作负载
	inflight *workload.Info

	// queueInadmissibleCycle 存储调用 QueueInadmissibleWorkloads 时的 popId
	queueInadmissibleCycle int64

	lessFunc func(a, b *workload.Info) bool

	queueingStrategy kueue.QueueingStrategy

	rwm sync.RWMutex

	clock clock.Clock
}

func (c *ClusterQueue) GetName() kueue.ClusterQueueReference {
	return c.name
}

func workloadKey(i *workload.Info) string {
	return workload.Key(i.Obj)
}

func afsResourceWeights(cq *kueue.ClusterQueue, afsConfig *config.AdmissionFairSharing) (bool, map[corev1.ResourceName]float64) {
	enableAdmissionFs, fsResWeights := false, make(map[corev1.ResourceName]float64)
	if afsConfig != nil && cq.Spec.AdmissionScope != nil && cq.Spec.AdmissionScope.AdmissionMode == kueue.UsageBasedAdmissionFairSharing && features.Enabled(features.AdmissionFairSharing) {
		enableAdmissionFs = true
		fsResWeights = afsConfig.ResourceWeights
	}
	return enableAdmissionFs, fsResWeights
}

// AddFromLocalQueue 将此队列所属的所有工作负载推送到 ClusterQueue。
// 如果至少添加了一个工作负载，返回 true，否则返回 false。
func (c *ClusterQueue) AddFromLocalQueue(q *LocalQueue) bool {
	c.rwm.Lock()
	defer c.rwm.Unlock()
	added := false
	for _, info := range q.items {
		if c.heap.PushIfNotPresent(info) {
			added = true
		}
	}
	return added
}

func (c *ClusterQueue) Heapify(lqName string) {
	c.rwm.Lock()
	defer c.rwm.Unlock()
	for _, wl := range c.heap.List() {
		if string(wl.Obj.Spec.QueueName) == lqName {
			c.heap.PushOrUpdate(wl)
		}
	}
}

// backoffWaitingTimeExpired 如果当前时间在 requeueAt 之后且 Requeued 条件不存在或等于 True，则返回 true。
func (c *ClusterQueue) backoffWaitingTimeExpired(wInfo *workload.Info) bool {
	if apimeta.IsStatusConditionFalse(wInfo.Obj.Status.Conditions, kueue.WorkloadRequeued) {
		return false
	}
	if wInfo.Obj.Status.RequeueState == nil || wInfo.Obj.Status.RequeueState.RequeueAt == nil {
		return true
	}
	// 需要使用 "Equal" 函数验证 requeueAt
	// 因为 "After" 函数会评估纳秒，而 metav1.Time 是秒级精度。
	return c.clock.Now().After(wInfo.Obj.Status.RequeueState.RequeueAt.Time) ||
		c.clock.Now().Equal(wInfo.Obj.Status.RequeueState.RequeueAt.Time)
}

// Delete 从 ClusterQueue 中删除工作负载。
func (c *ClusterQueue) Delete(w *kueue.Workload) {
	c.rwm.Lock()
	defer c.rwm.Unlock()
	c.delete(w)
}

// delete 在没有锁的情况下从 ClusterQueue 中删除工作负载。
func (c *ClusterQueue) delete(w *kueue.Workload) {
	key := workload.Key(w)
	delete(c.inadmissibleWorkloads, key)
	c.heap.Delete(key)
	c.forgetInflightByKey(key)
}

// DeleteFromLocalQueue 从此队列中删除属于此队列的所有工作负载
// 从 ClusterQueue 中删除。
func (c *ClusterQueue) DeleteFromLocalQueue(q *LocalQueue) {
	c.rwm.Lock()
	defer c.rwm.Unlock()
	for _, w := range q.items {
		key := workload.Key(w.Obj)
		if wl := c.inadmissibleWorkloads[key]; wl != nil {
			delete(c.inadmissibleWorkloads, key)
		}
	}
	for _, w := range q.items {
		c.delete(w.Obj)
	}
}

// requeueIfNotPresent 将无法接纳的工作负载插入到 ClusterQueue 中，
// 除非它已经在队列中。如果 immediate 为 true 或者在调用 Pop 后调用了 QueueInadmissibleWorkloads，
// 工作负载将直接推回到堆中。否则，工作负载将被放入 inadmissibleWorkloads。
func (c *ClusterQueue) requeueIfNotPresent(wInfo *workload.Info, immediate bool) bool {
	c.rwm.Lock()
	defer c.rwm.Unlock()
	key := workload.Key(wInfo.Obj)
	c.forgetInflightByKey(key)
	if c.backoffWaitingTimeExpired(wInfo) && (immediate || c.queueInadmissibleCycle >= c.popCycle || wInfo.LastAssignment.PendingFlavors()) {
		// 如果工作负载不可接纳，将其移回队列中。
		inadmissibleWl := c.inadmissibleWorkloads[key]
		if inadmissibleWl != nil {
			wInfo = inadmissibleWl
			delete(c.inadmissibleWorkloads, key)
		}
		return c.heap.PushIfNotPresent(wInfo)
	}

	if c.inadmissibleWorkloads[key] != nil {
		return false
	}

	if data := c.heap.GetByKey(key); data != nil {
		return false
	}

	c.inadmissibleWorkloads[key] = wInfo

	return true
}

func (c *ClusterQueue) forgetInflightByKey(key string) {
	if c.inflight != nil && workload.Key(c.inflight.Obj) == key {
		c.inflight = nil
	}
}

// Pending 返回待处理工作负载的总数。
func (c *ClusterQueue) Pending() int {
	c.rwm.RLock()
	defer c.rwm.RUnlock()
	return c.PendingActive() + c.PendingInadmissible()
}

// PendingActive 返回活跃待处理工作负载的数量，
// 即在接纳队列中的工作负载。
func (c *ClusterQueue) PendingActive() int {
	result := c.heap.Len()
	if c.inflight != nil {
		result++
	}
	return result
}

// PendingInadmissible 返回不可接纳待处理工作负载的数量，
// 即已经尝试过并等待集群条件改变以潜在地变为可接纳的工作负载。
func (c *ClusterQueue) PendingInadmissible() int {
	return len(c.inadmissibleWorkloads)
}

// Pop 移除队列头部并返回它。如果队列为空，返回 nil。
func (c *ClusterQueue) Pop() *workload.Info {
	c.rwm.Lock()
	defer c.rwm.Unlock()
	c.popCycle++
	if c.heap.Len() == 0 {
		c.inflight = nil
		return nil
	}
	c.inflight = c.heap.Pop()
	return c.inflight
}

// Dump 生成此 ClusterQueue 堆中当前工作负载的转储。
// 如果队列为空，返回 false，否则返回 true。
func (c *ClusterQueue) Dump() ([]string, bool) {
	c.rwm.RLock()
	defer c.rwm.RUnlock()
	if c.heap.Len() == 0 {
		return nil, false
	}
	elements := make([]string, c.heap.Len())
	for i, info := range c.heap.List() {
		elements[i] = workload.Key(info.Obj)
	}
	return elements, true
}

func (c *ClusterQueue) DumpInadmissible() ([]string, bool) {
	c.rwm.RLock()
	defer c.rwm.RUnlock()
	if len(c.inadmissibleWorkloads) == 0 {
		return nil, false
	}
	elements := make([]string, 0, len(c.inadmissibleWorkloads))
	for _, info := range c.inadmissibleWorkloads {
		elements = append(elements, workload.Key(info.Obj))
	}
	return elements, true
}

// Snapshot 返回此 ClusterQueue 堆中当前工作负载的副本。
func (c *ClusterQueue) Snapshot() []*workload.Info {
	elements := c.totalElements()
	sort.Slice(elements, func(i, j int) bool {
		return c.lessFunc(elements[i], elements[j])
	})
	return elements
}

// Info 返回工作负载键的 workload.Info。
// 此方法的用户不应修改返回的对象。
func (c *ClusterQueue) Info(key string) *workload.Info {
	c.rwm.RLock()
	defer c.rwm.RUnlock()
	return c.heap.GetByKey(key)
}

func (c *ClusterQueue) totalElements() []*workload.Info {
	c.rwm.RLock()
	defer c.rwm.RUnlock()
	totalLen := c.heap.Len() + len(c.inadmissibleWorkloads)
	elements := make([]*workload.Info, 0, totalLen)
	elements = append(elements, c.heap.List()...)
	for _, e := range c.inadmissibleWorkloads {
		elements = append(elements, e)
	}
	if c.inflight != nil {
		elements = append(elements, c.inflight)
	}
	return elements
}

// Active 如果队列处于活跃状态，返回 true
func (c *ClusterQueue) Active() bool {
	c.rwm.RLock()
	defer c.rwm.RUnlock()
	return c.active
}

// RequeueIfNotPresent 将未被接纳的工作负载重新插入到 ClusterQueue 中。
// 如果布尔值为 true，工作负载应该立即放回队列中，
// 因为我们在上一个周期中无法确定工作负载是否可接纳。
// 如果布尔值为 false，实现可能会选择将其保留在临时占位符阶段，
// 在那里它不会与其他工作负载竞争，直到集群事件释放配额。
// 如果工作负载已经在 ClusterQueue 中，则不应重新插入。
// 如果工作负载被插入，返回 true。
func (c *ClusterQueue) RequeueIfNotPresent(wInfo *workload.Info, reason RequeueReason) bool {
	if c.queueingStrategy == kueue.StrictFIFO {
		return c.requeueIfNotPresent(wInfo, reason != RequeueReasonNamespaceMismatch)
	}
	return c.requeueIfNotPresent(wInfo, reason == RequeueReasonFailedAfterNomination || reason == RequeueReasonPendingPreemption)
}

// queueOrderingFunc 返回由 clusterQueue 堆算法使用的函数
// 来排序工作负载。该函数基于工作负载的优先级进行排序。
// 当优先级相等时，它使用工作负载的创建或驱逐时间。
func queueOrderingFunc(ctx context.Context, c client.Client, wo workload.Ordering, fsResWeights map[corev1.ResourceName]float64, enableAdmissionFs bool) func(a, b *workload.Info) bool {
	log := ctrl.LoggerFrom(ctx)
	return func(a, b *workload.Info) bool {
		if enableAdmissionFs {
			lqAUsage, errA := a.LocalQueueUsage(ctx, c, fsResWeights)
			lqBUsage, errB := b.LocalQueueUsage(ctx, c, fsResWeights)
			switch {
			case errA != nil:
				log.V(2).Error(errA, "Error determining LocalQueue usage")
			case errB != nil:
				log.V(2).Error(errB, "Error determining LocalQueue usage")
			default:
				log.V(3).Info("Resource usage from LocalQueue", "LocalQueue", a.Obj.Spec.QueueName, "Usage", lqAUsage)
				log.V(3).Info("Resource usage from LocalQueue", "LocalQueue", b.Obj.Spec.QueueName, "Usage", lqBUsage)
				if lqAUsage != lqBUsage {
					return lqAUsage < lqBUsage
				}
			}
		}
		p1 := utilpriority.Priority(a.Obj)
		p2 := utilpriority.Priority(b.Obj)

		if p1 != p2 {
			return p1 > p2
		}

		tA := wo.GetQueueOrderTimestamp(a.Obj)
		tB := wo.GetQueueOrderTimestamp(b.Obj)
		return !tB.Before(tA)
	}
}

// QueueInadmissibleWorkloads 将所有工作负载从 inadmissibleWorkloads 移动到堆中。
// 如果至少移动了一个工作负载，返回 true，否则返回 false。
func (c *ClusterQueue) QueueInadmissibleWorkloads(ctx context.Context, client client.Client) bool {
	c.rwm.Lock()
	defer c.rwm.Unlock()
	c.queueInadmissibleCycle = c.popCycle
	if len(c.inadmissibleWorkloads) == 0 {
		return false
	}

	inadmissibleWorkloads := make(map[string]*workload.Info)
	moved := false
	for key, wInfo := range c.inadmissibleWorkloads {
		ns := corev1.Namespace{}
		err := client.Get(ctx, types.NamespacedName{Name: wInfo.Obj.Namespace}, &ns)
		if err != nil || !c.namespaceSelector.Matches(labels.Set(ns.Labels)) || !c.backoffWaitingTimeExpired(wInfo) {
			inadmissibleWorkloads[key] = wInfo
		} else {
			moved = c.heap.PushIfNotPresent(wInfo) || moved
		}
	}

	c.inadmissibleWorkloads = inadmissibleWorkloads
	return moved
}

func newClusterQueue(ctx context.Context, client client.Client, cq *kueue.ClusterQueue, wo workload.Ordering, afsConfig *config.AdmissionFairSharing) (*ClusterQueue, error) {
	enableAdmissionFs, fsResWeights := afsResourceWeights(cq, afsConfig)
	cqImpl := newClusterQueueImpl(ctx, client, wo, realClock, fsResWeights, enableAdmissionFs)
	err := cqImpl.Update(cq) // ✅
	if err != nil {
		return nil, err
	}
	return cqImpl, nil
}

func newClusterQueueImpl(ctx context.Context, client client.Client, wo workload.Ordering, clock clock.Clock, fsResWeights map[corev1.ResourceName]float64, enableAdmissionFs bool) *ClusterQueue {
	lessFunc := queueOrderingFunc(ctx, client, wo, fsResWeights, enableAdmissionFs)
	return &ClusterQueue{
		heap:                   *heap.New(workloadKey, lessFunc),
		inadmissibleWorkloads:  make(map[string]*workload.Info),
		queueInadmissibleCycle: -1,
		lessFunc:               lessFunc,
		rwm:                    sync.RWMutex{},
		clock:                  clock,
	}
}

// Update 更新此 ClusterQueue 的属性
func (c *ClusterQueue) Update(apiCQ *kueue.ClusterQueue) error {
	c.rwm.Lock()
	defer c.rwm.Unlock()
	c.name = kueue.ClusterQueueReference(apiCQ.Name)
	c.queueingStrategy = apiCQ.Spec.QueueingStrategy
	nsSelector, err := metav1.LabelSelectorAsSelector(apiCQ.Spec.NamespaceSelector)
	if err != nil {
		return err
	}
	c.namespaceSelector = nsSelector
	c.active = apimeta.IsStatusConditionTrue(apiCQ.Status.Conditions, kueue.ClusterQueueActive)
	return nil
}

// PushOrUpdate 将工作负载推送到 ClusterQueue。
// 如果工作负载已存在，则用新的更新它。
func (c *ClusterQueue) PushOrUpdate(wInfo *workload.Info) {
	c.rwm.Lock()
	defer c.rwm.Unlock()
	key := workload.Key(wInfo.Obj)
	c.forgetInflightByKey(key)
	oldInfo := c.inadmissibleWorkloads[key]
	if oldInfo != nil {
		// 如果工作负载不可接纳且没有改变，则就地更新
		// 以潜在地变为可接纳，除非 Eviction 状态发生变化
		// 这会影响队列中工作负载的顺序。
		if equality.Semantic.DeepEqual(oldInfo.Obj.Spec, wInfo.Obj.Spec) &&
			equality.Semantic.DeepEqual(oldInfo.Obj.Status.ReclaimablePods, wInfo.Obj.Status.ReclaimablePods) &&
			equality.Semantic.DeepEqual(apimeta.FindStatusCondition(oldInfo.Obj.Status.Conditions, kueue.WorkloadEvicted),
				apimeta.FindStatusCondition(wInfo.Obj.Status.Conditions, kueue.WorkloadEvicted)) &&
			equality.Semantic.DeepEqual(apimeta.FindStatusCondition(oldInfo.Obj.Status.Conditions, kueue.WorkloadRequeued),
				apimeta.FindStatusCondition(wInfo.Obj.Status.Conditions, kueue.WorkloadRequeued)) {
			c.inadmissibleWorkloads[key] = wInfo
			return
		}
		// 否则在队列中移动或就地更新。
		delete(c.inadmissibleWorkloads, key)
	}
	if c.heap.GetByKey(key) == nil && !c.backoffWaitingTimeExpired(wInfo) {
		c.inadmissibleWorkloads[key] = wInfo
		return
	}
	c.heap.PushOrUpdate(wInfo)
}
