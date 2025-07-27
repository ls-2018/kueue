package jobframework

import (
	"context"
	"strconv"

	"sigs.k8s.io/kueue/pkg/util/admissioncheck"
	"sigs.k8s.io/kueue/pkg/util/maps"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/controller/constants"
	"sigs.k8s.io/kueue/pkg/podset"
)

// GenericJob 是所有由 kueue 的 jobframework 管理的作业需要实现的接口。
type GenericJob interface {
	// Object 返回作业实例。
	Object() client.Object
	// IsSuspended 返回作业是否被挂起。
	IsSuspended() bool
	// Suspend 会挂起作业。
	Suspend()
	// RunWithPodSetsInfo 会将从 workload 提取的节点亲和性和 podSet 数量注入到作业中并取消挂起。
	RunWithPodSetsInfo(podSetsInfo []podset.PodSetInfo) error
	// RestorePodSetsInfo 会恢复作业的原始节点亲和性和 podSet 数量。
	// 返回是否有任何更改。
	RestorePodSetsInfo(podSetsInfo []podset.PodSetInfo) bool
	// Finished 表示作业是否已完成/失败，condition 代表 workload 的完成条件。
	// workload 的 observed generation 由 jobframework 设置。
	Finished() (message string, success, finished bool)
	// PodSets 会构建与作业对应的 workload podSets。
	PodSets() ([]kueue.PodSet, error)
	// IsActive 如果有正在运行的 pod，则返回 true。
	IsActive() bool
	// PodsReady 指示作业派生的 pod 是否都已就绪。
	PodsReady() bool
	// GVK 返回作业的 GVK（Group Version Kind）。
	GVK() schema.GroupVersionKind
}

// Optional interfaces, are meant to implemented by jobs to enable additional
// features of the jobframework reconciler.
// 可选接口，供作业实现以启用 jobframework reconciler 的附加功能。

type JobWithPodLabelSelector interface {
	// PodLabelSelector returns the label selector used by pods for the job.
	// PodLabelSelector 返回作业 pod 使用的标签选择器。
	PodLabelSelector() string
}

type JobWithReclaimablePods interface {
	// ReclaimablePods 返回可回收 pod 的列表。
	ReclaimablePods() ([]kueue.ReclaimablePod, error)
}

type StopReason string

const (
	StopReasonWorkloadDeleted    StopReason = "WorkloadDeleted"
	StopReasonWorkloadEvicted    StopReason = "WorkloadEvicted"
	StopReasonNoMatchingWorkload StopReason = "NoMatchingWorkload"
	StopReasonNotAdmitted        StopReason = "NotAdmitted"
)

type JobWithCustomStop interface {
	// Stop 实现自定义的停止流程。
	// 该函数应是幂等的：如果作业已停止，则不应进行任何 API 调用。
	// 返回此次调用是否停止了作业或错误。
	Stop(ctx context.Context, c client.Client, podSetsInfo []podset.PodSetInfo, stopReason StopReason, eventMsg string) (bool, error)
}

// JobWithFinalize interface should be implemented by generic jobs,
// when custom finalization logic is needed for a job, after it's finished.
// JobWithFinalize 接口应由通用作业实现，当作业完成后需要自定义的收尾逻辑时。
type JobWithFinalize interface {
	Finalize(ctx context.Context, c client.Client) error
}

// JobWithSkip interface should be implemented by generic jobs,
// when reconciliation should be skipped depending on the job's state
// JobWithSkip 接口应由通用作业实现，当根据作业状态需要跳过调和时。
type JobWithSkip interface {
	Skip() bool
}

type JobWithPriorityClass interface {
	// PriorityClass 返回作业的优先级类名。
	PriorityClass() string
}

// JobWithCustomValidation optional interface that allows custom webhook validation
// for Jobs that use BaseWebhook.
// JobWithCustomValidation 可选接口，允许使用 BaseWebhook 的作业自定义 webhook 校验。
type JobWithCustomValidation interface {
	// ValidateOnCreate returns list of webhook create validation errors.
	// ValidateOnCreate 返回 webhook 创建校验错误列表。
	ValidateOnCreate() field.ErrorList
	// ValidateOnUpdate returns list of webhook update validation errors.
	// ValidateOnUpdate 返回 webhook 更新校验错误列表。
	ValidateOnUpdate(oldJob GenericJob) field.ErrorList
}

// ComposablePod 接口应由由多个 API 对象组成的通用作业实现。
type ComposablePod interface {
	// Load 加载可组合作业的所有成员。如果 removeFinalizers == true，则应移除 workload 和 job 的 finalizer。
	Load(ctx context.Context, c client.Client, key *types.NamespacedName) (removeFinalizers bool, err error)
	// Run 取消挂起 ComposablePod 的所有成员，并将从 workload 提取的节点亲和性和 podSet 数量注入到所有成员中。
	Run(ctx context.Context, c client.Client, podSetsInfo []podset.PodSetInfo, r record.EventRecorder, msg string) error
	// ConstructComposableWorkload 返回由 ComposablePod 所有成员组装的新 Workload。
	ConstructComposableWorkload(ctx context.Context, c client.Client, r record.EventRecorder, labelKeysToCopy []string) (*kueue.Workload, error)
	// ListChildWorkloads 返回与可组合作业相关的所有 workload。
	ListChildWorkloads(ctx context.Context, c client.Client, parent types.NamespacedName) (*kueue.WorkloadList, error)
	// FindMatchingWorkloads 返回所有相关的 workload、与 ComposablePod 匹配的 workload 以及需要删除的重复项。
	FindMatchingWorkloads(ctx context.Context, c client.Client, r record.EventRecorder) (match *kueue.Workload, toDelete []*kueue.Workload, err error)
	// Stop 实现 ComposablePod 的自定义停止流程。
	Stop(ctx context.Context, c client.Client, podSetsInfo []podset.PodSetInfo, stopReason StopReason, eventMsg string) ([]client.Object, error)
	// ForEach 对 ComposablePod 的每个成员调用 f。
	ForEach(f func(obj runtime.Object))
	// EnsureWorkloadOwnedByAllMembers 确保所提供的 workload 被指定的所有者拥有。
	// 如果 workload 没有被所有指定的所有者拥有，则将其添加到 owner references。
	// 如果 workload 被更新则返回 true，出现问题则返回错误。
	EnsureWorkloadOwnedByAllMembers(ctx context.Context, c client.Client, r record.EventRecorder, workload *kueue.Workload) error
	// EquivalentToWorkload 检查所提供的 workload 是否等同于目标 workload。
	// 如果等同则返回 true，出现问题则返回错误。
	EquivalentToWorkload(ctx context.Context, c client.Client, wl *kueue.Workload) (bool, error)
}

// JobWithCustomWorkloadConditions 接口应由通用作业实现，当确保 workload 存在后需要更新自定义 workload 条件时。
type JobWithCustomWorkloadConditions interface {
	// CustomWorkloadConditions 返回自定义 workload 条件以及状态是否更改。
	CustomWorkloadConditions(wl *kueue.Workload) ([]metav1.Condition, bool)
}

// JobWithManagedBy interface should be implemented by generic jobs
// that implement the managedBy protocol for Multi-Kueue
// JobWithManagedBy 接口应由实现 Multi-Kueue managedBy 协议的通用作业实现。
type JobWithManagedBy interface {
	// CanDefaultManagedBy returns true of ManagedBy() would return nil or the default controller for the framework
	// CanDefaultManagedBy 如果 ManagedBy() 返回 nil 或框架的默认控制器，则返回 true。
	CanDefaultManagedBy() bool
	// ManagedBy returns the name of the controller that is managing the Job
	// ManagedBy 返回管理该作业的控制器名称。
	ManagedBy() *string
	// SetManagedBy sets the field in the spec that contains the name of the managing controller
	// SetManagedBy 设置 spec 中包含管理控制器名称的字段。
	SetManagedBy(*string)
}

func QueueName(jobOrPod GenericJob) kueue.LocalQueueName {
	return QueueNameForObject(jobOrPod.Object())
}

// MultiKueueAdapter 接口是 MultiKueue 作业委托所需的。
type MultiKueueAdapter interface {
	// SyncJob 使用远程 client 在工作集群中创建 Job 对象（如果尚未创建）。
	// 如果远程作业已存在，则复制其状态。
	SyncJob(ctx context.Context, localClient client.Client, remoteClient client.Client, key types.NamespacedName, workloadName, origin string) error
	// DeleteRemoteObject deletes the Job in the worker cluster.
	// DeleteRemoteObject 删除工作集群中的 Job。
	DeleteRemoteObject(ctx context.Context, remoteClient client.Client, key types.NamespacedName) error
	// IsJobManagedByKueue 返回：
	// - 一个 bool，指示由 key 标识的作业对象是否由 kueue 管理并可被委托。
	// - 一个字符串，说明作业未被 Kueue 管理的原因
	// - 检查过程中遇到的任何 API 错误
	IsJobManagedByKueue(ctx context.Context, localClient client.Client, key types.NamespacedName) (bool, string, error)
	// KeepAdmissionCheckPending 如果 multikueue admission check 的状态在作业在 worker 上运行时应保持 Pending，则返回 true。
	// 这可能需要让 manager 的作业保持挂起，不在本地启动执行。
	KeepAdmissionCheckPending() bool
	// GVK 返回作业的 GVK（Group Version Kind）。
	GVK() schema.GroupVersionKind
}

// MultiKueueWatcher optional interface that can be implemented by a MultiKueueAdapter
// to receive job related watch events from the worker cluster.
// If not implemented, MultiKueue will only receive events related to the job's workload.
// MultiKueueWatcher 可选接口，可由 MultiKueueAdapter 实现以接收来自 worker 集群的作业相关 watch 事件。
// 如果未实现，MultiKueue 只会接收与作业 workload 相关的事件。
type MultiKueueWatcher interface {
	// GetEmptyList 返回一个空对象列表。
	GetEmptyList() client.ObjectList
	// WorkloadKeyFor 返回感兴趣的 workload 的 key
	// - 对于 workload 是对象名
	// - 对于作业类型是预构建的 workload
	WorkloadKeyFor(runtime.Object) (types.NamespacedName, error)
}

func QueueNameForObject(object client.Object) kueue.LocalQueueName {
	if queueLabel := object.GetLabels()[constants.QueueLabel]; queueLabel != "" {
		return kueue.LocalQueueName(queueLabel)
	}
	// fallback to the annotation (deprecated)
	return kueue.LocalQueueName(object.GetAnnotations()[constants.QueueAnnotation])
}

func NewWorkload(name string, obj client.Object, podSets []kueue.PodSet, labelKeysToCopy []string) *kueue.Workload {
	return &kueue.Workload{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   obj.GetNamespace(),
			Labels:      maps.FilterKeys(obj.GetLabels(), labelKeysToCopy),
			Finalizers:  []string{kueue.ResourceInUseFinalizerName},
			Annotations: admissioncheck.FilterProvReqAnnotations(obj.GetAnnotations()),
		},
		Spec: kueue.WorkloadSpec{
			QueueName:                   QueueNameForObject(obj),
			PodSets:                     podSets,
			MaximumExecutionTimeSeconds: MaximumExecutionTimeSecondsForObject(obj),
		},
	}
}

// TopLevelJob interface is an optional interface used to indicate
// that the Job owns/manages the Workload object, regardless of the Job
// owner references.
// TopLevelJob 接口是一个可选接口，用于指示作业拥有/管理 Workload 对象，无论作业的 owner references 如何。
type TopLevelJob interface {
	// IsTopLevel returns true if the Job owns/manages the Workload.
	// IsTopLevel 如果作业拥有/管理 Workload，则返回 true。
	IsTopLevel() bool
}

func PrebuiltWorkloadFor(jobOrPod GenericJob) (string, bool) {
	name, found := jobOrPod.Object().GetLabels()[constants.PrebuiltWorkloadLabel]
	return name, found
}
func MaximumExecutionTimeSeconds(jobOrPod GenericJob) *int32 {
	return MaximumExecutionTimeSecondsForObject(jobOrPod.Object())
}

func MaximumExecutionTimeSecondsForObject(object client.Object) *int32 {
	strVal, found := object.GetLabels()[constants.MaxExecTimeSecondsLabel]
	if !found {
		return nil
	}

	v, err := strconv.ParseInt(strVal, 10, 32)
	if err != nil || v <= 0 {
		return nil
	}

	return ptr.To(int32(v))
}
func WorkloadPriorityClassName(object client.Object) string {
	if workloadPriorityClassLabel := object.GetLabels()[constants.WorkloadPriorityClassLabel]; workloadPriorityClassLabel != "" {
		return workloadPriorityClassLabel
	}
	return ""
}
