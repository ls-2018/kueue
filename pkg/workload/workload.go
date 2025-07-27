package workload

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"maps"
	"slices"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	resourcehelpers "k8s.io/component-helpers/resource"
	"k8s.io/klog/v2"
	"k8s.io/utils/clock"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	config "sigs.k8s.io/kueue/apis/config/v1beta1"
	kueuealpha "sigs.k8s.io/kueue/apis/kueue/v1alpha1"
	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/constants"
	"sigs.k8s.io/kueue/pkg/features"
	"sigs.k8s.io/kueue/pkg/metrics"
	"sigs.k8s.io/kueue/pkg/resources"
	"sigs.k8s.io/kueue/pkg/util/api"
	utilptr "sigs.k8s.io/kueue/pkg/util/ptr"
	utilslices "sigs.k8s.io/kueue/pkg/util/slices"
)

const (
	StatusPending       = "pending"
	StatusQuotaReserved = "quotaReserved"
	StatusAdmitted      = "admitted"
	StatusFinished      = "finished"
)

var (
	admissionManagedConditions = []string{
		kueue.WorkloadQuotaReserved,
		kueue.WorkloadEvicted,
		kueue.WorkloadAdmitted,
		kueue.WorkloadPreempted,
		kueue.WorkloadRequeued,
		kueue.WorkloadDeactivationTarget,
	}
)

func Status(w *kueue.Workload) string {
	if apimeta.IsStatusConditionTrue(w.Status.Conditions, kueue.WorkloadFinished) {
		return StatusFinished
	}
	if IsAdmitted(w) {
		return StatusAdmitted
	}
	if HasQuotaReservation(w) {
		return StatusQuotaReserved
	}
	return StatusPending
}

type AssignmentClusterQueueState struct {
	LastTriedFlavorIdx     []map[corev1.ResourceName]int
	ClusterQueueGeneration int64
}

type InfoOptions struct {
	excludedResourcePrefixes []string
	resourceTransformations  map[corev1.ResourceName]*config.ResourceTransformation
}

type InfoOption func(*InfoOptions)

var defaultOptions = InfoOptions{}

// WithExcludedResourcePrefixes 添加排除的资源前缀
func WithExcludedResourcePrefixes(n []string) InfoOption {
	return func(o *InfoOptions) {
		o.excludedResourcePrefixes = n
	}
}

// WithResourceTransformations 设置资源转换
func WithResourceTransformations(transforms []config.ResourceTransformation) InfoOption {
	return func(o *InfoOptions) {
		o.resourceTransformations = utilslices.ToRefMap(transforms, func(e *config.ResourceTransformation) corev1.ResourceName { return e.Input })
	}
}

func (s *AssignmentClusterQueueState) Clone() *AssignmentClusterQueueState {
	c := AssignmentClusterQueueState{
		LastTriedFlavorIdx:     make([]map[corev1.ResourceName]int, len(s.LastTriedFlavorIdx)),
		ClusterQueueGeneration: s.ClusterQueueGeneration,
	}
	for ps, flavorIdx := range s.LastTriedFlavorIdx {
		c.LastTriedFlavorIdx[ps] = maps.Clone(flavorIdx)
	}
	return &c
}

// PendingFlavors 返回是否还有待尝试的 flavor
// 在最后一次尝试之后。
func (s *AssignmentClusterQueueState) PendingFlavors() bool {
	if s == nil {
		// 这种情况仅在单元测试中达到。
		return false
	}
	for _, podSetIdxs := range s.LastTriedFlavorIdx {
		for _, idx := range podSetIdxs {
			if idx != -1 {
				return true
			}
		}
	}
	return false
}

// Info 持有 Workload 对象并进行一些预处理。
type Info struct {
	Obj *kueue.Workload
	// 工作负载请求的总资源列表。
	TotalRequests []PodSetResources
	// 从队列在准入或已准入时填充。
	ClusterQueue   kueue.ClusterQueueReference
	LastAssignment *AssignmentClusterQueueState
}

type PodSetResources struct {
	Name                   kueue.PodSetReference
	Requests               resources.Requests //总资源   pod * count
	Count                  int32
	TopologyRequest        *TopologyRequest
	DelayedTopologyRequest *kueue.DelayedTopologyRequestState
	Flavors                map[corev1.ResourceName]kueue.ResourceFlavorReference // 当工作负载被分配时，这些区域就会被填满。
}

func (p *PodSetResources) SinglePodRequests() resources.Requests {
	return p.Requests.ScaledDown(int64(p.Count))
}

type TopologyRequest struct {
	Levels         []string
	DomainRequests []TopologyDomainRequests
}

type TopologyDomainRequests struct {
	Values            []string
	SinglePodRequests resources.Requests
	// Count 表示此 TopologyDomain 中请求的 pod 数量。
	Count int32
}

func (t *TopologyDomainRequests) TotalRequests() resources.Requests {
	return t.SinglePodRequests.ScaledUp(int64(t.Count))
}

func (i *Info) Update(wl *kueue.Workload) {
	i.Obj = wl
}

// Usage 返回工作负载的总资源使用量，包括常规配额和 TAS 使用量。
func (i *Info) Usage() Usage {
	return Usage{
		Quota: i.FlavorResourceUsage(),
		TAS:   i.TASUsage(),
	}
}

// FlavorResourceUsage 返回工作负载的总资源使用量，
// 每种 flavor（如果已分配，否则 flavor 显示为空字符串），每种资源。
func (i *Info) FlavorResourceUsage() resources.FlavorResourceQuantities {
	total := make(resources.FlavorResourceQuantities)
	if i == nil {
		return total
	}
	for _, psReqs := range i.TotalRequests {
		for res, q := range psReqs.Requests {
			flv := psReqs.Flavors[res]
			total[resources.FlavorResource{Flavor: flv, Resource: res}] += q
		}
	}
	return total
}

func (i *Info) LocalQueueUsage(ctx context.Context, c client.Client, resWeights map[corev1.ResourceName]float64) (float64, error) {
	var lq kueue.LocalQueue
	lqKey := client.ObjectKey{Namespace: i.Obj.Namespace, Name: string(i.Obj.Spec.QueueName)}
	if err := c.Get(ctx, lqKey, &lq); err != nil {
		return 0, err
	}
	var usage float64
	for resName, resVal := range lq.Status.FairSharing.AdmissionFairSharingStatus.ConsumedResources {
		weight, found := resWeights[resName]
		if !found {
			weight = 1
		}
		usage += weight * resVal.AsApproximateFloat64()
	}
	if lq.Spec.FairSharing != nil && lq.Spec.FairSharing.Weight != nil {
		// 如果没有定义 lq 的权重，则使用默认权重 1
		usage /= lq.Spec.FairSharing.Weight.AsApproximateFloat64()
	}
	return usage, nil
}

// IsUsingTAS 返回工作负载是否使用 TAS
func (i *Info) IsUsingTAS() bool {
	return slices.ContainsFunc(i.TotalRequests,
		func(ps PodSetResources) bool {
			return ps.TopologyRequest != nil
		})
}

// TASUsage 返回工作负载请求的拓扑使用量
func (i *Info) TASUsage() TASUsage {
	if !features.Enabled(features.TopologyAwareScheduling) || !i.IsUsingTAS() {
		return nil
	}
	result := make(TASUsage, 0)
	for _, ps := range i.TotalRequests {
		if ps.TopologyRequest != nil {
			psFlavors := sets.New[kueue.ResourceFlavorReference]()
			for _, psFlavor := range ps.Flavors {
				psFlavors.Insert(psFlavor)
			}
			for psFlavor := range psFlavors {
				result[psFlavor] = append(result[psFlavor], ps.TopologyRequest.DomainRequests...)
			}
		}
	}
	return result
}

func Key(w *kueue.Workload) string {
	return fmt.Sprintf("%s/%s", w.Namespace, w.Name)
}

func PodSetNameToTopologyRequest(wl *kueue.Workload) map[kueue.PodSetReference]*kueue.PodSetTopologyRequest {
	return utilslices.ToMap(wl.Spec.PodSets, func(i int) (kueue.PodSetReference, *kueue.PodSetTopologyRequest) {
		return wl.Spec.PodSets[i].Name, wl.Spec.PodSets[i].TopologyRequest
	})
}

// UpdateStatus 更新工作负载的条件，
// fieldManager 设置为 managerPrefix + "-" + conditionType
func UpdateStatus(ctx context.Context,
	c client.Client,
	wl *kueue.Workload,
	conditionType string,
	conditionStatus metav1.ConditionStatus,
	reason, message string,
	managerPrefix string,
	clock clock.Clock) error {
	now := metav1.NewTime(clock.Now())
	condition := metav1.Condition{
		Type:               conditionType,
		Status:             conditionStatus,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            api.TruncateConditionMessage(message),
		ObservedGeneration: wl.Generation,
	}

	newWl := BaseSSAWorkload(wl)
	newWl.Status.Conditions = []metav1.Condition{condition}
	return c.Status().Patch(ctx, newWl, client.Apply, client.FieldOwner(managerPrefix+"-"+condition.Type))
}

// UnsetQuotaReservationWithCondition 将 QuotaReserved 条件设置为 false，清除
// 准入和设置 WorkloadRequeued 状态。
// 返回是否进行了任何更改。
func UnsetQuotaReservationWithCondition(wl *kueue.Workload, reason, message string, now time.Time) bool {
	condition := metav1.Condition{
		Type:               kueue.WorkloadQuotaReserved,
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            api.TruncateConditionMessage(message),
		ObservedGeneration: wl.Generation,
	}
	changed := apimeta.SetStatusCondition(&wl.Status.Conditions, condition)
	if wl.Status.Admission != nil {
		wl.Status.Admission = nil
		changed = true
	}

	// 如果需要，则重置已准入条件。
	if SyncAdmittedCondition(wl, now) {
		changed = true
	}
	return changed
}

// UpdateRequeueState 计算 requeueAt 时间并更新 requeuingCount
func UpdateRequeueState(wl *kueue.Workload, backoffBaseSeconds int32, backoffMaxSeconds int32, clock clock.Clock) {
	if wl.Status.RequeueState == nil {
		wl.Status.RequeueState = &kueue.RequeueState{}
	}
	requeuingCount := ptr.Deref(wl.Status.RequeueState.Count, 0) + 1

	// 每次回退持续时间大约为 "60s*2^(n-1)+Rand"，其中：
	// - "n" 表示 "requeuingCount"，
	// - "Rand" 表示随机抖动。
	// 在此期间，工作负载被视为不可接纳，其他
	// 工作负载将有资格被接纳。
	backoff := &wait.Backoff{
		Duration: time.Duration(backoffBaseSeconds) * time.Second,
		Factor:   2,
		Jitter:   0.0001,
		Steps:    int(requeuingCount),
	}
	var waitDuration time.Duration
	for backoff.Steps > 0 {
		waitDuration = min(backoff.Step(), time.Duration(backoffMaxSeconds)*time.Second)
	}

	wl.Status.RequeueState.RequeueAt = ptr.To(metav1.NewTime(clock.Now().Add(waitDuration)))
	wl.Status.RequeueState.Count = &requeuingCount
}

// SetRequeuedCondition 设置 WorkloadRequeued 条件为 true
func SetRequeuedCondition(wl *kueue.Workload, reason, message string, status bool) {
	condition := metav1.Condition{
		Type:               kueue.WorkloadRequeued,
		Reason:             reason,
		Message:            api.TruncateConditionMessage(message),
		ObservedGeneration: wl.Generation,
	}
	if status {
		condition.Status = metav1.ConditionTrue
	} else {
		condition.Status = metav1.ConditionFalse
	}
	apimeta.SetStatusCondition(&wl.Status.Conditions, condition)
}

func QueuedWaitTime(wl *kueue.Workload, clock clock.Clock) time.Duration {
	queuedTime := wl.CreationTimestamp.Time
	if c := apimeta.FindStatusCondition(wl.Status.Conditions, kueue.WorkloadRequeued); c != nil {
		queuedTime = c.LastTransitionTime.Time
	}
	return clock.Since(queuedTime)
}

// SetQuotaReservation 将提供的准入应用于工作负载。
// WorkloadAdmitted 和 WorkloadEvicted 根据需要添加或更新。
func SetQuotaReservation(w *kueue.Workload, admission *kueue.Admission, clock clock.Clock) {
	w.Status.Admission = admission
	message := fmt.Sprintf("ClusterQueue %s 中配额已保留", w.Status.Admission.ClusterQueue)
	admittedCond := metav1.Condition{
		Type:               kueue.WorkloadQuotaReserved,
		Status:             metav1.ConditionTrue,
		Reason:             "QuotaReserved",
		Message:            api.TruncateConditionMessage(message),
		ObservedGeneration: w.Generation,
	}
	apimeta.SetStatusCondition(&w.Status.Conditions, admittedCond)

	// 重置 Evicted 条件（如果存在）。
	if evictedCond := apimeta.FindStatusCondition(w.Status.Conditions, kueue.WorkloadEvicted); evictedCond != nil {
		evictedCond.Status = metav1.ConditionFalse
		evictedCond.Reason = "QuotaReserved"
		evictedCond.Message = api.TruncateConditionMessage("Previously: " + evictedCond.Message)
		evictedCond.LastTransitionTime = metav1.NewTime(clock.Now())
	}
	// 重置 Preempted 条件（如果存在）。
	if preemptedCond := apimeta.FindStatusCondition(w.Status.Conditions, kueue.WorkloadPreempted); preemptedCond != nil {
		preemptedCond.Status = metav1.ConditionFalse
		preemptedCond.Reason = "QuotaReserved"
		preemptedCond.Message = api.TruncateConditionMessage("Previously: " + preemptedCond.Message)
		preemptedCond.LastTransitionTime = metav1.NewTime(clock.Now())
	}
}

// NeedsSecondPass 检查是否需要工作负载的第二次调度。
func NeedsSecondPass(w *kueue.Workload) bool {
	if IsFinished(w) || IsEvicted(w) || !HasQuotaReservation(w) {
		return false
	}
	return needsSecondPassForDelayedAssignment(w) || needsSecondPassAfterNodeFailure(w)
}

func needsSecondPassForDelayedAssignment(w *kueue.Workload) bool {
	return len(w.Status.AdmissionChecks) > 0 &&
		HasAllChecksReady(w) &&
		HasTopologyAssignmentsPending(w) &&
		!IsAdmitted(w)
}

func needsSecondPassAfterNodeFailure(w *kueue.Workload) bool {
	return HasTopologyAssignmentWithNodeToReplace(w)
}

// HasTopologyAssignmentsPending 检查工作负载是否包含任何
// PodSetAssignment 具有 DelayedTopologyRequest=Pending。
func HasTopologyAssignmentsPending(w *kueue.Workload) bool {
	if w.Status.Admission == nil {
		return false
	}
	for _, psa := range w.Status.Admission.PodSetAssignments {
		if psa.TopologyAssignment == nil &&
			utilptr.ValEquals(psa.DelayedTopologyRequest, kueue.DelayedTopologyRequestStatePending) {
			return true
		}
	}
	return false
}

func SetPreemptedCondition(w *kueue.Workload, reason string, message string) {
	condition := metav1.Condition{
		Type:    kueue.WorkloadPreempted,
		Status:  metav1.ConditionTrue,
		Reason:  reason,
		Message: api.TruncateConditionMessage(message),
	}
	apimeta.SetStatusCondition(&w.Status.Conditions, condition)
}

func SetDeactivationTarget(w *kueue.Workload, reason string, message string) {
	condition := metav1.Condition{
		Type:               kueue.WorkloadDeactivationTarget,
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: w.Generation,
	}
	apimeta.SetStatusCondition(&w.Status.Conditions, condition)
}

func SetEvictedCondition(w *kueue.Workload, reason string, message string) {
	condition := metav1.Condition{
		Type:               kueue.WorkloadEvicted,
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            api.TruncateConditionMessage(message),
		ObservedGeneration: w.Generation,
	}
	apimeta.SetStatusCondition(&w.Status.Conditions, condition)
}

// PropagateResourceRequests 同步 w.Status.ResourceRequests 到
// 如果启用了功能门，则与 info.TotalRequests 匹配，并返回 true 如果 w 已更新
func PropagateResourceRequests(w *kueue.Workload, info *Info) bool {
	if len(w.Status.ResourceRequests) == len(info.TotalRequests) {
		match := true
		for idx := range w.Status.ResourceRequests {
			if w.Status.ResourceRequests[idx].Name != info.TotalRequests[idx].Name ||
				!equality.Semantic.DeepEqual(w.Status.ResourceRequests[idx].Resources, info.TotalRequests[idx].Requests.ToResourceList()) {
				match = false
				break
			}
		}
		if match {
			return false
		}
	}

	res := make([]kueue.PodSetRequest, len(info.TotalRequests))
	for idx := range info.TotalRequests {
		res[idx].Name = info.TotalRequests[idx].Name
		res[idx].Resources = info.TotalRequests[idx].Requests.ToResourceList()
	}
	w.Status.ResourceRequests = res
	return true
}

type Ordering struct {
	PodsReadyRequeuingTimestamp config.RequeuingTimestamp
}

// GetQueueOrderTimestamp 返回调度器使用的时戳。它可以是
// 工作负载创建时间或最后一次 PodsReady 超时发生的时间。
func (o Ordering) GetQueueOrderTimestamp(w *kueue.Workload) *metav1.Time {
	if o.PodsReadyRequeuingTimestamp == config.EvictionTimestamp {
		if evictedCond, evictedByTimeout := IsEvictedByPodsReadyTimeout(w); evictedByTimeout {
			return &evictedCond.LastTransitionTime
		}
	}
	if evictedCond, evictedByCheck := IsEvictedByAdmissionCheck(w); evictedByCheck {
		return &evictedCond.LastTransitionTime
	}
	if !features.Enabled(features.PrioritySortingWithinCohort) {
		if preemptedCond := apimeta.FindStatusCondition(w.Status.Conditions, kueue.WorkloadPreempted); preemptedCond != nil &&
			preemptedCond.Status == metav1.ConditionTrue &&
			preemptedCond.Reason == kueue.InCohortReclaimWhileBorrowingReason {
			// 我们添加一个 epsilon 以确保抢占的工作负载的时戳严格大于抢占者的
			return &metav1.Time{Time: preemptedCond.LastTransitionTime.Add(time.Millisecond)}
		}
	}
	return &w.CreationTimestamp
}

// ReclaimablePodsAreEqual 检查两个 Reclaimable pods 是否语义相等
// 具有相同的长度和所有键的值都相同。
func ReclaimablePodsAreEqual(a, b []kueue.ReclaimablePod) bool {
	if len(a) != len(b) {
		return false
	}
	ma := utilslices.ToMap(a, func(i int) (kueue.PodSetReference, int32) { return a[i].Name, a[i].Count })
	mb := utilslices.ToMap(b, func(i int) (kueue.PodSetReference, int32) { return b[i].Name, b[i].Count })
	return maps.Equal(ma, mb)
}

// IsFinished 返回工作负载是否已完成。
func IsFinished(w *kueue.Workload) bool {
	return apimeta.IsStatusConditionTrue(w.Status.Conditions, kueue.WorkloadFinished)
}

// IsActive 返回工作负载是否活跃。
func IsActive(w *kueue.Workload) bool {
	return ptr.Deref(w.Spec.Active, true)
}

// IsEvictedDueToDeactivationByKueue 返回工作负载是否因 kueue 驱逐而驱逐。
func IsEvictedDueToDeactivationByKueue(w *kueue.Workload) bool {
	cond := apimeta.FindStatusCondition(w.Status.Conditions, kueue.WorkloadEvicted)
	return cond != nil && cond.Status == metav1.ConditionTrue &&
		strings.HasPrefix(cond.Reason, fmt.Sprintf("%sDueTo", kueue.WorkloadDeactivated))
}

func IsEvictedByPodsReadyTimeout(w *kueue.Workload) (*metav1.Condition, bool) {
	cond := apimeta.FindStatusCondition(w.Status.Conditions, kueue.WorkloadEvicted)
	if cond == nil || cond.Status != metav1.ConditionTrue || cond.Reason != kueue.WorkloadEvictedByPodsReadyTimeout {
		return nil, false
	}
	return cond, true
}

func IsEvictedByAdmissionCheck(w *kueue.Workload) (*metav1.Condition, bool) {
	cond := apimeta.FindStatusCondition(w.Status.Conditions, kueue.WorkloadEvicted)
	if cond == nil || cond.Status != metav1.ConditionTrue || cond.Reason != kueue.WorkloadEvictedByAdmissionCheck {
		return nil, false
	}
	return cond, true
}

func IsEvicted(w *kueue.Workload) bool {
	return apimeta.IsStatusConditionPresentAndEqual(w.Status.Conditions, kueue.WorkloadEvicted, metav1.ConditionTrue)
}

// HasConditionWithTypeAndReason 检查工作负载状态中是否存在
// 具有完全相同的 Type、Status 和 Reason 的条件。
func HasConditionWithTypeAndReason(w *kueue.Workload, cond *metav1.Condition) bool {
	for _, statusCond := range w.Status.Conditions {
		if statusCond.Type == cond.Type && statusCond.Reason == cond.Reason &&
			statusCond.Status == cond.Status {
			return true
		}
	}
	return false
}

func HasNodeToReplace(w *kueue.Workload) bool {
	if w == nil {
		return false
	}
	annotations := w.GetAnnotations()
	_, found := annotations[kueuealpha.NodeToReplaceAnnotation]
	return found
}

func CreatePodsReadyCondition(status metav1.ConditionStatus, reason, message string, clock clock.Clock) metav1.Condition {
	return metav1.Condition{
		Type:               kueue.WorkloadPodsReady,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.NewTime(clock.Now()),
		// ObservedGeneration 通过 workload.UpdateStatus 添加
	}
}

func RemoveFinalizer(ctx context.Context, c client.Client, wl *kueue.Workload) error {
	if controllerutil.RemoveFinalizer(wl, kueue.ResourceInUseFinalizerName) {
		return c.Update(ctx, wl)
	}
	return nil
}

func ReportEvictedWorkload(recorder record.EventRecorder, wl *kueue.Workload, cqName kueue.ClusterQueueReference, reason, message string) {
	metrics.ReportEvictedWorkloads(cqName, reason)
	if features.Enabled(features.LocalQueueMetrics) {
		metrics.ReportLocalQueueEvictedWorkloads(metrics.LQRefFromWorkload(wl), reason)
	}
	recorder.Event(wl, corev1.EventTypeNormal, fmt.Sprintf("%sDueTo%s", kueue.WorkloadEvicted, reason), message)
}

func References(wls []*Info) []klog.ObjectRef {
	if len(wls) == 0 {
		return nil
	}
	keys := make([]klog.ObjectRef, len(wls))
	for i, wl := range wls {
		keys[i] = klog.KObj(wl.Obj)
	}
	return keys
}

func WorkloadEvictionStateInc(wl *kueue.Workload, reason, underlyingCause string) bool {
	evictionState := FindSchedulingStatsEvictionByReason(wl, reason, underlyingCause)
	if evictionState == nil {
		evictionState = &kueue.WorkloadSchedulingStatsEviction{
			Reason:          reason,
			UnderlyingCause: underlyingCause,
		}
	}
	report := evictionState.Count == 0
	evictionState.Count++
	SetSchedulingStatsEviction(wl, *evictionState)
	return report
}

func FindSchedulingStatsEvictionByReason(wl *kueue.Workload, reason, underlyingCause string) *kueue.WorkloadSchedulingStatsEviction {
	if wl.Status.SchedulingStats != nil {
		for i := range wl.Status.SchedulingStats.Evictions {
			if wl.Status.SchedulingStats.Evictions[i].Reason == reason && wl.Status.SchedulingStats.Evictions[i].UnderlyingCause == underlyingCause {
				return &wl.Status.SchedulingStats.Evictions[i]
			}
		}
	}
	return nil
}

func SetSchedulingStatsEviction(wl *kueue.Workload, newEvictionState kueue.WorkloadSchedulingStatsEviction) bool {
	if wl.Status.SchedulingStats == nil {
		wl.Status.SchedulingStats = &kueue.SchedulingStats{}
	}
	evictionState := FindSchedulingStatsEvictionByReason(wl, newEvictionState.Reason, newEvictionState.UnderlyingCause)
	if evictionState == nil {
		wl.Status.SchedulingStats.Evictions = append(wl.Status.SchedulingStats.Evictions, newEvictionState)
		return true
	}
	if evictionState.Count != newEvictionState.Count {
		evictionState.Count = newEvictionState.Count
		return true
	}
	return false
}

// IsAdmitted 返回工作负载是否 准入。
func IsAdmitted(w *kueue.Workload) bool {
	return apimeta.IsStatusConditionTrue(w.Status.Conditions, kueue.WorkloadAdmitted)
}

// BaseSSAWorkload 会根据输入的工作负载创建一个新的对象，该对象仅包含识别原始对象所必需的字段。此对象可用于作为服务器端应用的基础。
func BaseSSAWorkload(w *kueue.Workload) *kueue.Workload {
	wlCopy := &kueue.Workload{
		ObjectMeta: metav1.ObjectMeta{
			UID:         w.UID,
			Name:        w.Name,
			Namespace:   w.Namespace,
			Generation:  w.Generation, // 如果规范发生变化，则产生冲突。
			Annotations: maps.Clone(w.Annotations),
			Labels:      maps.Clone(w.Labels),
		},
		TypeMeta: w.TypeMeta,
	}
	if wlCopy.APIVersion == "" {
		wlCopy.APIVersion = kueue.GroupVersion.String()
	}
	if wlCopy.Kind == "" {
		wlCopy.Kind = "Workload"
	}
	return wlCopy
}

// UpdateReclaimablePods 使用 SSA 更新工作负载的 ReclaimablePods 列表。
func UpdateReclaimablePods(ctx context.Context, c client.Client, w *kueue.Workload, reclaimablePods []kueue.ReclaimablePod) error {
	patch := BaseSSAWorkload(w)
	patch.Status.ReclaimablePods = reclaimablePods
	return c.Status().Patch(ctx, patch, client.Apply, client.FieldOwner(constants.ReclaimablePodsMgr))
}

func NewInfo(w *kueue.Workload, opts ...InfoOption) *Info {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	info := &Info{
		Obj: w,
	}
	if w.Status.Admission != nil {
		info.ClusterQueue = w.Status.Admission.ClusterQueue
		info.TotalRequests = totalRequestsFromAdmission(w) // ✅
	} else {
		info.TotalRequests = totalRequestsFromPodSets(w, &options) // ✅
	}
	return info
}

func totalRequestsFromPodSets(wl *kueue.Workload, info *InfoOptions) []PodSetResources {
	if len(wl.Spec.PodSets) == 0 {
		return nil
	}
	res := make([]PodSetResources, 0, len(wl.Spec.PodSets))
	currentCounts := podSetsCountsAfterReclaim(wl) // ✅
	for _, ps := range wl.Spec.PodSets {
		count := currentCounts[ps.Name]
		setRes := PodSetResources{
			Name:  ps.Name,
			Count: count,
		}
		specRequests := resourcehelpers.PodRequests(&corev1.Pod{Spec: ps.Template.Spec}, resourcehelpers.PodResourcesOptions{})
		effectiveRequests := dropExcludedResources(specRequests, info.excludedResourcePrefixes) // 实际上的
		if features.Enabled(features.ConfigurableResourceTransformations) {
			effectiveRequests = applyResourceTransformations(effectiveRequests, info.resourceTransformations)
		}
		setRes.Requests = resources.NewRequests(effectiveRequests)
		setRes.Requests.Mul(int64(count))
		res = append(res, setRes)
	}
	return res
}

func podSetsCountsAfterReclaim(wl *kueue.Workload) map[kueue.PodSetReference]int32 {
	totalCounts := podSetsCounts(wl)       // ✅
	reclaimCounts := reclaimableCounts(wl) // ✅
	for podSetName := range totalCounts {
		if rc, found := reclaimCounts[podSetName]; found {
			totalCounts[podSetName] -= rc
		}
	}
	return totalCounts
}
func podSetsCounts(wl *kueue.Workload) map[kueue.PodSetReference]int32 {
	return utilslices.ToMap(wl.Spec.PodSets, func(i int) (kueue.PodSetReference, int32) {
		return wl.Spec.PodSets[i].Name, wl.Spec.PodSets[i].Count
	})
}
func reclaimableCounts(wl *kueue.Workload) map[kueue.PodSetReference]int32 {
	return utilslices.ToMap(wl.Status.ReclaimablePods, func(i int) (kueue.PodSetReference, int32) {
		return wl.Status.ReclaimablePods[i].Name, wl.Status.ReclaimablePods[i].Count
	})
}

func dropExcludedResources(input corev1.ResourceList, excludedPrefixes []string) corev1.ResourceList {
	res := corev1.ResourceList{}
	for inputName, inputQuantity := range input {
		exclude := false
		for _, excludedPrefix := range excludedPrefixes {
			if strings.HasPrefix(string(inputName), excludedPrefix) {
				exclude = true
				break
			}
		}
		if !exclude {
			res[inputName] = inputQuantity
		}
	}
	return res
}

func applyResourceTransformations(input corev1.ResourceList, transforms map[corev1.ResourceName]*config.ResourceTransformation) corev1.ResourceList {
	match := false
	for resourceName := range input {
		if _, ok := transforms[resourceName]; ok {
			match = true
			break
		}
	}
	if !match {
		return input
	}
	output := make(corev1.ResourceList)
	for inputName, inputQuantity := range input {
		if mapping, ok := transforms[inputName]; ok {
			for outputName, baseFactor := range mapping.Outputs {
				outputQuantity := baseFactor.DeepCopy()
				outputQuantity.Mul(inputQuantity.Value())
				if accumulated, ok := output[outputName]; ok {
					outputQuantity.Add(accumulated) // 这里也有规格的意思
				}
				output[outputName] = outputQuantity
			}
			if ptr.Deref(mapping.Strategy, config.Retain) == config.Retain {
				output[inputName] = inputQuantity
			}
		} else {
			output[inputName] = inputQuantity
		}
	}
	return output
}
func totalRequestsFromAdmission(wl *kueue.Workload) []PodSetResources {
	if wl.Status.Admission == nil {
		return nil
	}
	res := make([]PodSetResources, 0, len(wl.Spec.PodSets))
	currentCounts := podSetsCountsAfterReclaim(wl) // ✅
	totalCounts := podSetsCounts(wl)
	for _, psa := range wl.Status.Admission.PodSetAssignments {
		setRes := PodSetResources{
			Name:     psa.Name,
			Flavors:  psa.Flavors,
			Count:    ptr.Deref(psa.Count, totalCounts[psa.Name]),
			Requests: resources.NewRequests(psa.ResourceUsage),
		}
		if features.Enabled(features.TopologyAwareScheduling) && psa.TopologyAssignment != nil {
			setRes.TopologyRequest = &TopologyRequest{
				Levels: psa.TopologyAssignment.Levels,
			}
			for _, domain := range psa.TopologyAssignment.Domains {
				setRes.TopologyRequest.DomainRequests = append(setRes.TopologyRequest.DomainRequests, TopologyDomainRequests{
					Values:            domain.Values,
					SinglePodRequests: setRes.SinglePodRequests(),
					Count:             domain.Count,
				})
			}
		}
		if features.Enabled(features.TopologyAwareScheduling) && psa.DelayedTopologyRequest != nil {
			setRes.DelayedTopologyRequest = ptr.To(*psa.DelayedTopologyRequest)
		}

		// 如果 countAfterReclaim 低于 admission 计数，则表示
		// 额外的 pod 被标记为可回收，消耗应向下缩放。
		if countAfterReclaim := currentCounts[psa.Name]; countAfterReclaim < setRes.Count {
			setRes.Requests.Divide(int64(setRes.Count))
			setRes.Requests.Mul(int64(countAfterReclaim))
			setRes.Count = countAfterReclaim
		}
		// 否则，如果 countAfterReclaim 更高，则表示 podSet 部分已准入
		// 并且计数应保持不变。
		res = append(res, setRes)
	}
	return res
}

// ScaledTo 按份数调整资源   副本数调整
func (p *PodSetResources) ScaledTo(newCount int32) *PodSetResources {
	if p.TopologyRequest != nil {
		return p
	}
	ret := &PodSetResources{
		Name:     p.Name,
		Requests: maps.Clone(p.Requests),
		Count:    p.Count,
		Flavors:  maps.Clone(p.Flavors),
	}

	if p.Count != 0 && p.Count != newCount {
		ret.Requests.Divide(int64(ret.Count))
		ret.Requests.Mul(int64(newCount))
		ret.Count = newCount
	}
	return ret
}

// NextFlavorToTryForPodSetResource 下一个要尝试的用于 PodSet 资源的 flavor
func (s *AssignmentClusterQueueState) NextFlavorToTryForPodSetResource(ps int, res corev1.ResourceName) int {
	if !features.Enabled(features.FlavorFungibility) {
		return 0
	}
	if s == nil || ps >= len(s.LastTriedFlavorIdx) {
		return 0
	}
	idx, ok := s.LastTriedFlavorIdx[ps][res]
	if !ok {
		return 0
	}
	return idx + 1
}

func (i *Info) CanBePartiallyAdmitted() bool {
	return CanBePartiallyAdmitted(i.Obj) // ✅
}

func CanBePartiallyAdmitted(wl *kueue.Workload) bool {
	ps := wl.Spec.PodSets
	for psi := range ps {
		if ps[psi].Count > ptr.Deref(ps[psi].MinCount, ps[psi].Count) {
			return true
		}
	}
	return false
}

// IsRequestingTAS 返回工作负载是否请求 TAS
func (i *Info) IsRequestingTAS() bool {
	return slices.ContainsFunc(i.Obj.Spec.PodSets,
		func(ps kueue.PodSet) bool {
			return ps.TopologyRequest != nil
		})
}

func HasTopologyAssignmentWithNodeToReplace(w *kueue.Workload) bool {
	if !HasNodeToReplace(w) || !IsAdmitted(w) {
		return false
	}
	annotations := w.GetAnnotations()
	failedNode := annotations[kueuealpha.NodeToReplaceAnnotation]
	for _, psa := range w.Status.Admission.PodSetAssignments {
		if psa.TopologyAssignment == nil {
			continue
		}
		for _, domain := range psa.TopologyAssignment.Domains {
			if domain.Values[len(domain.Values)-1] == failedNode {
				return true
			}
		}
	}
	return false
}

// AdmissionChecksForWorkload 返回应分配给特定工作负载的 AdmissionChecks，
// 基于 ClusterQueue 配置和 ResourceFlavors。
func AdmissionChecksForWorkload(log logr.Logger, wl *kueue.Workload, admissionChecks map[kueue.AdmissionCheckReference]sets.Set[kueue.ResourceFlavorReference]) sets.Set[kueue.AdmissionCheckReference] {
	// 如果所有 admissionChecks 都应针对所有 flavor 运行，则不需要等待 Workload 的 Admission 被设置。
	// 这也是 admissionChecks 通过 ClusterQueue.Spec.AdmissionChecks 指定的情况，而不是
	// ClusterQueue.Spec.AdmissionCheckStrategy
	allFlavors := true
	for _, flavors := range admissionChecks {
		if len(flavors) != 0 {
			allFlavors = false
		}
	}
	if allFlavors {
		return sets.New(slices.Collect(maps.Keys(admissionChecks))...)
	}

	// Kueue 首先根据 ClusterQueue 配置设置 AdmissionChecks，此时 Workload 没有
	// 分配 ResourceFlavors，因此我们无法将 AdmissionChecks 与 ResourceFlavor 匹配。
	// 在配额保留后，另一个重新协调发生，我们可以将 AdmissionChecks 与 ResourceFlavors 匹配
	if wl.Status.Admission == nil {
		log.V(2).Info("Workload has no Admission", "Workload", klog.KObj(wl))
		return nil
	}

	var assignedFlavors []kueue.ResourceFlavorReference
	for _, podSet := range wl.Status.Admission.PodSetAssignments {
		for _, flavor := range podSet.Flavors {
			assignedFlavors = append(assignedFlavors, flavor)
		}
	}

	acNames := sets.New[kueue.AdmissionCheckReference]()
	for acName, flavors := range admissionChecks {
		if len(flavors) == 0 {
			acNames.Insert(acName)
			continue
		}
		for _, fName := range assignedFlavors {
			if flavors.Has(fName) {
				acNames.Insert(acName)
			}
		}
	}
	return acNames
}

// ApplyAdmissionStatus 使用 SSA 更新工作负载的所有准入相关状态字段。
// 如果 strict 为 true，则资源版本将是补丁的一部分，如果 Workload
// 已更改，则使此调用失败。
func ApplyAdmissionStatus(ctx context.Context, c client.Client, w *kueue.Workload, strict bool, clk clock.Clock) error {
	wlCopy := PrepareWorkloadPatch(w, strict, clk)
	return ApplyAdmissionStatusPatch(ctx, c, wlCopy)
}

func PrepareWorkloadPatch(w *kueue.Workload, strict bool, clk clock.Clock) *kueue.Workload {
	wlCopy := BaseSSAWorkload(w)
	AdmissionStatusPatch(w, wlCopy, strict)
	AdmissionChecksStatusPatch(w, wlCopy, clk)
	return wlCopy
}

// AdmissionStatusPatch 创建一个基于输入工作负载的新对象，其中包含
// 准入和相关条件。该对象可用于服务器端应用。
// 如果 strict 为 true，则资源版本将是补丁的一部分。
func AdmissionStatusPatch(w *kueue.Workload, wlCopy *kueue.Workload, strict bool) {
	wlCopy.Status.Admission = w.Status.Admission.DeepCopy()
	wlCopy.Status.RequeueState = w.Status.RequeueState.DeepCopy()
	if wlCopy.Status.Admission != nil {
		// 清除 ResourceRequests；Assignment.PodSetAssignment[].ResourceUsage 覆盖它
		wlCopy.Status.ResourceRequests = []kueue.PodSetRequest{}
	} else {
		for _, rr := range w.Status.ResourceRequests {
			wlCopy.Status.ResourceRequests = append(wlCopy.Status.ResourceRequests, *rr.DeepCopy())
		}
	}
	for _, conditionName := range admissionManagedConditions {
		if existing := apimeta.FindStatusCondition(w.Status.Conditions, conditionName); existing != nil {
			wlCopy.Status.Conditions = append(wlCopy.Status.Conditions, *existing.DeepCopy())
		}
	}
	if strict {
		wlCopy.ResourceVersion = w.ResourceVersion
	}
	wlCopy.Status.AccumulatedPastExexcutionTimeSeconds = w.Status.AccumulatedPastExexcutionTimeSeconds
	if w.Status.SchedulingStats != nil {
		if wlCopy.Status.SchedulingStats == nil {
			wlCopy.Status.SchedulingStats = &kueue.SchedulingStats{}
		}
		wlCopy.Status.SchedulingStats.Evictions = append(wlCopy.Status.SchedulingStats.Evictions, w.Status.SchedulingStats.Evictions...)
	}
}
func AdmissionChecksStatusPatch(w *kueue.Workload, wlCopy *kueue.Workload, c clock.Clock) {
	if wlCopy.Status.AdmissionChecks == nil && w.Status.AdmissionChecks != nil {
		wlCopy.Status.AdmissionChecks = make([]kueue.AdmissionCheckState, 0)
	}
	for _, ac := range w.Status.AdmissionChecks {
		SetAdmissionCheckState(&wlCopy.Status.AdmissionChecks, ac, c)
	}
}

// ApplyAdmissionStatusPatch 应用工作负载准入相关状态字段的补丁，使用 SSA。
func ApplyAdmissionStatusPatch(ctx context.Context, c client.Client, patch *kueue.Workload) error {
	return c.Status().Patch(ctx, patch, client.Apply, client.FieldOwner(constants.AdmissionName), client.ForceOwnership)
}

// HasQuotaReservation 检查工作负载是否基于条件进行准入。
func HasQuotaReservation(w *kueue.Workload) bool {
	return apimeta.IsStatusConditionTrue(w.Status.Conditions, kueue.WorkloadQuotaReserved)
}
