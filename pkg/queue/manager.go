// Package queue 实现了 Kueue 的核心队列管理逻辑，负责本地队列、集群队列、cohort 组的管理，以及工作负载的调度、重排、二次调度等。
package queue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/sets"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	config "sigs.k8s.io/kueue/apis/config/v1beta1"
	kueuealpha "sigs.k8s.io/kueue/apis/kueue/v1alpha1"
	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	utilindexer "sigs.k8s.io/kueue/pkg/controller/core/over_indexer"
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

// Option configures the manager.
type Option func(*options)

var defaultOptions = options{
	podsReadyRequeuingTimestamp: config.EvictionTimestamp,
	workloadInfoOptions:         []workload.InfoOption{},
	clock:                       clock.RealClock{},
}

// WithClock allows to specify a custom clock
func WithClock(c clock.WithDelayedExecution) Option {
	return func(o *options) {
		o.clock = c
	}
}

func WithAdmissionFairSharing(cfg *config.AdmissionFairSharing) Option {
	return func(o *options) {
		if features.Enabled(features.AdmissionFairSharing) {
			o.admissionFairSharing = cfg
		}
	}
}

// WithPodsReadyRequeuingTimestamp sets the timestamp that is used for ordering
// workloads that have been requeued due to the PodsReady condition.
func WithPodsReadyRequeuingTimestamp(ts config.RequeuingTimestamp) Option {
	return func(o *options) {
		o.podsReadyRequeuingTimestamp = ts
	}
}

// WithExcludedResourcePrefixes sets the list of excluded resource prefixes
func WithExcludedResourcePrefixes(excludedPrefixes []string) Option {
	return func(o *options) {
		o.workloadInfoOptions = append(o.workloadInfoOptions, workload.WithExcludedResourcePrefixes(excludedPrefixes))
	}
}

type TopologyUpdateWatcher interface {
	NotifyTopologyUpdate(oldTopology, newTopology *kueuealpha.Topology)
}

// Manager 是队列的核心管理器，负责本地队列、集群队列、cohort 组的增删改查，
// 以及工作负载的入队、出队、重排、二次调度等。
// 该结构体通过锁保证并发安全，支持多种调度策略和特性开关。
type Manager struct {
	sync.RWMutex           // 读写锁，保护大部分成员变量，所有涉及本地队列、集群队列、cohort 的操作都需加锁
	cond         sync.Cond // 条件变量，用于队列空时阻塞等待调度器唤醒

	clock         clock.WithDelayedExecution          // 时钟接口，便于测试和延迟调度
	client        client.Client                       // k8s 客户端，访问 API 对象
	statusChecker StatusChecker                       // 状态检查器，判断集群队列是否可用（如暂停、禁用等）
	localQueues   map[LocalQueueReference]*LocalQueue // 本地队列映射，key 为 namespace/name

	snapshotsMutex sync.RWMutex                                                        // 快照锁，保护 snapshots，避免并发读写
	snapshots      map[kueue.ClusterQueueReference][]kueue.ClusterQueuePendingWorkload // 集群队列快照，记录每个 CQ 的待调度 workload 列表

	workloadOrdering workload.Ordering // 工作负载排序策略，影响堆的优先级

	workloadInfoOptions []workload.InfoOption // 工作负载信息构造参数，支持资源过滤、转换等

	hm hierarchy.Manager[*ClusterQueue, *cohort] // 层级管理器，管理 Cohort 及 CQ 的层级关系，支持父子 Cohort、CQ 归属

	topologyUpdateWatchers []TopologyUpdateWatcher // 拓扑变更监听器，支持外部感知拓扑变化

	admissionFairSharingConfig *config.AdmissionFairSharing // 公平调度配置，影响调度策略
	secondPassQueue            *secondPassQueue             // 二次调度队列，存放需要延迟再次调度的 workload
}

// AddTopologyUpdateWatcher 添加一个拓扑变更监听器。
// 该监听器会在拓扑（如 Topology CRD）变更时被通知。
func (m *Manager) AddTopologyUpdateWatcher(watcher TopologyUpdateWatcher) {
	m.topologyUpdateWatchers = append(m.topologyUpdateWatchers, watcher)
}

// NotifyTopologyUpdateWatchers 通知所有拓扑变更监听器。
// 传递旧拓扑和新拓扑对象，供监听器感知变化。
func (m *Manager) NotifyTopologyUpdateWatchers(oldTopology, newTopology *kueuealpha.Topology) {
	for _, watcher := range m.topologyUpdateWatchers {
		watcher.NotifyTopologyUpdate(oldTopology, newTopology)
	}
}

// AddOrUpdateCohort 新增或更新 cohort 组，并根据变更尝试重排相关工作负载。
// 1. 加锁保护 cohort 结构。
// 2. 更新 cohort 及其父子关系。
// 3. 若 cohort 变更导致有 workload 可调度，则唤醒调度器。
func (m *Manager) AddOrUpdateCohort(ctx context.Context, cohort *kueue.Cohort) {
	m.Lock()
	defer m.Unlock()
	cohortName := kueue.CohortReference(cohort.Name)

	// 添加或更新 cohort 及其父子关系
	m.hm.AddCohort(cohortName)
	m.hm.UpdateCohortEdge(cohortName, cohort.Spec.ParentName)
	// 尝试重排 cohort 下所有 CQ 的不可接纳 workload
	if m.requeueWorkloadsCohort(ctx, m.hm.Cohort(cohortName)) {
		m.Broadcast() // 唤醒调度器
	}
}

// DeleteCohort 删除 cohort 组。
// 只修改内存结构，不影响已存在的 CQ 和 workload。
func (m *Manager) DeleteCohort(cohortName kueue.CohortReference) {
	m.Lock()
	defer m.Unlock()
	m.hm.DeleteCohort(cohortName)
}

// AddClusterQueue 新增集群队列，并将相关本地队列和工作负载加入。
// 1. 检查 CQ 是否已存在，若存在报错。
// 2. 创建 CQ 实例，加入层级管理器。
// 3. 遍历所有指向该 CQ 的本地队列，将其 workload 加入 CQ。
// 4. 尝试重排 CQ 下的不可接纳 workload。
// 5. 上报指标，必要时唤醒调度器。
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
	m.hm.UpdateClusterQueueEdge(kueue.ClusterQueueReference(cq.Name), cq.Spec.Cohort)

	// 遍历所有指向该 CQ 的本地队列，将其 workload 加入 CQ
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

	// 尝试重排 CQ 下的不可接纳 workload
	queued := m.requeueWorkloadsCQ(ctx, cqImpl)
	m.reportPendingWorkloads(kueue.ClusterQueueReference(cq.Name), cqImpl)

	// 再次遍历本地队列，上报指标（防止有新 workload 被加入）
	if features.Enabled(features.LocalQueueMetrics) {
		for _, q := range queues.Items {
			qImpl := m.localQueues[Key(&q)]
			if qImpl != nil {
				m.reportLQPendingWorkloads(qImpl)
			}
		}
	}

	if queued || addedWorkloads {
		m.Broadcast() // 唤醒调度器
	}
	return nil
}

// UpdateClusterQueue 更新集群队列，并根据变更重排相关工作负载。
// 1. 查找 CQ 实例，若不存在报错。
// 2. 更新 CQ 配置（如调度策略、cohort 归属等）。
// 3. 若变更导致有 workload 可调度，则唤醒调度器。
func (m *Manager) UpdateClusterQueue(ctx context.Context, cq *kueue.ClusterQueue, specUpdated bool) error {
	m.Lock()
	defer m.Unlock()
	cqName := kueue.ClusterQueueReference(cq.Name)

	cqImpl := m.hm.ClusterQueue(cqName)
	if cqImpl == nil {
		return ErrClusterQueueDoesNotExist
	}

	oldActive := cqImpl.Active()
	// TODO(#8): 若调度策略变更，需重建堆结构
	if err := cqImpl.Update(cq); err != nil {
		return err
	}
	m.hm.UpdateClusterQueueEdge(cqName, cq.Spec.Cohort)

	// 若变更导致有 workload 可调度，则唤醒调度器
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

// HeapifyClusterQueue 重新构建指定集群队列的堆结构。
// 主要用于本地队列变更后，保证堆顺序正确。
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

// DeleteClusterQueue 删除集群队列及其相关指标。
// 只影响内存结构和指标，不会删除本地队列。
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

// DefaultLocalQueueExist 判断指定命名空间下的默认本地队列是否存在。
// 通过 DefaultQueueKey(namespace) 查询。
func (m *Manager) DefaultLocalQueueExist(namespace string) bool {
	m.Lock()
	defer m.Unlock()

	_, ok := m.localQueues[DefaultQueueKey(namespace)]
	return ok
}

// AddLocalQueue 新增本地队列，并将相关工作负载加入。
// 1. 检查队列是否已存在。
// 2. 创建本地队列实例，加入管理器。
// 3. 遍历所有指向该队列的 workload，将其加入本地队列。
// 4. 若对应 CQ 存在，则将本地队列的 workload 加入 CQ。
func (m *Manager) AddLocalQueue(ctx context.Context, q *kueue.LocalQueue) error {
	m.Lock()
	defer m.Unlock()

	key := Key(q)
	if _, ok := m.localQueues[key]; ok {
		return fmt.Errorf("queue %q already exists", q.Name)
	}
	qImpl := newLocalQueue(q)
	m.localQueues[key] = qImpl
	// 遍历所有指向该队列的 workload，将其加入本地队列
	var workloads kueue.WorkloadList
	if err := m.client.List(ctx, &workloads, client.MatchingFields{utilindexer.WorkloadQueueKey: q.Name}, client.InNamespace(q.Namespace)); err != nil {
		return fmt.Errorf("listing workloads that match the queue: %w", err)
	}
	for _, w := range workloads.Items {
		if !workload.IsActive(&w) || workload.HasQuotaReservation(&w) {
			continue // 跳过非活跃或已分配配额的 workload
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

// UpdateLocalQueue 更新本地队列信息，并处理队列迁移。
// 若队列归属的 CQ 发生变化，则从旧 CQ 删除并加入新 CQ。
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

// DeleteLocalQueue 删除本地队列及其相关指标。
// 1. 从 CQ 删除本地队列。
// 2. 清理本地队列指标。
// 3. 从管理器移除本地队列。
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

// PendingWorkloads 返回本地队列中待调度的工作负载数量。
// 若队列不存在则报错。
func (m *Manager) PendingWorkloads(q *kueue.LocalQueue) (int32, error) {
	m.RLock()
	defer m.RUnlock()

	qImpl, ok := m.localQueues[Key(q)]
	if !ok {
		return 0, ErrLocalQueueDoesNotExistOrInactive
	}

	return int32(len(qImpl.items)), nil
}

// Pending 返回集群队列中待调度的工作负载数量。
// 若队列不存在则报错。
func (m *Manager) Pending(cq *kueue.ClusterQueue) (int, error) {
	m.RLock()
	defer m.RUnlock()

	cqImpl := m.hm.ClusterQueue(kueue.ClusterQueueReference(cq.Name))
	if cqImpl == nil {
		return 0, ErrClusterQueueDoesNotExist
	}

	return cqImpl.Pending(), nil
}

// QueueForWorkloadExists 判断工作负载对应的本地队列是否存在。
// 通过 workload 的队列 key 查询。
func (m *Manager) QueueForWorkloadExists(wl *kueue.Workload) bool {
	m.RLock()
	defer m.RUnlock()
	_, ok := m.localQueues[KeyFromWorkload(wl)]
	return ok
}

// ClusterQueueForWorkload 返回工作负载应入队的集群队列名及其是否存在。
// 若本地队列不存在或 CQ 不存在则返回 false。
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

// AddOrUpdateWorkloadWithoutLock 在不加锁的情况下将工作负载加入队列。
// 1. 查找本地队列，若不存在报错。
// 2. 构造 workload.Info 并加入本地队列。
// 3. 加入对应的 CQ。
// 4. 上报指标并唤醒调度器。
func (m *Manager) AddOrUpdateWorkloadWithoutLock(w *kueue.Workload) error {
	qKey := KeyFromWorkload(w)
	q := m.localQueues[qKey]
	if q == nil {
		return ErrLocalQueueDoesNotExistOrInactive
	}
	wInfo := workload.NewInfo(w, m.workloadInfoOptions...)
	q.AddOrUpdate(wInfo)
	cq := m.hm.ClusterQueue(q.ClusterQueue)
	if cq == nil {
		return ErrClusterQueueDoesNotExist
	}
	cq.PushOrUpdate(wInfo)
	if features.Enabled(features.LocalQueueMetrics) {
		m.reportLQPendingWorkloads(q)
	}
	m.reportPendingWorkloads(q.ClusterQueue, cq)
	m.Broadcast()
	return nil
}

// RequeueWorkload 重新入队指定工作负载，确保其仍然有效且未被调度。
// 1. 重新从 API 获取最新 workload，避免过期。
// 2. 若 workload 已被调度或本地队列不存在则跳过。
// 3. 更新 workload.Info 并加入本地队列。
// 4. 加入 CQ 并上报指标，必要时唤醒调度器。
func (m *Manager) RequeueWorkload(ctx context.Context, info *workload.Info, reason RequeueReason) bool {
	m.Lock()
	defer m.Unlock()

	var w kueue.Workload
	// 始终获取最新 workload，避免过期对象
	err := m.client.Get(ctx, client.ObjectKeyFromObject(info.Obj), &w)
	// 只可能是 NotFound
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

// DeleteWorkload 从本地队列和集群队列中删除工作负载，并清理二次调度队列。
// 1. 删除本地队列和 CQ 中的 workload。
// 2. 从二次调度队列中移除。
func (m *Manager) DeleteWorkload(w *kueue.Workload) {
	m.Lock()
	defer m.Unlock()
	m.deleteWorkloadFromQueueAndClusterQueue(w, KeyFromWorkload(w))
	m.DeleteSecondPassWithoutLock(w)
}

// deleteWorkloadFromQueueAndClusterQueue 内部方法，从本地队列和集群队列中删除工作负载。
// 若 CQ 存在则同步删除。
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

// QueueAssociatedInadmissibleWorkloadsAfter 在指定工作负载被调度后，尝试将同 cohort 下的不可接纳工作负载重新入队。
// 1. 可选地执行 action（如原子操作）。
// 2. 若本地队列和 CQ 存在，则重排 CQ 下的不可接纳 workload。
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

// UpdateWorkload 更新工作负载，若队列变更则先删除再添加。
// 1. 若队列名变更，先从旧队列删除。
// 2. 再将新 workload 加入队列。
func (m *Manager) UpdateWorkload(oldW, w *kueue.Workload) error {
	m.Lock()
	defer m.Unlock()
	if oldW.Spec.QueueName != w.Spec.QueueName {
		m.deleteWorkloadFromQueueAndClusterQueue(w, KeyFromWorkload(oldW))
	}
	return m.AddOrUpdateWorkloadWithoutLock(w)
}

// CleanUpOnContext 监听 context 关闭，唤醒所有等待队列的协程。
// 用于优雅关闭调度器。
func (m *Manager) CleanUpOnContext(ctx context.Context) {
	<-ctx.Done()
	m.Broadcast()
}

// heads 返回所有可调度的工作负载（包括二次调度队列和各集群队列的堆顶），并做必要的指标上报和队列维护。
// 1. 先取出所有 ready 的二次调度 workload。
// 2. 遍历所有 CQ，若 CQ 可用则弹出堆顶 workload。
// 3. 删除本地队列中的 workload，更新指标。
func (m *Manager) heads() []workload.Info {
	workloads := m.secondPassQueue.takeAllReady()
	for cqName, cq := range m.hm.ClusterQueues() {
		// statusChecker 可用于跳过不可用的 CQ
		if m.statusChecker != nil && !m.statusChecker.ClusterQueueActive(cqName) {
			continue
		}
		wl := cq.Pop()
		if wl == nil {
			continue
		}
		m.reportPendingWorkloads(cqName, cq) // ✅
		wlCopy := *wl
		wlCopy.ClusterQueue = cqName
		workloads = append(workloads, wlCopy)
		q := m.localQueues[KeyFromWorkload(wl.Obj)]
		delete(q.items, workload.Key(wl.Obj))
		if features.Enabled(features.LocalQueueMetrics) {
			m.reportLQPendingWorkloads(q) // ✅
		}
	}
	return workloads
}

// GetClusterQueueNames 返回所有集群队列的名称。
func (m *Manager) GetClusterQueueNames() []kueue.ClusterQueueReference {
	m.RLock()
	defer m.RUnlock()
	return m.hm.ClusterQueuesNames()
}

// PendingWorkloadsInfo 返回指定集群队列的所有待调度工作负载信息。
func (m *Manager) PendingWorkloadsInfo(cqName kueue.ClusterQueueReference) []*workload.Info {
	cq := m.getClusterQueue(cqName)
	if cq == nil {
		return nil
	}
	return cq.Snapshot()
}

// ClusterQueueFromLocalQueue 根据本地队列 key 返回对应的集群队列名。
func (m *Manager) ClusterQueueFromLocalQueue(localQueueKey LocalQueueReference) (kueue.ClusterQueueReference, bool) {
	m.RLock()
	defer m.RUnlock()
	if lq, ok := m.localQueues[localQueueKey]; ok {
		return lq.ClusterQueue, true
	}
	return "", false
}

// DeleteSecondPassWithoutLock 从二次调度队列中删除指定工作负载。
func (m *Manager) DeleteSecondPassWithoutLock(w *kueue.Workload) {
	m.secondPassQueue.deleteByKey(workload.Key(w))
}

// QueueSecondPassIfNeeded 如果工作负载需要二次调度，则延迟 1 秒后入队。
// 1. 先将 workload 放入 prequeue 状态。
// 2. 1 秒后异步调用 queueSecondPass 正式入队。
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

// queueSecondPass 真正将工作负载加入二次调度队列，并唤醒调度协程。
// 若队列有变化则唤醒调度器。
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

// WithResourceTransformations 设置资源转换规则。
func WithResourceTransformations(transforms []config.ResourceTransformation) Option {
	return func(o *options) {
		o.workloadInfoOptions = append(o.workloadInfoOptions, workload.WithResourceTransformations(transforms))
	}
}

// NewManager 创建并初始化一个队列管理器。
// 初始化所有成员变量，设置默认参数。
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

// QueueInadmissibleWorkloads 将指定集群队列中的不可接纳工作负载重新入队。
// 1. 遍历所有 CQ，调用 requeueWorkloadsCQ。
// 2. 若有 workload 被重新入队，则唤醒调度器。
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

// requeueWorkloadsCQ 会将同一群组中与该集群队列相同的所有工作负载从“不可接纳工作负载”列表中移至堆区。
// 如果该集群队列的队列中已无元素，它只会将此集群队列中的所有工作负载移出。
// 如果移出了至少一个工作负载，则返回真，否则返回假。
// 下列事件可能会使同一队列中的工作负载变得可执行。然后需要调用 requeueWorkloadsCQ。
// 1. 删除队列中任何已获准的工作负载的事件。
// 2. 添加队列中的任何集群队列的事件。
// 3. 更新队列中的任何集群队列的事件。
// 4. 更新队列本身。
// 警告：在调用时必须对管理器持有读锁，否则若引入了群组循环，则可能会陷入无限循环的危险境地。
func (m *Manager) requeueWorkloadsCQ(ctx context.Context, cq *ClusterQueue) bool {
	if cq.HasParent() {
		return m.requeueWorkloadsCohort(ctx, cq.Parent())
	}
	return cq.QueueInadmissibleWorkloads(ctx, m.client)
}

// moveWorkloadsCohorts 检查 cohort 是否有环，并递归将 cohort 树下所有不可接纳工作负载重新入队。
// 如果有环或没有移动任何工作负载，则返回 false。
//
// 警告：调用时必须持有读锁，否则可能因 cohort 循环导致死循环。
func (m *Manager) requeueWorkloadsCohort(ctx context.Context, cohort *cohort) bool {
	log := ctrl.LoggerFrom(ctx)

	if hierarchy.HasCycle(cohort) {
		log.V(2).Info("Attempted to move workloads from Cohort which has cycle", "cohort", cohort.GetName())
		return false
	}
	root := cohort.getRootUnsafe()
	log.V(2).Info("Attempting to move workloads", "cohort", cohort.Name, "root", root.Name)
	return requeueWorkloadsCohortSubtree(ctx, m, root)
}

// requeueWorkloadsCohortSubtree 递归将 cohort 子树下所有集群队列的不可接纳工作负载重新入队。
// 返回是否有 workload 被重新入队。
func requeueWorkloadsCohortSubtree(ctx context.Context, m *Manager, cohort *cohort) bool {
	queued := false
	for _, clusterQueue := range cohort.ChildCQs() {
		queued = clusterQueue.QueueInadmissibleWorkloads(ctx, m.client) || queued
	}
	for _, childCohort := range cohort.ChildCohorts() {
		queued = requeueWorkloadsCohortSubtree(ctx, m, childCohort) || queued
	}
	return queued
}

// Broadcast 唤醒所有等待队列的协程。
func (m *Manager) Broadcast() {
	m.cond.Broadcast()
}

// UpdateSnapshot 计算并更新集群队列的快照，若有变化则返回 true。
// 1. 获取 CQ 的所有待调度 workload。
// 2. 若快照内容有变化则更新。
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

// getClusterQueue 获取指定名称的集群队列（加读锁）。
func (m *Manager) getClusterQueue(cqName kueue.ClusterQueueReference) *ClusterQueue {
	m.RLock()
	defer m.RUnlock()
	return m.hm.ClusterQueue(cqName)
}

// setSnapshot 设置集群队列的快照。
func (m *Manager) setSnapshot(cqName kueue.ClusterQueueReference, workloads []kueue.ClusterQueuePendingWorkload) {
	m.snapshotsMutex.Lock()
	defer m.snapshotsMutex.Unlock()
	m.snapshots[cqName] = workloads
}

// GetSnapshot 获取集群队列的快照。
func (m *Manager) GetSnapshot(cqName kueue.ClusterQueueReference) []kueue.ClusterQueuePendingWorkload {
	m.snapshotsMutex.RLock()
	defer m.snapshotsMutex.RUnlock()
	return m.snapshots[cqName]
}

// DeleteSnapshot 删除集群队列的快照。
func (m *Manager) DeleteSnapshot(cq *kueue.ClusterQueue) {
	m.snapshotsMutex.Lock()
	defer m.snapshotsMutex.Unlock()
	delete(m.snapshots, kueue.ClusterQueueReference(cq.Name))
}

// AddOrUpdateWorkload 加锁后将工作负载加入队列。
func (m *Manager) AddOrUpdateWorkload(w *kueue.Workload) error {
	m.Lock()
	defer m.Unlock()
	return m.AddOrUpdateWorkloadWithoutLock(w)
}

// Heads 返回所有可调度的工作负载（阻塞直到有元素或 context 关闭）。
// 1. 若队列有可调度 workload，则直接返回。
// 2. 否则阻塞等待，直到有新 workload 或 context 关闭。
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

// reportPendingWorkloads 上报集群队列的待调度和不可接纳工作负载指标。
// 若 CQ 不可用，则全部计为不可接纳。
func (m *Manager) reportPendingWorkloads(cqName kueue.ClusterQueueReference, cq *ClusterQueue) {
	active := cq.PendingActive()
	inadmissible := cq.PendingInadmissible()
	if m.statusChecker != nil && !m.statusChecker.ClusterQueueActive(cqName) {
		inadmissible += active
		active = 0
	}
	metrics.ReportPendingWorkloads(cqName, active, inadmissible)
}

// getClusterQueueLockless 无锁获取集群队列。
func (m *Manager) getClusterQueueLockless(cqName kueue.ClusterQueueReference) (val *ClusterQueue, ok bool) {
	val = m.hm.ClusterQueue(cqName)
	return val, val != nil
}

// reportLQPendingWorkloads 上报本地队列的待调度和不可接纳工作负载指标。
// 若 CQ 不可用，则全部计为不可接纳。
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
