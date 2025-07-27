package queue

import (
	"context"
	"errors"
	"fmt"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	config "sigs.k8s.io/kueue/apis/config/v1beta1"
	kueuealpha "sigs.k8s.io/kueue/apis/kueue/v1alpha1"
	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	utilindexer "sigs.k8s.io/kueue/pkg/controller/core/indexer"
	"sigs.k8s.io/kueue/pkg/features"
	"sigs.k8s.io/kueue/pkg/hierarchy"
	"sigs.k8s.io/kueue/pkg/metrics"
	"sigs.k8s.io/kueue/pkg/workload"
)

var (
	ErrLocalQueueDoesNotExistOrInactive = errors.New("localQueue doesn't exist or inactive")
	ErrClusterQueueDoesNotExist         = errors.New("clusterQueue doesn't exist")
	errClusterQueueAlreadyExists        = errors.New("clusterQueue already exists")
)

type options struct {
	podsReadyRequeuingTimestamp config.RequeuingTimestamp
	workloadInfoOptions         []workload.InfoOption
	admissionFairSharing        *config.AdmissionFairSharing
	clock                       clock.WithDelayedExecution
}

// Option 配置管理器
type Option func(*options)

var defaultOptions = options{
	podsReadyRequeuingTimestamp: config.EvictionTimestamp,
	workloadInfoOptions:         []workload.InfoOption{},
	clock:                       clock.RealClock{},
}

// WithClock 允许指定自定义时钟

func WithAdmissionFairSharing(cfg *config.AdmissionFairSharing) Option {
	return func(o *options) {
		if features.Enabled(features.AdmissionFairSharing) {
			o.admissionFairSharing = cfg
		}
	}
}

// WithPodsReadyRequeuingTimestamp 设置用于排序因 PodsReady 条件而重新排队的工作负载的时间戳
func WithPodsReadyRequeuingTimestamp(ts config.RequeuingTimestamp) Option {
	return func(o *options) {
		o.podsReadyRequeuingTimestamp = ts
	}
}

// WithExcludedResourcePrefixes 设置排除的资源前缀列表
func WithExcludedResourcePrefixes(excludedPrefixes []string) Option {
	return func(o *options) {
		o.workloadInfoOptions = append(o.workloadInfoOptions, workload.WithExcludedResourcePrefixes(excludedPrefixes))
	}
}

// WithResourceTransformations 设置资源转换
func WithResourceTransformations(transforms []config.ResourceTransformation) Option {
	return func(o *options) {
		o.workloadInfoOptions = append(o.workloadInfoOptions, workload.WithResourceTransformations(transforms))
	}
}

type TopologyUpdateWatcher interface {
	NotifyTopologyUpdate(oldTopology, newTopology *kueuealpha.Topology)
}

type Manager struct {
	sync.RWMutex
	cond sync.Cond

	clock         clock.WithDelayedExecution
	client        client.Client
	statusChecker StatusChecker
	localQueues   map[LocalQueueReference]*LocalQueue

	snapshotsMutex sync.RWMutex
	snapshots      map[kueue.ClusterQueueReference][]kueue.ClusterQueuePendingWorkload

	workloadOrdering workload.Ordering

	workloadInfoOptions []workload.InfoOption

	hm hierarchy.Manager[*ClusterQueue, *cohort]

	topologyUpdateWatchers []TopologyUpdateWatcher

	admissionFairSharingConfig *config.AdmissionFairSharing
	secondPassQueue            *secondPassQueue
}

func NewManager(client client.Client, checker StatusChecker, opts ...Option) *Manager {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	m := &Manager{
		clock:          options.clock,
		client:         client,
		statusChecker:  checker,
		localQueues:    make(map[LocalQueueReference]*LocalQueue),
		snapshotsMutex: sync.RWMutex{},
		snapshots:      make(map[kueue.ClusterQueueReference][]kueue.ClusterQueuePendingWorkload, 0),
		workloadOrdering: workload.Ordering{
			PodsReadyRequeuingTimestamp: options.podsReadyRequeuingTimestamp,
		},
		workloadInfoOptions: options.workloadInfoOptions,
		hm:                  hierarchy.NewManager[*ClusterQueue, *cohort](newCohort),

		topologyUpdateWatchers:     make([]TopologyUpdateWatcher, 0),
		admissionFairSharingConfig: options.admissionFairSharing,
		secondPassQueue:            newSecondPassQueue(),
	}
	m.cond.L = &m.RWMutex
	return m
}

func (m *Manager) AddTopologyUpdateWatcher(watcher TopologyUpdateWatcher) {
	m.topologyUpdateWatchers = append(m.topologyUpdateWatchers, watcher)
}

func (m *Manager) NotifyTopologyUpdateWatchers(oldTopology, newTopology *kueuealpha.Topology) {
	for _, watcher := range m.topologyUpdateWatchers {
		watcher.NotifyTopologyUpdate(oldTopology, newTopology)
	}
}

func (m *Manager) HeapifyClusterQueue(cq *kueue.ClusterQueue, lqName string) error {
	m.Lock()
	defer m.Unlock()
	cqImpl := m.hm.ClusterQueue(kueue.ClusterQueueReference(cq.Name))
	if cqImpl == nil {
		return ErrClusterQueueDoesNotExist
	}
	cqImpl.Heapify(lqName)
	return nil
}

func (m *Manager) DeleteClusterQueue(cq *kueue.ClusterQueue) {
	m.Lock()
	defer m.Unlock()
	cqImpl := m.hm.ClusterQueue(kueue.ClusterQueueReference(cq.Name))
	if cqImpl == nil {
		return
	}
	m.hm.DeleteClusterQueue(kueue.ClusterQueueReference(cq.Name))
	metrics.ClearClusterQueueMetrics(cq.Name)
}

func (m *Manager) AddLocalQueue(ctx context.Context, q *kueue.LocalQueue) error {
	m.Lock()
	defer m.Unlock()

	key := Key(q)
	if _, ok := m.localQueues[key]; ok {
		return fmt.Errorf("queue %q already exists", q.Name)
	}
	qImpl := newLocalQueue(q)
	m.localQueues[key] = qImpl
	// 遍历现有工作负载，因为对应此队列的工作负载可能已经提前添加
	var workloads kueue.WorkloadList
	if err := m.client.List(ctx, &workloads, client.MatchingFields{utilindexer.WorkloadQueueKey: q.Name}, client.InNamespace(q.Namespace)); err != nil {
		return fmt.Errorf("listing workloads that match the queue: %w", err)
	}
	for _, w := range workloads.Items {
		if !workload.IsActive(&w) || workload.HasQuotaReservation(&w) {
			continue
		}
		workload.AdjustResources(ctx, m.client, &w)
		qImpl.AddOrUpdate(workload.NewInfo(&w, m.workloadInfoOptions...))
	}
	cq := m.hm.ClusterQueue(qImpl.ClusterQueue)
	if cq != nil && cq.AddFromLocalQueue(qImpl) {
		m.Broadcast()
	}
	return nil
}

func (m *Manager) UpdateLocalQueue(q *kueue.LocalQueue) error {
	m.Lock()
	defer m.Unlock()
	qImpl, ok := m.localQueues[Key(q)]
	if !ok {
		return ErrLocalQueueDoesNotExistOrInactive
	}
	if qImpl.ClusterQueue != q.Spec.ClusterQueue {
		oldCQ := m.hm.ClusterQueue(qImpl.ClusterQueue)
		if oldCQ != nil {
			oldCQ.DeleteFromLocalQueue(qImpl)
		}
		newCQ := m.hm.ClusterQueue(q.Spec.ClusterQueue)
		if newCQ != nil && newCQ.AddFromLocalQueue(qImpl) {
			m.Broadcast()
		}
	}
	qImpl.update(q)
	return nil
}

func (m *Manager) DeleteLocalQueue(q *kueue.LocalQueue) {
	m.Lock()
	defer m.Unlock()
	key := Key(q)
	qImpl := m.localQueues[key]
	if qImpl == nil {
		return
	}
	cq := m.hm.ClusterQueue(qImpl.ClusterQueue)
	if cq != nil {
		cq.DeleteFromLocalQueue(qImpl)
	}
	if features.Enabled(features.LocalQueueMetrics) {
		namespace, lqName := MustParseLocalQueueReference(key)
		metrics.ClearLocalQueueMetrics(metrics.LocalQueueReference{
			Name:      lqName,
			Namespace: namespace,
		})
	}
	delete(m.localQueues, key)
}

func (m *Manager) PendingWorkloads(q *kueue.LocalQueue) (int32, error) {
	m.RLock()
	defer m.RUnlock()

	qImpl, ok := m.localQueues[Key(q)]
	if !ok {
		return 0, ErrLocalQueueDoesNotExistOrInactive
	}

	return int32(len(qImpl.items)), nil
}

func (m *Manager) Pending(cq *kueue.ClusterQueue) (int, error) {
	m.RLock()
	defer m.RUnlock()

	cqImpl := m.hm.ClusterQueue(kueue.ClusterQueueReference(cq.Name))
	if cqImpl == nil {
		return 0, ErrClusterQueueDoesNotExist
	}

	return cqImpl.Pending(), nil
}

func (m *Manager) deleteWorkloadFromQueueAndClusterQueue(w *kueue.Workload, qKey LocalQueueReference) {
	q := m.localQueues[qKey]
	if q == nil {
		return
	}
	delete(q.items, workload.Key(w))
	cq := m.hm.ClusterQueue(q.ClusterQueue)
	if cq != nil {
		cq.Delete(w)
		m.reportPendingWorkloads(q.ClusterQueue, cq)
	}
	if features.Enabled(features.LocalQueueMetrics) {
		m.reportLQPendingWorkloads(q)
	}
}

// QueueAssociatedInadmissibleWorkloadsAfter 将同一集群队列和队列组（如果存在）中
// 之前不可接纳的工作负载重新排队到堆中，这些工作负载与提供的已接纳工作负载相关
// 可以在函数开始时执行可选操作，在持有锁的同时，以提供与队列中操作的原子性
func (m *Manager) QueueAssociatedInadmissibleWorkloadsAfter(ctx context.Context, w *kueue.Workload, action func()) {
	m.Lock()
	defer m.Unlock()
	if action != nil {
		action()
	}

	q := m.localQueues[KeyFromWorkload(w)]
	if q == nil {
		return
	}
	cq := m.hm.ClusterQueue(q.ClusterQueue)
	if cq == nil {
		return
	}

	if m.requeueWorkloadsCQ(ctx, cq) {
		m.Broadcast()
	}
}

// CleanUpOnContext 跟踪上下文。当关闭时，它会唤醒等待元素可用的例程
// 应该在调用 Heads 之前调用
func (m *Manager) CleanUpOnContext(ctx context.Context) {
	<-ctx.Done()
	m.Broadcast()
}

// Heads 返回队列的头部，以及它们关联的集群队列
// 如果队列为空，它会阻塞直到有元素或上下文终止
func (m *Manager) Heads(ctx context.Context) []workload.Info {
	m.Lock()
	defer m.Unlock()
	log := ctrl.LoggerFrom(ctx)
	for {
		workloads := m.heads()
		log.V(3).Info("Obtained ClusterQueue heads", "count", len(workloads))
		if len(workloads) != 0 {
			return workloads
		}
		select {
		case <-ctx.Done():
			return nil
		default:
			m.cond.Wait()
		}
	}
}

func (m *Manager) heads() []workload.Info {
	var workloads []workload.Info
	workloads = append(workloads, m.secondPassQueue.takeAllReady()...)
	for cqName, cq := range m.hm.ClusterQueues() {
		// 缓存可能在测试中为 nil，如果缓存为 nil，我们将跳过检查
		if m.statusChecker != nil && !m.statusChecker.ClusterQueueActive(cqName) {
			continue
		}
		wl := cq.Pop()
		if wl == nil {
			continue
		}
		m.reportPendingWorkloads(cqName, cq)
		wlCopy := *wl
		wlCopy.ClusterQueue = cqName
		workloads = append(workloads, wlCopy)
		q := m.localQueues[KeyFromWorkload(wl.Obj)]
		delete(q.items, workload.Key(wl.Obj))
		if features.Enabled(features.LocalQueueMetrics) {
			m.reportLQPendingWorkloads(q)
		}
	}
	return workloads
}

func (m *Manager) reportPendingWorkloads(cqName kueue.ClusterQueueReference, cq *ClusterQueue) {
	active := cq.PendingActive()
	inadmissible := cq.PendingInadmissible()
	if m.statusChecker != nil && !m.statusChecker.ClusterQueueActive(cqName) {
		inadmissible += active
		active = 0
	}
	metrics.ReportPendingWorkloads(cqName, active, inadmissible)
}

func (m *Manager) GetClusterQueueNames() []kueue.ClusterQueueReference {
	m.RLock()
	defer m.RUnlock()
	return m.hm.ClusterQueuesNames()
}

func (m *Manager) getClusterQueue(cqName kueue.ClusterQueueReference) *ClusterQueue {
	m.RLock()
	defer m.RUnlock()
	return m.hm.ClusterQueue(cqName)
}

func (m *Manager) getClusterQueueLockless(cqName kueue.ClusterQueueReference) (val *ClusterQueue, ok bool) {
	val = m.hm.ClusterQueue(cqName)
	return val, val != nil
}

func (m *Manager) PendingWorkloadsInfo(cqName kueue.ClusterQueueReference) []*workload.Info {
	cq := m.getClusterQueue(cqName)
	if cq == nil {
		return nil
	}
	return cq.Snapshot()
}

// UpdateSnapshot 计算新的快照，如果与前一版本不同则替换
// 如果快照实际被更新，返回 true
func (m *Manager) UpdateSnapshot(cqName kueue.ClusterQueueReference, maxCount int32) bool {
	cq := m.getClusterQueue(cqName)
	if cq == nil {
		return false
	}
	newSnapshot := make([]kueue.ClusterQueuePendingWorkload, 0)
	for index, info := range cq.Snapshot() {
		if int32(index) >= maxCount {
			break
		}
		if info == nil {
			continue
		}
		newSnapshot = append(newSnapshot, kueue.ClusterQueuePendingWorkload{
			Name:      info.Obj.Name,
			Namespace: info.Obj.Namespace,
		})
	}
	prevSnapshot := m.GetSnapshot(cqName)
	if !equality.Semantic.DeepEqual(prevSnapshot, newSnapshot) {
		m.setSnapshot(cqName, newSnapshot)
		return true
	}
	return false
}

func (m *Manager) setSnapshot(cqName kueue.ClusterQueueReference, workloads []kueue.ClusterQueuePendingWorkload) {
	m.snapshotsMutex.Lock()
	defer m.snapshotsMutex.Unlock()
	m.snapshots[cqName] = workloads
}

func (m *Manager) GetSnapshot(cqName kueue.ClusterQueueReference) []kueue.ClusterQueuePendingWorkload {
	m.snapshotsMutex.RLock()
	defer m.snapshotsMutex.RUnlock()
	return m.snapshots[cqName]
}

func (m *Manager) DeleteSnapshot(cq *kueue.ClusterQueue) {
	m.snapshotsMutex.Lock()
	defer m.snapshotsMutex.Unlock()
	delete(m.snapshots, kueue.ClusterQueueReference(cq.Name))
}

// DeleteSecondPassWithoutLock 从第二遍队列中删除待处理的工作负载
func (m *Manager) DeleteSecondPassWithoutLock(w *kueue.Workload) {
	m.secondPassQueue.deleteByKey(workload.Key(w))
}

// QueueSecondPassIfNeeded 如果需要，将工作负载排队进行第二遍调度，延迟 1 秒
func (m *Manager) QueueSecondPassIfNeeded(ctx context.Context, w *kueue.Workload) bool {
	if workload.NeedsSecondPass(w) {
		log := ctrl.LoggerFrom(ctx)
		log.V(3).Info("Workload pre-queued for second pass", "workload", workload.Key(w))
		m.secondPassQueue.prequeue(w)
		m.clock.AfterFunc(time.Second, func() {
			m.queueSecondPass(ctx, w)
		})
		return true
	}
	return false
}

func (m *Manager) queueSecondPass(ctx context.Context, w *kueue.Workload) {
	m.Lock()
	defer m.Unlock()

	log := ctrl.LoggerFrom(ctx)
	wInfo := workload.NewInfo(w, m.workloadInfoOptions...)
	if m.secondPassQueue.queue(wInfo) {
		log.V(3).Info("Workload queued for second pass of scheduling", "workload", workload.Key(w))
		m.Broadcast()
	}
}

func (m *Manager) DefaultLocalQueueExist(namespace string) bool {
	m.Lock()
	defer m.Unlock()

	_, ok := m.localQueues[DefaultQueueKey(namespace)]
	return ok
}

// ClusterQueueFromLocalQueue 返回集群队列名称以及是否找到，
// 给定 QueueKey(namespace/localQueueName) 作为参数
func (m *Manager) ClusterQueueFromLocalQueue(localQueueKey LocalQueueReference) (kueue.ClusterQueueReference, bool) {
	m.RLock()
	defer m.RUnlock()
	if lq, ok := m.localQueues[localQueueKey]; ok {
		return lq.ClusterQueue, true
	}
	return "", false
}

// QueueInadmissibleWorkloads 将相应集群队列中的所有不可接纳工作负载移动到堆中
// 如果至少有一个工作负载排队，我们将广播事件
func (m *Manager) QueueInadmissibleWorkloads(ctx context.Context, cqNames sets.Set[kueue.ClusterQueueReference]) {
	m.Lock()
	defer m.Unlock()
	if len(cqNames) == 0 {
		return
	}

	var queued bool
	for name := range cqNames {
		cq := m.hm.ClusterQueue(name)
		if cq == nil {
			continue
		}
		if m.requeueWorkloadsCQ(ctx, cq) {
			queued = true
		}
	}

	if queued {
		m.Broadcast()
	}
}

// requeueWorkloadsCQ 将同一队列组中此集群队列的所有工作负载从不可接纳工作负载移动到堆中
// 如果此集群队列的队列组为空，它只移动此集群队列中的所有工作负载
// 如果至少有一个工作负载被移动，返回 true，否则返回 false
// 下面列出的事件可能使同一队列组中的工作负载变得可接纳
// 然后需要调用 requeueWorkloadsCQ
// 1. 队列组中任何已接纳工作负载的删除事件
// 2. 队列组中任何集群队列的添加事件
// 3. 队列组中任何集群队列的更新事件
// 4. 队列组的更新
//
// 警告：调用时必须持有管理器的读锁，
// 否则如果引入队列组循环，可能会遇到无限循环
func (m *Manager) requeueWorkloadsCQ(ctx context.Context, cq *ClusterQueue) bool {
	if cq.HasParent() {
		return m.requeueWorkloadsCohort(ctx, cq.Parent()) // ✅
	}
	return cq.QueueInadmissibleWorkloads(ctx, m.client) // 从不可用的 map 中 移动到 heap   ✅
}

// moveWorkloadsCohorts 检查循环，然后移动队列组树中的所有不可接纳工作负载
// 如果存在循环或没有工作负载被移动，返回 false
//
// 警告：调用时必须持有管理器的读锁，
// 否则如果引入队列组循环，可能会遇到无限循环
func (m *Manager) requeueWorkloadsCohort(ctx context.Context, cohort *cohort) bool {
	log := ctrl.LoggerFrom(ctx)

	if hierarchy.HasCycle(cohort) {
		log.V(2).Info("Attempted to move workloads from Cohort which has cycle", "cohort", cohort.GetName())
		return false
	}
	root := cohort.getRootUnsafe()
	log.V(2).Info("Attempting to move workloads", "cohort", cohort.Name, "root", root.Name)
	return requeueWorkloadsCohortSubtree(ctx, m, root) // ✅
}

func requeueWorkloadsCohortSubtree(ctx context.Context, m *Manager, cohort *cohort) bool {
	queued := false
	for _, clusterQueue := range cohort.ChildCQs() {
		queued = clusterQueue.QueueInadmissibleWorkloads(ctx, m.client) || queued
	}
	for _, childCohort := range cohort.ChildCohorts() {
		queued = requeueWorkloadsCohortSubtree(ctx, m, childCohort) || queued // ✅
	}
	return queued
}

func (m *Manager) AddOrUpdateCohort(ctx context.Context, cohort *kueuealpha.Cohort) {
	m.Lock()
	defer m.Unlock()
	cohortName := kueue.CohortReference(cohort.Name)

	m.hm.AddCohort(cohortName)
	m.hm.UpdateCohortEdge(cohortName, cohort.Spec.Parent)
	if m.requeueWorkloadsCohort(ctx, m.hm.Cohort(cohortName)) {
		m.Broadcast()
	}
}

func (m *Manager) DeleteCohort(cohortName kueue.CohortReference) {
	m.Lock()
	defer m.Unlock()
	m.hm.DeleteCohort(cohortName)
}

func (m *Manager) AddClusterQueue(ctx context.Context, cq *kueue.ClusterQueue) error {
	m.Lock()
	defer m.Unlock()

	if cq := m.hm.ClusterQueue(kueue.ClusterQueueReference(cq.Name)); cq != nil {
		return errClusterQueueAlreadyExists
	}

	cqImpl, err := newClusterQueue(ctx, m.client, cq, m.workloadOrdering, m.admissionFairSharingConfig)
	if err != nil {
		return err
	}
	m.hm.AddClusterQueue(cqImpl)
	m.hm.UpdateClusterQueueEdge(kueue.ClusterQueueReference(cq.Name), cq.Spec.Cohort) // ✅

	// 遍历现有队列，因为对应此集群队列的队列可能已经提前添加
	var queues kueue.LocalQueueList
	if err := m.client.List(ctx, &queues, client.MatchingFields{utilindexer.QueueClusterQueueKey: cq.Name}); err != nil {
		return fmt.Errorf("listing queues pointing to the cluster queue: %w", err)
	}
	addedWorkloads := false
	for _, q := range queues.Items {
		qImpl := m.localQueues[Key(&q)]
		if qImpl != nil {
			added := cqImpl.AddFromLocalQueue(qImpl)
			addedWorkloads = addedWorkloads || added
		}
	}

	queued := m.requeueWorkloadsCQ(ctx, cqImpl) // ✅
	m.reportPendingWorkloads(kueue.ClusterQueueReference(cq.Name), cqImpl)

	// 需要在这里再次遍历，以防 requeueWorkloadsCQ 添加了不可接纳的工作负载
	if features.Enabled(features.LocalQueueMetrics) {
		for _, q := range queues.Items {
			qImpl := m.localQueues[Key(&q)]
			if qImpl != nil {
				m.reportLQPendingWorkloads(qImpl)
			}
		}
	}

	if queued || addedWorkloads {
		m.Broadcast()
	}
	return nil
}

func (m *Manager) UpdateClusterQueue(ctx context.Context, cq *kueue.ClusterQueue, specUpdated bool) error {
	m.Lock()
	defer m.Unlock()
	cqName := kueue.ClusterQueueReference(cq.Name)

	cqImpl := m.hm.ClusterQueue(cqName)
	if cqImpl == nil {
		return ErrClusterQueueDoesNotExist
	}

	oldActive := cqImpl.Active()
	// TODO(#8): 基于队列策略的变化重新创建堆
	if err := cqImpl.Update(cq); err != nil {
		return err
	}
	m.hm.UpdateClusterQueueEdge(cqName, cq.Spec.Cohort)

	// TODO(#8): 根据确切事件选择性地移动工作负载
	// 如果有任何工作负载变得可接纳或队列变为活跃状态
	if (specUpdated && m.requeueWorkloadsCQ(ctx, cqImpl)) || (!oldActive && cqImpl.Active()) {
		m.reportPendingWorkloads(cqName, cqImpl)
		if features.Enabled(features.LocalQueueMetrics) {
			for _, q := range m.localQueues {
				if q.ClusterQueue == cqName {
					m.reportLQPendingWorkloads(q)
				}
			}
		}
		m.Broadcast()
	}
	return nil
}

// AddOrUpdateWorkload 添加或更新工作负载到相应的队列
// 返回队列是否存在
func (m *Manager) AddOrUpdateWorkload(w *kueue.Workload) error {
	m.Lock()
	defer m.Unlock()
	return m.AddOrUpdateWorkloadWithoutLock(w) // ✅
}

func (m *Manager) AddOrUpdateWorkloadWithoutLock(w *kueue.Workload) error {
	qKey := KeyFromWorkload(w)
	q := m.localQueues[qKey]
	if q == nil {
		return ErrLocalQueueDoesNotExistOrInactive
	}
	wInfo := workload.NewInfo(w, m.workloadInfoOptions...) // ✅
	q.AddOrUpdate(wInfo)                                   // ✅
	cq := m.hm.ClusterQueue(q.ClusterQueue)
	if cq == nil {
		return ErrClusterQueueDoesNotExist
	}
	cq.PushOrUpdate(wInfo) // ✅
	if features.Enabled(features.LocalQueueMetrics) {
		m.reportLQPendingWorkloads(q)
	}
	m.reportPendingWorkloads(q.ClusterQueue, cq)
	m.Broadcast()
	return nil
}

func (m *Manager) reportLQPendingWorkloads(lq *LocalQueue) {
	active := m.PendingActiveInLocalQueue(lq)
	inadmissible := m.PendingInadmissibleInLocalQueue(lq)
	if m.statusChecker != nil && !m.statusChecker.ClusterQueueActive(lq.ClusterQueue) {
		inadmissible += active
		active = 0
	}
	namespace, lqName := MustParseLocalQueueReference(lq.Key)
	metrics.ReportLocalQueuePendingWorkloads(metrics.LocalQueueReference{
		Name:      lqName,
		Namespace: namespace,
	}, active, inadmissible)
}

func (m *Manager) Broadcast() {
	m.cond.Broadcast()
}

// UpdateWorkload 更新工作负载到相应的队列，如果不存在则添加它
// 返回队列是否存在
func (m *Manager) UpdateWorkload(oldW, w *kueue.Workload) error {
	m.Lock()
	defer m.Unlock()
	if oldW.Spec.QueueName != w.Spec.QueueName {
		m.deleteWorkloadFromQueueAndClusterQueue(w, KeyFromWorkload(oldW))
	}
	return m.AddOrUpdateWorkloadWithoutLock(w) // ✅
}

func (m *Manager) QueueForWorkloadExists(wl *kueue.Workload) bool {
	m.RLock()
	defer m.RUnlock()
	_, ok := m.localQueues[KeyFromWorkload(wl)]
	return ok
}

// ClusterQueueForWorkload 返回工作负载应该排队的集群队列名称以及它是否存在
// 如果队列不存在则返回空字符串
func (m *Manager) ClusterQueueForWorkload(wl *kueue.Workload) (kueue.ClusterQueueReference, bool) {
	m.RLock()
	defer m.RUnlock()
	q, ok := m.localQueues[KeyFromWorkload(wl)]
	if !ok {
		return "", false
	}
	ok = m.hm.ClusterQueue(q.ClusterQueue) != nil
	return q.ClusterQueue, ok
}

// RequeueWorkload 重新排队工作负载，确保队列和工作负载在客户端缓存中仍然存在且未被接纳
// 如果工作负载已经在队列中（如果工作负载被更新则可能），则不会重新排队
func (m *Manager) RequeueWorkload(ctx context.Context, info *workload.Info, reason RequeueReason) bool {
	m.Lock()
	defer m.Unlock()

	var w kueue.Workload
	// 始终获取最新的工作负载以避免重新排队过时的对象
	err := m.client.Get(ctx, client.ObjectKeyFromObject(info.Obj), &w)
	// 由于客户端是缓存的，唯一可能的错误是 NotFound
	if apierrors.IsNotFound(err) || workload.HasQuotaReservation(&w) {
		return false
	}

	q := m.localQueues[KeyFromWorkload(&w)]
	if q == nil {
		return false
	}
	info.Update(&w)
	q.AddOrUpdate(info)
	cq := m.hm.ClusterQueue(q.ClusterQueue)
	if cq == nil {
		return false
	}

	added := cq.RequeueIfNotPresent(info, reason)
	m.reportPendingWorkloads(q.ClusterQueue, cq)
	if features.Enabled(features.LocalQueueMetrics) {
		m.reportLQPendingWorkloads(q)
	}
	if added {
		m.Broadcast()
	}
	return added
}

func (m *Manager) DeleteWorkload(w *kueue.Workload) {
	m.Lock()
	defer m.Unlock()
	m.deleteWorkloadFromQueueAndClusterQueue(w, KeyFromWorkload(w))
	m.DeleteSecondPassWithoutLock(w)
}
