package jobframework

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/utils/clock"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	configapi "sigs.k8s.io/kueue/apis/config/v1beta1"
	kueuealpha "sigs.k8s.io/kueue/apis/kueue/v1alpha1"
	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/cache"
	"sigs.k8s.io/kueue/pkg/constants"
	controllerconsts "sigs.k8s.io/kueue/pkg/controller/constants"
	"sigs.k8s.io/kueue/pkg/controller/core/indexer"
	"sigs.k8s.io/kueue/pkg/features"
	"sigs.k8s.io/kueue/pkg/metrics"
	"sigs.k8s.io/kueue/pkg/podset"
	"sigs.k8s.io/kueue/pkg/queue"
	clientutil "sigs.k8s.io/kueue/pkg/util/client"
	"sigs.k8s.io/kueue/pkg/util/equality"
	"sigs.k8s.io/kueue/pkg/util/kubeversion"
	"sigs.k8s.io/kueue/pkg/util/maps"
	utilpriority "sigs.k8s.io/kueue/pkg/util/priority"
	"sigs.k8s.io/kueue/pkg/util/slices"
	"sigs.k8s.io/kueue/pkg/workload"
)

const (
	FailedToStartFinishedReason = "FailedToStart"
	managedOwnersChainLimit     = 10
)

var (
	ErrCyclicOwnership                = errors.New("cyclic ownership")
	ErrWorkloadOwnerNotFound          = errors.New("workload owner not found")
	ErrManagedOwnersChainLimitReached = errors.New("managed owner chain limit reached")
	ErrNoMatchingWorkloads            = errors.New("no matching workloads")
	ErrExtraWorkloads                 = errors.New("extra workloads")
	ErrPrebuiltWorkloadNotFound       = errors.New("prebuilt workload not found")
)

type WorkloadRetentionPolicy struct {
	AfterDeactivatedByKueue *time.Duration
}

// JobReconciler 用于调谐（管理）一个通用的 GenericJob 对象
type JobReconciler struct {
	client                       client.Client
	record                       record.EventRecorder
	manageJobsWithoutQueueName   bool
	managedJobsNamespaceSelector labels.Selector
	waitForPodsReady             bool
	labelKeysToCopy              []string
	clock                        clock.Clock
	workloadRetentionPolicy      WorkloadRetentionPolicy
}

type Options struct {
	ManageJobsWithoutQueueName   bool
	ManagedJobsNamespaceSelector labels.Selector
	WaitForPodsReady             bool
	KubeServerVersion            *kubeversion.ServerVersionFetcher
	IntegrationOptions           map[string]any // IntegrationOptions key is "$GROUP/$VERSION, Kind=$KIND".
	EnabledFrameworks            sets.Set[string]
	EnabledExternalFrameworks    sets.Set[string]
	ManagerName                  string
	LabelKeysToCopy              []string
	Queues                       *queue.Manager
	Cache                        *cache.Cache
	Clock                        clock.Clock
	WorkloadRetentionPolicy      WorkloadRetentionPolicy
}

// Option 用于配置调谐器的选项。
type Option func(*Options)

func ProcessOptions(opts ...Option) Options {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	return options
}

// WithWaitForPodsReady 指示控制器是否应在对应的 job 所有 pod 就绪或成功时，向 workload 添加 PodsReady 条件。
func WithWaitForPodsReady(w *configapi.WaitForPodsReady) Option {
	return func(o *Options) {
		o.WaitForPodsReady = w != nil && w.Enable
	}
}

func WithKubeServerVersion(v *kubeversion.ServerVersionFetcher) Option {
	return func(o *Options) {
		o.KubeServerVersion = v
	}
}

// WithIntegrationOptions 添加集成选项，如 podOptions。
// 第二个参数 `opts` 应为任意选项结构体。
func WithIntegrationOptions(integrationName string, opts any) Option {
	return func(o *Options) {
		if len(o.IntegrationOptions) == 0 {
			o.IntegrationOptions = make(map[string]any)
		}
		o.IntegrationOptions[integrationName] = opts
	}
}

// WithEnabledFrameworks 添加在 ConfigAPI 中启用的框架名称。
func WithEnabledFrameworks(frameworks []string) Option {
	return func(o *Options) {
		if len(frameworks) == 0 {
			return
		}
		o.EnabledFrameworks = sets.New(frameworks...)
	}
}

// WithEnabledExternalFrameworks 添加由外部控制器管理的框架名称（在 Config API 中）。
func WithEnabledExternalFrameworks(exFrameworks []string) Option {
	return func(o *Options) {
		if len(exFrameworks) == 0 {
			return
		}
		o.EnabledExternalFrameworks = sets.New(exFrameworks...)
	}
}

// WithManagerName 设置 kueue 的 manager 名称。
func WithManagerName(n string) Option {
	return func(o *Options) {
		o.ManagerName = n
	}
}

// WithLabelKeysToCopy 设置需要复制的 label key。
func WithLabelKeysToCopy(n []string) Option {
	return func(o *Options) {
		o.LabelKeysToCopy = n
	}
}

// WithQueues 设置队列管理器。
func WithQueues(q *queue.Manager) Option {
	return func(o *Options) {
		o.Queues = q
	}
}

// WithCache 设置缓存管理器。
func WithCache(c *cache.Cache) Option {
	return func(o *Options) {
		o.Cache = c
	}
}

// WithClock 设置调谐器的时钟。默认使用系统时钟，仅建议在测试时更改。
func WithClock(c clock.Clock) Option {
	return func(o *Options) {
		o.Clock = c
	}
}

var defaultOptions = Options{
	Clock: clock.RealClock{},
}

func EnsurePrebuiltWorkloadOwnership(ctx context.Context, c client.Client, wl *kueue.Workload, object client.Object) error {
	if !metav1.IsControlledBy(wl, object) {
		if err := ctrl.SetControllerReference(object, wl, c.Scheme()); err != nil {
			return err
		}

		if errs := validation.IsValidLabelValue(string(object.GetUID())); len(errs) == 0 {
			wl.Labels = maps.MergeKeepFirst(map[string]string{controllerconsts.JobUIDLabel: string(object.GetUID())}, wl.Labels)
		}

		if err := c.Update(ctx, wl); err != nil {
			return err
		}
	}
	return nil
}

// TODO 需要后看
// expectedRunningPodSets 获取作业执行期间期望的 podsets，如果 workload 没有配额预留或准入不匹配则返回 nil。
func expectedRunningPodSets(ctx context.Context, c client.Client, wl *kueue.Workload) []kueue.PodSet {
	if !workload.HasQuotaReservation(wl) {
		return nil
	}
	info, err := getPodSetsInfoFromStatus(ctx, c, wl)
	if err != nil {
		return nil
	}
	infoMap := slices.ToRefMap(info, func(psi *podset.PodSetInfo) kueue.PodSetReference { return psi.Name })
	runningPodSets := wl.Spec.DeepCopy().PodSets
	canBePartiallyAdmitted := workload.CanBePartiallyAdmitted(wl)
	for i := range runningPodSets {
		ps := &runningPodSets[i]
		psi, found := infoMap[ps.Name]
		if !found {
			return nil
		}
		err := podset.Merge(&ps.Template.ObjectMeta, &ps.Template.Spec, *psi)
		if err != nil {
			return nil
		}
		if canBePartiallyAdmitted && ps.MinCount != nil {
			// update the expected running count
			ps.Count = psi.Count
		}
	}
	return runningPodSets
}

// startJob 会取消挂起 job，并注入节点亲和性。
func (r *JobReconciler) startJob(ctx context.Context, jobOrPod GenericJob, object client.Object, wl *kueue.Workload) error {
	info, err := getPodSetsInfoFromStatus(ctx, r.client, wl)
	if err != nil {
		return err
	}
	msg := fmt.Sprintf("Admitted by clusterQueue %v", wl.Status.Admission.ClusterQueue)

	if cj, implements := jobOrPod.(ComposablePod); implements {
		if err := cj.Run(ctx, r.client, info, r.record, msg); err != nil {
			return err
		}
	} else {
		if err := clientutil.Patch(ctx, r.client, object, true, func() (bool, error) {
			return true, jobOrPod.RunWithPodSetsInfo(info)
		}); err != nil {
			return err
		}
		r.record.Event(object, corev1.EventTypeNormal, ReasonStarted, msg)
	}

	return nil
}

// getPodSetsInfoFromStatus extracts podSetsInfo from workload status, based on
// admission, and admission checks.
// getPodSetsInfoFromStatus 根据准入和准入检查，从 workload 状态中提取 podSetsInfo。
func getPodSetsInfoFromStatus(ctx context.Context, c client.Client, w *kueue.Workload) ([]podset.PodSetInfo, error) {
	if len(w.Status.Admission.PodSetAssignments) == 0 {
		return nil, nil
	}

	podSetsInfo := make([]podset.PodSetInfo, len(w.Status.Admission.PodSetAssignments))

	for i, psAssignment := range w.Status.Admission.PodSetAssignments {
		info, err := podset.FromAssignment(ctx, c, &psAssignment, w.Spec.PodSets[i].Count)
		if err != nil {
			return nil, err
		}
		if features.Enabled(features.TopologyAwareScheduling) {
			info.Annotations[kueuealpha.WorkloadAnnotation] = w.Name
		}

		info.Labels[controllerconsts.PodSetLabel] = string(psAssignment.Name)

		for _, admissionCheck := range w.Status.AdmissionChecks {
			for _, podSetUpdate := range admissionCheck.PodSetUpdates {
				if podSetUpdate.Name == info.Name {
					if err := info.Merge(podset.FromUpdate(&podSetUpdate)); err != nil {
						return nil, fmt.Errorf("in admission check %q: %w", admissionCheck.Name, err)
					}
					break
				}
			}
		}
		podSetsInfo[i] = info
	}
	return podSetsInfo, nil
}

func (r *JobReconciler) ignoreUnretryableError(log logr.Logger, err error) error {
	if IsUnretryableError(err) {
		log.V(2).Info("Received an unretryable error", "error", err)
		return nil
	}
	return err
}

func generatePodsReadyCondition(log logr.Logger, jobOrPod GenericJob, wl *kueue.Workload, clock clock.Clock) metav1.Condition {
	const (
		notReadyMsg           = "Not all pods are ready or succeeded"
		waitingForRecoveryMsg = "At least one pod has failed, waiting for recovery"
		readyMsg              = "All pods reached readiness and the workload is running"
	)
	if !workload.IsAdmitted(wl) {
		// The workload has not been admitted yet
		// or it was admitted in the past but it's evicted/requeued
		return workload.CreatePodsReadyCondition(metav1.ConditionFalse,
			kueue.WorkloadWaitForStart,
			notReadyMsg,
			clock)
	}

	podsReadyCond := apimeta.FindStatusCondition(wl.Status.Conditions, kueue.WorkloadPodsReady)
	podsReady := jobOrPod.PodsReady()
	log.V(3).Info("Generating PodsReady condition",
		"Current PodsReady condition", podsReadyCond,
		"Pods are ready", podsReady)

	if podsReady {
		reason := kueue.WorkloadStarted
		if podsReadyCond != nil && (podsReadyCond.Reason == kueue.WorkloadWaitForRecovery || podsReadyCond.Reason == kueue.WorkloadRecovered) {
			reason = kueue.WorkloadRecovered
		}
		return workload.CreatePodsReadyCondition(metav1.ConditionTrue,
			reason,
			readyMsg,
			clock)
	}

	switch {
	case podsReadyCond == nil:
		return workload.CreatePodsReadyCondition(metav1.ConditionFalse,
			kueue.WorkloadWaitForStart,
			notReadyMsg,
			clock)

	case podsReadyCond.Status == metav1.ConditionTrue:
		return workload.CreatePodsReadyCondition(metav1.ConditionFalse,
			kueue.WorkloadWaitForRecovery,
			waitingForRecoveryMsg,
			clock)

	case podsReadyCond.Reason == kueue.WorkloadWaitForRecovery:
		return workload.CreatePodsReadyCondition(metav1.ConditionFalse,
			kueue.WorkloadWaitForRecovery,
			waitingForRecoveryMsg,
			clock)

	default:
		// handles both "WaitForPodsStart" and the old "PodsReady" reasons
		return workload.CreatePodsReadyCondition(metav1.ConditionFalse,
			kueue.WorkloadWaitForStart,
			notReadyMsg,
			clock)
	}
}

type ReconcilerSetup func(*builder.Builder, client.Client) *builder.Builder

type genericReconciler struct {
	jr     *JobReconciler
	newJob func() GenericJob
	setup  []ReconcilerSetup
}

// clearMinCountsIfFeatureDisabled 如果未启用 PartialAdmission 特性，则将所有 podSet 的 minCount 设为 nil。
func clearMinCountsIfFeatureDisabled(in []kueue.PodSet) []kueue.PodSet {
	if features.Enabled(features.PartialAdmission) || len(in) == 0 {
		return in
	}
	for i := range in {
		in[i].MinCount = nil
	}
	return in
}

// WithManageJobsWithoutQueueName 指示控制器是否应调谐不设置队列名称注解的作业。
func WithManageJobsWithoutQueueName(f bool) Option {
	return func(o *Options) {
		o.ManageJobsWithoutQueueName = f
	}
}

// WithManagedJobsNamespaceSelector 用于基于命名空间的 ManagedJobsWithoutQueueName 过滤。
func WithManagedJobsNamespaceSelector(ls labels.Selector) Option {
	return func(o *Options) {
		o.ManagedJobsNamespaceSelector = ls
	}
}

// FindAncestorJobManagedByKueue 遍历 controllerRefs 以查找 Kueue 管理的顶级祖先 Job。
// 如果 manageJobsWithoutQueueName 设置为 false，则仅返回带有队列名称的作业。
// 如果 manageJobsWithoutQueueName 设置为 true，则可能返回没有队列名称的作业。
//
// 示例：
//
// With manageJobsWithoutQueueName=false:
// Job -> JobSet -> AppWrapper => nil
// Job (queue-name) -> JobSet (queue-name) -> AppWrapper => JobSet
// Job (queue-name) -> JobSet -> AppWrapper (queue-name) => AppWrapper
// Job (queue-name) -> JobSet (queue-name) -> AppWrapper (queue-name) => AppWrapper
// Job -> JobSet (disabled) -> AppWrapper (queue-name) => AppWrapper
//
// With manageJobsWithoutQueueName=true:
// Job -> JobSet -> AppWrapper => AppWrapper
// Job (queue-name) -> JobSet (queue-name) -> AppWrapper => AppWrapper
// Job (queue-name) -> JobSet -> AppWrapper (queue-name) => AppWrapper
// Job (queue-name) -> JobSet (queue-name) -> AppWrapper (queue-name) => AppWrapper
// Job -> JobSet (disabled) -> AppWrapper => AppWrapper
func FindAncestorJobManagedByKueue(ctx context.Context, c client.Client, jobObj client.Object, manageJobsWithoutQueueName bool) (client.Object, error) {
	log := ctrl.LoggerFrom(ctx)
	seen := sets.New[types.UID]()
	currentObj := jobObj

	var topLevelJob client.Object
	for {
		if seen.Has(currentObj.GetUID()) {
			log.Error(ErrCyclicOwnership,
				"Terminated search for Kueue-managed Job because of cyclic ownership",
				"owner", currentObj,
			)
			return nil, ErrCyclicOwnership
		}
		seen.Insert(currentObj.GetUID())

		owner := metav1.GetControllerOf(currentObj)
		if owner == nil {
			log.V(3).Info("stop walking up as the owner is not found", "owner", klog.KObj(currentObj))
			return topLevelJob, nil
		}

		if !manager.isKnownOwner(owner) {
			log.V(3).Info("stop walking up as the owner is not known", "owner", klog.KObj(currentObj))
			return topLevelJob, nil
		}
		parentObj := getEmptyOwnerObject(owner)
		managed := parentObj != nil
		if parentObj == nil {
			parentObj = &metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: owner.APIVersion,
					Kind:       owner.Kind,
				},
			}
		}
		if err := c.Get(ctx, client.ObjectKey{Name: owner.Name, Namespace: jobObj.GetNamespace()}, parentObj); err != nil {
			return nil, errors.Join(ErrWorkloadOwnerNotFound, err)
		}
		if managed && (manageJobsWithoutQueueName || QueueNameForObject(parentObj) != "") {
			topLevelJob = parentObj
		}
		currentObj = parentObj
		if len(seen) > managedOwnersChainLimit {
			return nil, ErrManagedOwnersChainLimitReached
		}
	}
}

// NewGenericReconcilerFactory 创建一个新的通用作业类型调谐器工厂。
// newJob 应返回一个新的空作业。
func NewGenericReconcilerFactory(newJob func() GenericJob, setup ...ReconcilerSetup) ReconcilerFactory {
	return func(client client.Client, record record.EventRecorder, opts ...Option) JobReconcilerInterface {
		return &genericReconciler{
			jr:     NewReconciler(client, record, opts...),
			newJob: newJob,
			setup:  setup,
		}
	}
}
func NewReconciler(
	client client.Client,
	record record.EventRecorder,
	opts ...Option) *JobReconciler {
	options := ProcessOptions(opts...)

	return &JobReconciler{
		client:                       client,
		record:                       record,
		manageJobsWithoutQueueName:   options.ManageJobsWithoutQueueName,
		managedJobsNamespaceSelector: options.ManagedJobsNamespaceSelector,
		waitForPodsReady:             options.WaitForPodsReady,
		labelKeysToCopy:              options.LabelKeysToCopy,
		clock:                        options.Clock,
		workloadRetentionPolicy:      options.WorkloadRetentionPolicy,
	}
}
func (r *genericReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.jr.ReconcileGenericJob(ctx, req, r.newJob())
}
func (r *genericReconciler) SetupWithManager(mgr ctrl.Manager) error {
	b := ctrl.NewControllerManagedBy(mgr).
		For(r.newJob().Object()).Owns(&kueue.Workload{})
	c := mgr.GetClient()
	for _, f := range r.setup {
		b = f(b, c)
	}
	return b.Complete(r)
}

// getWorkloadForObject 返回与给定 job 关联的 Workload。
func (r *JobReconciler) getWorkloadForObject(ctx context.Context, jobObj client.Object) (*kueue.Workload, error) { // 获取与 pod\job 相关联的 workload
	wlList := kueue.WorkloadList{}
	if err := r.client.List(ctx, &wlList, client.InNamespace(jobObj.GetNamespace()), client.MatchingFields{indexer.OwnerReferenceUID: string(jobObj.GetUID())}); client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	if len(wlList.Items) == 0 {
		return nil, nil
	}

	// 理论上 job 可能拥有多个 Workload，这里仅做日志记录。
	if len(wlList.Items) > 1 {
		ctrl.LoggerFrom(ctx).V(2).Info(
			"WARNING: The job has multiple associated Workloads",
			"job", klog.KObj(jobObj),
			"workloads", klog.KObjSlice(wlList.Items),
		)
	}

	return &wlList.Items[0], nil
}

// EquivalentToWorkload 检查 job 是否与 workload 对应
func EquivalentToWorkload(ctx context.Context, c client.Client, jobOrPod GenericJob, wl *kueue.Workload) (bool, error) {
	owner := metav1.GetControllerOf(wl)
	// Indexes don't work in unit tests, so we explicitly check for the
	// owner here.
	if owner.Name != jobOrPod.Object().GetName() {
		return false, nil
	}

	defaultDuration := int32(-1)
	if ptr.Deref(wl.Spec.MaximumExecutionTimeSeconds, defaultDuration) != ptr.Deref(MaximumExecutionTimeSeconds(jobOrPod), defaultDuration) { // label
		return false, nil
	}

	getPodSets, err := jobOrPod.PodSets() // ✅
	if err != nil {
		return false, err
	}
	jobPodSets := clearMinCountsIfFeatureDisabled(getPodSets) // ✅

	if runningPodSets := expectedRunningPodSets(ctx, c, wl); runningPodSets != nil { // todo
		if equality.ComparePodSetSlices(jobPodSets, runningPodSets, workload.IsAdmitted(wl)) {
			return true, nil
		}
		// If the workload is admitted but the job is suspended, do the check
		// against the non-running info.
		// This might allow some violating jobs to pass equivalency checks, but their
		// workloads would be invalidated in the next sync after unsuspending.
		return jobOrPod.IsSuspended() && equality.ComparePodSetSlices(jobPodSets, wl.Spec.PodSets, workload.IsAdmitted(wl)), nil
	}

	return equality.ComparePodSetSlices(jobPodSets, wl.Spec.PodSets, workload.IsAdmitted(wl)), nil
}

func FindMatchingWorkloads(ctx context.Context, c client.Client, jobOrPod GenericJob) (match *kueue.Workload, toDelete []*kueue.Workload, err error) {
	object := jobOrPod.Object()

	workloads := &kueue.WorkloadList{} // ".metadata.ownerReferences[%s.%s]", ownerGVK.Group, ownerGVK.Kind
	if err := c.List(ctx, workloads, client.InNamespace(object.GetNamespace()),
		client.MatchingFields{GetOwnerKey(jobOrPod.GVK()): object.GetName()}); err != nil {
		return nil, nil, err
	}

	for i := range workloads.Items {
		w := &workloads.Items[i]
		isEquivalent, err := EquivalentToWorkload(ctx, c, jobOrPod, w) // 相同
		if err != nil {
			return nil, nil, err
		}
		if match == nil && isEquivalent {
			match = w
		} else {
			toDelete = append(toDelete, w)
		}
	}

	return match, toDelete, nil
}

// stopJob 会挂起 job，同时恢复节点亲和性，如有需要还会重置 job 状态。
// 返回是否执行了停止操作或错误。
func (r *JobReconciler) stopJob(ctx context.Context, jobOrPod GenericJob, wl *kueue.Workload, stopReason StopReason, eventMsg string) error {
	object := jobOrPod.Object()

	info := GetPodSetsInfoFromWorkload(wl)

	if jws, implements := jobOrPod.(JobWithCustomStop); implements { // 只有Job实现了
		stoppedNow, err := jws.Stop(ctx, r.client, info, stopReason, eventMsg)
		if stoppedNow {
			r.record.Event(object, corev1.EventTypeNormal, ReasonStopped, eventMsg)
		}
		return err
	}

	if jws, implements := jobOrPod.(ComposablePod); implements {
		reason := stopReason
		if stopReason == StopReasonWorkloadEvicted {
			if evCond := apimeta.FindStatusCondition(wl.Status.Conditions, kueue.WorkloadEvicted); evCond != nil && evCond.Status == metav1.ConditionTrue {
				reason = StopReason(string(StopReasonWorkloadEvicted) + "DueTo" + evCond.Reason)
			}
		}
		stoppedNow, err := jws.Stop(ctx, r.client, info, reason, eventMsg)
		for _, objStoppedNow := range stoppedNow {
			r.record.Event(objStoppedNow, corev1.EventTypeNormal, ReasonStopped, eventMsg)
		}
		return err
	}

	if jobOrPod.IsSuspended() {
		return nil
	}

	if err := clientutil.Patch(ctx, r.client, object, true, func() (bool, error) {
		jobOrPod.Suspend()
		if info != nil {
			jobOrPod.RestorePodSetsInfo(info)
		}
		return true, nil
	}); err != nil {
		return err
	}

	r.record.Event(object, corev1.EventTypeNormal, ReasonStopped, eventMsg)
	return nil
}

// GetPodSetsInfoFromWorkload 从给定 workload 的 spec 中获取 podSetsInfo 切片。
func GetPodSetsInfoFromWorkload(wl *kueue.Workload) []podset.PodSetInfo {
	if wl == nil {
		return nil
	}
	return slices.Map(wl.Spec.PodSets, podset.FromPodSet)
}

func (r *JobReconciler) updateWorkloadToMatchJob(ctx context.Context, jobOrPod GenericJob, object client.Object, wl *kueue.Workload) (*kueue.Workload, error) {
	newWl, err := r.constructWorkload(ctx, jobOrPod) // ✅
	if err != nil {
		return nil, fmt.Errorf("can't construct workload for update: %w", err)
	}
	err = r.prepareWorkload(ctx, jobOrPod, newWl) // ✅
	if err != nil {
		return nil, fmt.Errorf("can't construct workload for update: %w", err)
	}
	wl.Spec = newWl.Spec
	if err = r.client.Update(ctx, wl); err != nil {
		return nil, fmt.Errorf("updating existed workload: %w", err)
	}

	r.record.Eventf(object, corev1.EventTypeNormal, ReasonUpdatedWorkload, "Updated not matching Workload for suspended job: %v", klog.KObj(wl))
	return newWl, nil
}

// constructWorkload 会从对应的 job 派生出一个 workload。
func (r *JobReconciler) constructWorkload(ctx context.Context, jobOrPod GenericJob) (*kueue.Workload, error) {
	if cj, implements := jobOrPod.(ComposablePod); implements {
		wl, err := cj.ConstructComposableWorkload(ctx, r.client, r.record, r.labelKeysToCopy)
		if err != nil {
			return nil, err
		}
		return wl, nil
	}
	return ConstructWorkload(ctx, r.client, jobOrPod, r.labelKeysToCopy) // ✅
}

func ConstructWorkload(ctx context.Context, c client.Client, jobOrPod GenericJob, labelKeysToCopy []string) (*kueue.Workload, error) {
	log := ctrl.LoggerFrom(ctx)
	object := jobOrPod.Object()

	podSets, err := jobOrPod.PodSets() // ✅
	if err != nil {
		return nil, err
	}

	wl := NewWorkload(GetWorkloadNameForOwnerWithGVK(object.GetName(), object.GetUID(), jobOrPod.GVK()), object, podSets, labelKeysToCopy) // ✅

	if wl.Labels == nil {
		wl.Labels = make(map[string]string)
	}
	jobUID := string(jobOrPod.Object().GetUID())
	if errs := validation.IsValidLabelValue(jobUID); len(errs) == 0 {
		wl.Labels[controllerconsts.JobUIDLabel] = jobUID
	} else {
		log.V(2).Info(
			"Validation of the owner job UID label has failed. Creating workload without the label.",
			"ValidationErrors", errs,
			"LabelValue", jobUID,
		)
	}

	if err := ctrl.SetControllerReference(object, wl, c.Scheme()); err != nil {
		return nil, err
	}
	return wl, nil
}

// prepareWorkload 为构建的 workload 添加优先级信息。
func (r *JobReconciler) prepareWorkload(ctx context.Context, jobOrPod GenericJob, wl *kueue.Workload) error {
	priorityClassName, source, p, err := r.extractPriority(ctx, wl.Spec.PodSets, jobOrPod)
	if err != nil {
		return err
	}
	wl.Spec.PriorityClassName = priorityClassName
	wl.Spec.Priority = &p
	wl.Spec.PriorityClassSource = source

	wl.Spec.PodSets = clearMinCountsIfFeatureDisabled(wl.Spec.PodSets)

	return nil
}

func (r *JobReconciler) extractPriority(ctx context.Context, podSets []kueue.PodSet, jobOrPod GenericJob) (string, string, int32, error) {
	var customPriorityFunc func() string
	if jobWithPriorityClass, isImplemented := jobOrPod.(JobWithPriorityClass); isImplemented {
		customPriorityFunc = jobWithPriorityClass.PriorityClass
	}
	return ExtractPriority(ctx, r.client, jobOrPod.Object(), podSets, customPriorityFunc)
}

func ExtractPriority(ctx context.Context, c client.Client, jobOrPod client.Object, podSets []kueue.PodSet, customPriorityFunc func() string) (string, string, int32, error) {
	if workloadPriorityClass := WorkloadPriorityClassName(jobOrPod); len(workloadPriorityClass) > 0 {
		return utilpriority.GetPriorityFromWorkloadPriorityClass(ctx, c, workloadPriorityClass)
	}
	if customPriorityFunc != nil {
		return utilpriority.GetPriorityFromPriorityClass(ctx, c, customPriorityFunc())
	}
	return utilpriority.GetPriorityFromPriorityClass(ctx, c, extractPriorityFromPodSets(podSets))
}
func extractPriorityFromPodSets(podSets []kueue.PodSet) string {
	for _, podSet := range podSets {
		if len(podSet.Template.Spec.PriorityClassName) > 0 {
			return podSet.Template.Spec.PriorityClassName
		}
	}
	return ""
}

// ensureOneWorkload 会查询与 job 匹配的唯一 workload 并返回。
// 如果存在多个 workload，则应删除多余的。
// 返回的 workload 可能为 nil。
func (r *JobReconciler) ensureOneWorkload(ctx context.Context, jobOrPod GenericJob, object client.Object) (*kueue.Workload, error) {
	log := ctrl.LoggerFrom(ctx)

	if prebuiltWorkloadName, usePrebuiltWorkload := PrebuiltWorkloadFor(jobOrPod); usePrebuiltWorkload {
		wl := &kueue.Workload{}
		err := r.client.Get(ctx, types.NamespacedName{Name: prebuiltWorkloadName, Namespace: object.GetNamespace()}, wl)
		if err != nil {
			return nil, client.IgnoreNotFound(err)
		}
		// Ignore the workload is controlled by another object.
		if controlledBy := metav1.GetControllerOfNoCopy(wl); controlledBy != nil && controlledBy.UID != object.GetUID() {
			log.V(2).Info(
				"WARNING: The workload is already controlled by another object",
				"workload", klog.KObj(wl),
				"controlledBy", controlledBy,
			)
			return nil, nil
		}

		if cj, implements := jobOrPod.(ComposablePod); implements {
			err = cj.EnsureWorkloadOwnedByAllMembers(ctx, r.client, r.record, wl) // todo
		} else {
			err = EnsurePrebuiltWorkloadOwnership(ctx, r.client, wl, object) // todo
		}
		if err != nil {
			return nil, err
		}

		if inSync, err := r.ensurePrebuiltWorkloadInSync(ctx, wl, jobOrPod); !inSync || err != nil {
			return nil, err
		}
		return wl, nil
	}

	// Find a matching workload first if there is one.
	var toDelete []*kueue.Workload
	var match *kueue.Workload
	if cj, implements := jobOrPod.(ComposablePod); implements {
		var err error
		match, toDelete, err = cj.FindMatchingWorkloads(ctx, r.client, r.record)
		if err != nil {
			log.Error(err, "Composable jobOrPod is unable to find matching workloads")
			return nil, err
		}
	} else {
		var err error
		match, toDelete, err = FindMatchingWorkloads(ctx, r.client, jobOrPod)
		if err != nil {
			log.Error(err, "Unable to list child workloads")
			return nil, err
		}
	}

	var toUpdate *kueue.Workload
	if match == nil && len(toDelete) > 0 && jobOrPod.IsSuspended() && !workload.HasQuotaReservation(toDelete[0]) {
		toUpdate = toDelete[0]
		toDelete = toDelete[1:]
	}

	// If there is no matching workload and the jobOrPod is running, suspend it.
	// 如果没有匹配的 workload 且 jobOrPod 正在运行，则挂起 jobOrPod。
	if match == nil && !jobOrPod.IsSuspended() {
		log.V(2).Info("jobOrPod with no matching workload, suspending")
		var w *kueue.Workload
		if len(toDelete) == 1 {
			// The jobOrPod may have been modified and hence the existing workload
			// doesn't match the jobOrPod anymore. All bets are off if there are more
			// than one workload...
			w = toDelete[0]
		}

		if _, _, finished := jobOrPod.Finished(); !finished {
			var msg string
			if w == nil {
				msg = "Missing Workload; unable to restore pod templates"
			} else {
				msg = "No matching Workload; restoring pod templates according to existent Workload"
			}
			if err := r.stopJob(ctx, jobOrPod, w, StopReasonNoMatchingWorkload, msg); err != nil { // ✅
				return nil, fmt.Errorf("stopping jobOrPod with no matching workload: %w", err)
			}
		}
	}

	// 删除重复的 workload 实例。
	existedWls := 0
	for _, wl := range toDelete {
		wlKey := workload.Key(wl)
		err := workload.RemoveFinalizer(ctx, r.client, wl)
		if err != nil && !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to remove workload finalizer for: %w ", err)
		}

		err = r.client.Delete(ctx, wl)
		if err != nil && !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("deleting not matching workload: %w", err)
		}
		if err == nil {
			existedWls++
			r.record.Eventf(object, corev1.EventTypeNormal, ReasonDeletedWorkload,
				"Deleted not matching Workload: %v", wlKey)
		}
	}

	if existedWls != 0 {
		if match == nil {
			return nil, fmt.Errorf("%w: deleted %d workloads", ErrNoMatchingWorkloads, len(toDelete))
		}
		return nil, fmt.Errorf("%w: deleted %d workloads", ErrExtraWorkloads, len(toDelete))
	}

	if toUpdate != nil {
		return r.updateWorkloadToMatchJob(ctx, jobOrPod, object, toUpdate) // ✅
	}

	return match, nil
}

func (r *JobReconciler) handleJobWithNoWorkload(ctx context.Context, jobOrPod GenericJob, object client.Object) error {
	log := ctrl.LoggerFrom(ctx)

	_, usePrebuiltWorkload := PrebuiltWorkloadFor(jobOrPod)
	if usePrebuiltWorkload {
		// Stop the job if not already suspended
		if stopErr := r.stopJob(ctx, jobOrPod, nil, StopReasonNoMatchingWorkload, "missing workload"); stopErr != nil {
			return stopErr
		}
	}

	// Wait until there are no active pods.
	if jobOrPod.IsActive() {
		log.V(2).Info("Job is suspended but still has active pods, waiting")
		return nil
	}

	if usePrebuiltWorkload {
		return ErrPrebuiltWorkloadNotFound
	}

	// Create the corresponding workload.
	wl, err := r.constructWorkload(ctx, jobOrPod) // ✅
	if err != nil {
		return err
	}
	err = r.prepareWorkload(ctx, jobOrPod, wl) // ✅
	if err != nil {
		return err
	}
	if err = r.client.Create(ctx, wl); err != nil { // ✅
		return err
	}
	r.record.Eventf(object, corev1.EventTypeNormal, ReasonCreatedWorkload, "Created Workload: %v", workload.Key(wl))
	return nil
}
func (r *JobReconciler) finalizeJob(ctx context.Context, jobOrPod GenericJob) error {
	if jwf, implements := jobOrPod.(JobWithFinalize); implements {
		if err := jwf.Finalize(ctx, r.client); err != nil {
			return err
		}
	}

	return nil
}

// handleWorkloadAfterDeactivatedPolicy
// 设计目的：
//   - 支持对象保留策略（ObjectRetentionPolicies）中的 afterDeactivatedByKueue 配置，
//     即 Kueue 管理的 Workload 被标记为停用（Deactivated）后，延迟一段时间自动删除对应的 Job。
//   - 该策略可用于释放资源、触发级联垃圾回收等。
//
// 参数说明：
//
//	ctx        ：上下文对象，用于日志和 API 调用。
//	jobOrPod   ：通用 Job 或 Pod 对象，需实现 GenericJob 接口。
//	wl         ：与 Job 关联的 Workload 对象。
//
// 主要逻辑：
//  1. 首先判断当前 Workload 是否满足“应根据去激活策略删除”的条件（shouldHandleDeletionOfDeactivatedWorkload）。
//     - 需满足：配置了 afterDeactivatedByKueue，且 Workload 已被 Kueue 驱逐（Evicted），且原因是 Deactivated。
//  2. 计算距离“去激活保留期”到期的剩余时间（requeueAfter）。
//     - 如果已到期（requeueAfter <= 0）：
//     a. 删除对应的 Job（带后台级联删除策略）。
//     b. 记录事件。
//     c. 调用 finalizeJob 做善后处理。
//     d. 返回 0 和错误（如有）。
//     - 如果未到期：
//     a. 记录日志，返回 requeueAfter，表示需在该时间后再次检查。
//
// 返回值：
//   - time.Duration：距离下次检查的时间（0 表示无需再次检查）。
//   - error        ：操作过程中的错误。
func (r *JobReconciler) handleWorkloadAfterDeactivatedPolicy(ctx context.Context, jobOrPod GenericJob, wl *kueue.Workload) (time.Duration, error) {
	//根据 释放（Deactivated）策略处理 Workload 对应的 Job 删除时机。
	if !r.shouldHandleDeletionOfDeactivatedWorkload(wl) {
		return 0, nil
	}

	log := ctrl.LoggerFrom(ctx)
	evCond := apimeta.FindStatusCondition(wl.Status.Conditions, kueue.WorkloadEvicted)
	requeueAfter := evCond.LastTransitionTime.Add(*r.workloadRetentionPolicy.AfterDeactivatedByKueue).Sub(r.clock.Now())

	if requeueAfter <= 0 {
		object := jobOrPod.Object()
		log.V(2).Info(
			"Deleting job: deactivation retention period expired",
			"retention", *r.workloadRetentionPolicy.AfterDeactivatedByKueue,
		)
		if err := r.client.Delete(ctx, object, client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil {
			return 0, client.IgnoreNotFound(err)
		}
		r.record.Event(object, corev1.EventTypeNormal, ReasonDeleted, "Deleted job: deactivation retention period expired")
		if err := r.finalizeJob(ctx, jobOrPod); err != nil {
			return 0, err
		}
		return 0, nil
	}

	log.V(3).Info("Requeuing job for deletion", "requeueAfter", requeueAfter)
	return requeueAfter, nil
}

func (r *JobReconciler) shouldHandleDeletionOfDeactivatedWorkload(wl *kueue.Workload) bool {
	return r.workloadRetentionPolicy.AfterDeactivatedByKueue != nil && !workload.IsActive(wl) && workload.IsEvictedDueToDeactivationByKueue(wl)
}

// WithObjectRetentionPolicies 设置 DeactivationRetentionPeriod（去激活保留期）策略。
func WithObjectRetentionPolicies(value *configapi.ObjectRetentionPolicies) Option {
	return func(o *Options) {
		if value != nil && value.Workloads != nil && value.Workloads.AfterDeactivatedByKueue != nil {
			o.WorkloadRetentionPolicy.AfterDeactivatedByKueue = &value.Workloads.AfterDeactivatedByKueue.Duration
		}
	}
}

func (r *JobReconciler) recordAdmissionCheckUpdate(wl *kueue.Workload, jobOrPod GenericJob) {
	message := ""
	object := jobOrPod.Object()
	for _, check := range wl.Status.AdmissionChecks {
		if check.State == kueue.CheckStatePending && check.Message != "" {
			if message != "" {
				message += "; "
			}
			message += string(check.Name) + ": " + check.Message
		}
	}
	if message != "" {
		if cJob, isComposable := jobOrPod.(ComposablePod); isComposable {
			cJob.ForEach(func(obj runtime.Object) {
				r.record.Eventf(obj, corev1.EventTypeNormal, ReasonUpdatedAdmissionCheck, message)
			})
		} else {
			r.record.Eventf(object, corev1.EventTypeNormal, ReasonUpdatedAdmissionCheck, message)
		}
	}
}

// ReconcileGenericJob 是 Kueue Job 框架的核心调谐循环，负责调度和管理通用 Job 类型的生命周期。
// 它根据 Job 及其关联 Workload 的状态，执行如下主要流程：
//  1. 处理 Job/Pod 的加载与前置跳过逻辑（如 ComposablePod、JobWithSkip）。
//  2. 处理 Job/Pod 及其 Workload 的 Finalizer 清理。
//  3. 递归查找 Kueue 管理的祖先 Job，决定当前 Job 是否为顶层 Job。
//  4. 根据 manageJobsWithoutQueueName 和命名空间选择器，决定是否管理该 Job。
//  5. 对于非顶层 Job，根据祖先 Workload 的状态决定是否挂起当前 Job。
//  6. 确保只有一个 Workload 与 Job 关联，并根据接口扩展点（如 JobWithCustomWorkloadConditions）更新 Workload 状态。
//  7. 处理 Workload 已完成、待删除、Job 已完成、Workload 不存在等场景。
//  8. 支持 ReclaimablePods、WaitForPodsReady、Evicted、优先级变更等扩展特性。
//  9. 根据 Workload Admission 状态决定是否启动/挂起 Job，并同步相关状态。
//  10. 处理调度失败、驱逐、资源释放、优先级变更等异常和边界情况。
//
// 该方法是 Kueue 控制器的核心调度循环，确保 Job 与 Workload 状态一致，并实现资源高效调度与回收。
func (r *JobReconciler) ReconcileGenericJob(ctx context.Context, req ctrl.Request, jobOrPod GenericJob) (result ctrl.Result, err error) {
	object := jobOrPod.Object()
	log := ctrl.LoggerFrom(ctx).WithValues("jobOrPod", req.String(), "gvk", jobOrPod.GVK()) // pod 、jobOrPod
	ctx = ctrl.LoggerInto(ctx, log)

	defer func() {
		err = r.ignoreUnretryableError(log, err)
	}()

	dropFinalizers := false
	if cJob, isComposable := jobOrPod.(ComposablePod); isComposable { // Pod
		dropFinalizers, err = cJob.Load(ctx, r.client, &req.NamespacedName)
	} else { // Job
		err = r.client.Get(ctx, req.NamespacedName, object)
		dropFinalizers = apierrors.IsNotFound(err) || !object.GetDeletionTimestamp().IsZero()
	}

	if jws, implements := jobOrPod.(JobWithSkip); implements {
		if jws.Skip() {
			return ctrl.Result{}, nil
		}
	}

	if dropFinalizers {
		// Remove workload finalizer
		workloads := &kueue.WorkloadList{}

		if cJob, isComposable := jobOrPod.(ComposablePod); isComposable {
			var err error
			workloads, err = cJob.ListChildWorkloads(ctx, r.client, req.NamespacedName)
			if err != nil {
				log.Error(err, "Removing finalizer")
				return ctrl.Result{}, err
			}
		} else {
			if err := r.client.List(ctx, workloads, client.InNamespace(req.Namespace),
				client.MatchingFields{GetOwnerKey(jobOrPod.GVK()): req.Name}); err != nil {
				log.Error(err, "Unable to list child workloads")
				return ctrl.Result{}, err
			}
		}
		for i := range workloads.Items {
			err := workload.RemoveFinalizer(ctx, r.client, &workloads.Items[i])
			if client.IgnoreNotFound(err) != nil {
				log.Error(err, "Removing finalizer")
				return ctrl.Result{}, err
			}
		}

		// Remove jobOrPod finalizer
		if !object.GetDeletionTimestamp().IsZero() {
			if err = r.finalizeJob(ctx, jobOrPod); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var (
		ancestorJob   client.Object
		isTopLevelJob bool
	)

	if topLevelJob, ok := jobOrPod.(TopLevelJob); ok && topLevelJob.IsTopLevel() { // Pod
		// Skipping traversal to top-level ancestor jobOrPod because this is already a top-level jobOrPod.
		isTopLevelJob = true
	} else { // 查找最上一级   管理的资源类型
		ancestorJob, err = FindAncestorJobManagedByKueue(ctx, r.client, object, r.manageJobsWithoutQueueName)
		if err != nil {
			if errors.Is(err, ErrManagedOwnersChainLimitReached) {
				errMsg := fmt.Sprintf("Terminated search for Kueue-managed Job because ancestor depth exceeded limit of %d", managedOwnersChainLimit)
				r.record.Eventf(object, corev1.EventTypeWarning, ReasonJobNestingTooDeep, errMsg)
				log.Error(err, errMsg)
			}
			return ctrl.Result{}, err
		}
		isTopLevelJob = ancestorJob == nil
	}

	//当“manageJobsWithoutQueueName”功能被禁用时，我们只会对那些要么带有“queue-name”标签、
	//要么其“kueue-managed”祖先带有“queue-name”标签的作业进行整合操作。
	if !r.manageJobsWithoutQueueName && QueueName(jobOrPod) == "" { // 管理没有queue的 jobOrPod\pod
		if isTopLevelJob {
			log.V(3).Info("queue-name label is not set, ignoring the jobOrPod")
			return ctrl.Result{}, nil
		}
		if QueueNameForObject(ancestorJob) == "" {
			log.V(3).Info("No kueue-managed ancestors have a queue-name label, ignoring the jobOrPod")
			return ctrl.Result{}, nil
		}
	}

	// 如果这是一个非顶级任务，则在其父级任务的工作负载未找到或未被接纳的情况下，应暂停该任务。
	if !isTopLevelJob {
		_, _, finished := jobOrPod.Finished()
		if !finished && !jobOrPod.IsSuspended() { // 运行中
			if ancestorWorkload, err := r.getWorkloadForObject(ctx, ancestorJob); err != nil {
				log.Error(err, "couldn't get an ancestor jobOrPod workload")
				return ctrl.Result{}, err
			} else if ancestorWorkload == nil || !workload.IsAdmitted(ancestorWorkload) { // 不允许运行
				if err := clientutil.Patch(ctx, r.client, object, true, func() (bool, error) {
					jobOrPod.Suspend()
					return true, nil
				}); err != nil {
					log.Error(err, "suspending child jobOrPod failed")
					return ctrl.Result{}, err
				}
				r.record.Event(object, corev1.EventTypeNormal, ReasonSuspended, "Kueue managed child jobOrPod suspended")
			}
		}
		return ctrl.Result{}, nil
	}

	// when manageJobsWithoutQueueName is enabled, standalone jobs without queue names
	// are still not managed if they don't match the namespace selector.
	if features.Enabled(features.ManagedJobsNamespaceSelector) && r.manageJobsWithoutQueueName && QueueName(jobOrPod) == "" {
		ns := corev1.Namespace{}
		err := r.client.Get(ctx, client.ObjectKey{Name: jobOrPod.Object().GetNamespace()}, &ns)
		if err != nil {
			log.Error(err, "failed to get jobOrPod namespace")
			return ctrl.Result{}, err
		}
		if !r.managedJobsNamespaceSelector.Matches(labels.Set(ns.GetLabels())) {
			log.V(3).Info("namespace selector does not match, ignoring the jobOrPod", "namespace", ns.Name)
			return ctrl.Result{}, nil
		}
	}

	log.V(2).Info("Reconciling Job")

	// 1. 确保只存在一个关联的 workload 实例。
	// 如果没有 workload 且 jobOrPod 未挂起，则立即挂起 jobOrPod。
	wl, err := r.ensureOneWorkload(ctx, jobOrPod, object)
	if err != nil {
		return ctrl.Result{}, err
	}

	// 如果实现了 JobWithCustomWorkloadConditions 接口，则更新 workload 的 conditions。
	if jobCond, ok := jobOrPod.(JobWithCustomWorkloadConditions); wl != nil && ok { // Pod
		if conditions, updated := jobCond.CustomWorkloadConditions(wl); updated { // ✅
			wlPatch := workload.BaseSSAWorkload(wl) // ✅
			wlPatch.Status.Conditions = conditions
			return reconcile.Result{}, r.client.Status().Patch(ctx, wlPatch, client.Apply, client.FieldOwner(fmt.Sprintf("%s-%s-controller", constants.KueueName, strings.ToLower(jobOrPod.GVK().Kind))))
		}
	}

	if wl != nil && apimeta.IsStatusConditionTrue(wl.Status.Conditions, kueue.WorkloadFinished) {
		// 如果 workload 已经完成，执行 jobOrPod 的最终处理逻辑。
		if err := r.finalizeJob(ctx, jobOrPod); err != nil {
			return ctrl.Result{}, err
		} // 移除finalize

		r.record.Eventf(object, corev1.EventTypeNormal, ReasonFinishedWorkload, "Workload '%s' is declared finished", workload.Key(wl))
		return ctrl.Result{}, workload.RemoveFinalizer(ctx, r.client, wl)
	}

	// 1.1 如果 workload 正在被删除，则挂起 jobOrPod（如有必要）并移除 finalizer。
	if wl != nil && !wl.DeletionTimestamp.IsZero() {
		log.V(2).Info("The workload is marked for deletion")
		err := r.stopJob(ctx, jobOrPod, wl, StopReasonWorkloadDeleted, "Workload is deleted")
		if err != nil {
			log.Error(err, "Suspending jobOrPod with deleted workload")
		} else {
			err = workload.RemoveFinalizer(ctx, r.client, wl)
		}
		return ctrl.Result{}, err
	}

	// 2. 处理 jobOrPod 已完成的情况。
	if message, success, finished := jobOrPod.Finished(); finished {
		log.V(3).Info("The workload is already finished")
		if wl != nil && !apimeta.IsStatusConditionTrue(wl.Status.Conditions, kueue.WorkloadFinished) {
			reason := kueue.WorkloadFinishedReasonSucceeded
			if !success {
				reason = kueue.WorkloadFinishedReasonFailed
			}
			err := workload.UpdateStatus(ctx, r.client, wl, kueue.WorkloadFinished, metav1.ConditionTrue, reason, message, constants.JobControllerName, r.clock)
			if err != nil && !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
			r.record.Eventf(object, corev1.EventTypeNormal, ReasonFinishedWorkload, "Workload '%s' is declared finished", workload.Key(wl))
		}

		// Execute jobOrPod finalization logic
		// 执行 jobOrPod 的最终处理逻辑
		if err := r.finalizeJob(ctx, jobOrPod); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// 3. handle workload is nil.
	// 处理 workload 不存在的情况。
	if wl == nil {
		log.V(3).Info("The workload is nil, handle jobOrPod with no workload")
		err := r.handleJobWithNoWorkload(ctx, jobOrPod, object) // 不存在创建 workload
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				log.V(3).Info("Handling jobOrPod with no workload found an existing workload")
				return ctrl.Result{Requeue: true}, nil
			}
			if IsUnretryableError(err) {
				log.V(3).Info("Handling jobOrPod with no workload", "unretryableError", err)
			} else {
				log.Error(err, "Handling jobOrPod with no workload")
			}
		}
		return ctrl.Result{}, err
	}

	// 4. 如果 jobOrPod 实现了 JobWithReclaimablePods 接口，则更新 reclaimable pods 数量。
	if jobRecl, implementsReclaimable := jobOrPod.(JobWithReclaimablePods); implementsReclaimable { // pod\job都实现了
		log.V(3).Info("update reclaimable counts if implemented by the jobOrPod")
		reclPods, err := jobRecl.ReclaimablePods() // ✅
		if err != nil {
			log.Error(err, "Getting reclaimable pods")
			return ctrl.Result{}, err
		}

		if !workload.ReclaimablePodsAreEqual(reclPods, wl.Status.ReclaimablePods) {
			err = workload.UpdateReclaimablePods(ctx, r.client, wl, reclPods) // ✅
			if err != nil {
				log.Error(err, "Updating reclaimable pods")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	}

	// 5. 仅对独立 jobOrPod 处理 WaitForPodsReady。当 waitForPodsReady 启用且为主 jobOrPod 时处理。
	if r.waitForPodsReady { // 指示控制器是否应在对应的 job 所有 pod 就绪或成功时，向 workload 添加 PodsReady 条件。
		log.V(3).Info("Handling a jobOrPod when waitForPodsReady is enabled")
		condition := generatePodsReadyCondition(log, jobOrPod, wl, r.clock)
		if !workload.HasConditionWithTypeAndReason(wl, &condition) {
			log.V(3).Info("Updating the PodsReady condition", "reason", condition.Reason, "status", condition.Status)
			apimeta.SetStatusCondition(&wl.Status.Conditions, condition)
			err := workload.UpdateStatus(ctx, r.client, wl, condition.Type, condition.Status, condition.Reason, condition.Message, constants.JobControllerName, r.clock)
			if err != nil {
				log.Error(err, "Updating workload status")
			}
			// update the metrics only when PodsReady condition status is true
			// 仅当 PodsReady 条件为 true 时才更新相关指标
			if condition.Status == metav1.ConditionTrue {
				cqName := wl.Status.Admission.ClusterQueue
				queuedUntilReadyWaitTime := workload.QueuedWaitTime(wl, r.clock)
				metrics.ReadyWaitTime(cqName, queuedUntilReadyWaitTime)
				admittedCond := apimeta.FindStatusCondition(wl.Status.Conditions, kueue.WorkloadAdmitted)
				admittedUntilReadyWaitTime := condition.LastTransitionTime.Sub(admittedCond.LastTransitionTime.Time)
				metrics.AdmittedUntilReadyWaitTime(cqName, admittedUntilReadyWaitTime)
				if features.Enabled(features.LocalQueueMetrics) {
					metrics.LocalQueueReadyWaitTime(metrics.LQRefFromWorkload(wl), queuedUntilReadyWaitTime)
					metrics.LocalQueueAdmittedUntilReadyWaitTime(metrics.LQRefFromWorkload(wl), admittedUntilReadyWaitTime)
				}
			}
			return ctrl.Result{}, nil
		} else {
			log.V(3).Info("No update for PodsReady condition")
		}
	}

	// 6. 处理驱逐场景。
	if evCond := apimeta.FindStatusCondition(wl.Status.Conditions, kueue.WorkloadEvicted); evCond != nil && evCond.Status == metav1.ConditionTrue {
		log.V(3).Info("Handling a jobOrPod with evicted condition")
		if err := r.stopJob(ctx, jobOrPod, wl, StopReasonWorkloadEvicted, evCond.Message); err != nil {
			return ctrl.Result{}, err
		}
		if workload.HasQuotaReservation(wl) {
			if !jobOrPod.IsActive() {
				log.V(6).Info("The jobOrPod is no longer active, clear the workloads admission")
				// 仅在被抢占驱逐时设置 requeued 条件为 true
				setRequeued := evCond.Reason == kueue.WorkloadEvictedByPreemption
				workload.SetRequeuedCondition(wl, evCond.Reason, evCond.Message, setRequeued)
				_ = workload.UnsetQuotaReservationWithCondition(wl, "Pending", evCond.Message, r.clock.Now())
				err := workload.ApplyAdmissionStatus(ctx, r.client, wl, true, r.clock)
				if err != nil {
					return ctrl.Result{}, fmt.Errorf("clearing admission: %w", err)
				}
			}
		}

		if features.Enabled(features.ObjectRetentionPolicies) {
			requeueAfter, err := r.handleWorkloadAfterDeactivatedPolicy(ctx, jobOrPod, wl) // ✅
			return ctrl.Result{RequeueAfter: requeueAfter}, err
		}
		return ctrl.Result{}, nil
	}

	// 7. 处理 jobOrPod 被挂起的场景。
	if jobOrPod.IsSuspended() {
		// 如果 workload 已被准入且 jobOrPod 仍挂起，则启动 jobOrPod
		if workload.IsAdmitted(wl) {
			log.V(2).Info("Job admitted, unsuspending")
			err := r.startJob(ctx, jobOrPod, object, wl) // job 取消暂停，pod 删掉 schedulingGate
			if err != nil {
				log.Error(err, "Unsuspending jobOrPod")
				if podset.IsPermanent(err) {
					// 标记 workload 失败完成，因为无需重试
					errUpdateStatus := workload.UpdateStatus(ctx, r.client, wl, kueue.WorkloadFinished, metav1.ConditionTrue, FailedToStartFinishedReason, err.Error(), constants.JobControllerName, r.clock)
					if errUpdateStatus != nil {
						log.Error(errUpdateStatus, "Updating workload status, on start failure", "err", err)
					}
					return ctrl.Result{}, errUpdateStatus
				}
			}
			return ctrl.Result{}, err
		}

		if workload.HasQuotaReservation(wl) {
			r.recordAdmissionCheckUpdate(wl, jobOrPod)
		}
		// 如果队列名发生变化，更新 workload
		q := QueueName(jobOrPod)
		if wl.Spec.QueueName != q {
			log.V(2).Info("Job changed queues, updating workload")
			wl.Spec.QueueName = q
			err := r.client.Update(ctx, wl)
			if err != nil {
				log.Error(err, "Updating workload queue")
			}
			return ctrl.Result{}, err
		}

		// 如果 jobOrPod 的优先级标签发生变化，更新 workload 的优先级
		if WorkloadPriorityClassName(object) != wl.Spec.PriorityClassName {
			log.V(2).Info("Job changed priority, updating workload", "oldPriority", wl.Spec.PriorityClassName, "newPriority", WorkloadPriorityClassName(object))
			if _, err = r.updateWorkloadToMatchJob(ctx, jobOrPod, object, wl); err != nil {
				log.Error(err, "Updating workload priority")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		log.V(3).Info("Job is suspended and workload not yet admitted by a clusterQueue, nothing to do")
		return ctrl.Result{}, nil
	}

	// 8. 未暂停
	if !workload.IsAdmitted(wl) {
		// the jobOrPod must be suspended if the workload is not yet admitted.
		log.V(2).Info("Running jobOrPod is not admitted by a cluster queue, suspending")
		err := r.stopJob(ctx, jobOrPod, wl, StopReasonNotAdmitted, "Not admitted by cluster queue")
		if err != nil {
			log.Error(err, "Suspending jobOrPod with non admitted workload")
		}
		return ctrl.Result{}, err
	}

	// workload is admitted and jobOrPod is running, nothing to do.
	log.V(3).Info("Job running with admitted workload, nothing to do")
	return ctrl.Result{}, nil
}

func (r *JobReconciler) ensurePrebuiltWorkloadInSync(ctx context.Context, wl *kueue.Workload, jobOrPod GenericJob) (bool, error) {
	var (
		equivalent bool // 等效
		err        error
	)

	if cj, implements := jobOrPod.(ComposablePod); implements {
		equivalent, err = cj.EquivalentToWorkload(ctx, r.client, wl)
	} else {
		equivalent, err = EquivalentToWorkload(ctx, r.client, jobOrPod, wl) // ✅ job 是够与workload 相匹配，以及删除多余的workload
	}

	if !equivalent || err != nil {
		if err != nil {
			return false, err
		}
		// mark the workload as finished
		err := workload.UpdateStatus(ctx, r.client, wl, kueue.WorkloadFinished, metav1.ConditionTrue, // 不一致， 就认为是完成了
			kueue.WorkloadFinishedReasonOutOfSync,
			"预先设定的工作负载与用户的任务不一致了。",
			constants.JobControllerName, r.clock)
		return false, err
	}
	return true, nil
}
