package features

import (
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/version"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/component-base/featuregate"
	featuregatetesting "k8s.io/component-base/featuregate/testing"
)

const (
	// 启用部分准入。
	PartialAdmission featuregate.Feature = "PartialAdmission"

	// 启用队列可见性。
	QueueVisibility featuregate.Feature = "QueueVisibility"

	// 启用Flavor可替代性。
	FlavorFungibility featuregate.Feature = "FlavorFungibility"

	// 启用供应准入检查控制器。
	ProvisioningACC featuregate.Feature = "ProvisioningACC"

	// 启用按需 Kueue 可见性
	VisibilityOnDemand featuregate.Feature = "VisibilityOnDemand"

	// 启用队列组内的优先级排序。
	PrioritySortingWithinCohort featuregate.Feature = "PrioritySortingWithinCohort"

	// 启用 MultiKueue 支持。
	MultiKueue featuregate.Feature = "MultiKueue"

	// 启用借贷限额。
	LendingLimit featuregate.Feature = "LendingLimit"

	// 启用 batch.Job spec.managedBy 字段在 MultiKueue 集成中的使用。
	MultiKueueBatchJobWithManagedBy featuregate.Feature = "MultiKueueBatchJobWithManagedBy"

	// 允许一个以上的 Workload 在队列组内共享风味进行抢占，只要抢占目标不重叠。
	MultiplePreemptions featuregate.Feature = "MultiplePreemptions"

	// 启用拓扑感知调度，允许优化 Pod 的放置，使其位于相邻节点（如同一机架或区块）上。
	TopologyAwareScheduling featuregate.Feature = "TopologyAwareScheduling"

	// 启用在计算 Workload 资源请求时应用可配置的资源转换。
	ConfigurableResourceTransformations featuregate.Feature = "ConfigurableResourceTransformations"

	// 汇总未准入 Workload 的资源请求到 Workload.Status.resourceRequest 字段，以提升可观测性。
	WorkloadResourceRequestsSummary featuregate.Feature = "WorkloadResourceRequestsSummary"

	// 启用 LocalQueue 的 Flavors 状态字段，允许用户查看 LocalQueue 当前可用的所有 ResourceFlavors。
	ExposeFlavorsInLocalQueue featuregate.Feature = "ExposeFlavorsInLocalQueue"

	// 启用基于命名空间的 manageJobsWithoutQueueNames 控制，适用于所有 Job 集成。
	ManagedJobsNamespaceSelector featuregate.Feature = "ManagedJobsNamespaceSelector"

	// 启用收集 LocalQueue 指标。
	LocalQueueMetrics featuregate.Feature = "LocalQueueMetrics"

	// 启用设置默认 LocalQueue。
	LocalQueueDefaulting featuregate.Feature = "LocalQueueDefaulting"

	// 启用为 TAS 使用 LeastFreeCapacity 算法。
	TASProfileLeastFreeCapacity featuregate.Feature = "TASProfileLeastFreeCapacity"

	// 启用为 TAS 使用混合算法（BestFit 或 LeastFreeCapacity），根据 TAS 需求级别切换算法。
	TASProfileMixed featuregate.Feature = "TASProfileMixed"

	// 启用分层队列组。
	HierarchicalCohorts featuregate.Feature = "HierarchicalCohorts"

	// 启用准入公平共享。
	AdmissionFairSharing featuregate.Feature = "AdmissionFairSharing"

	// 启用对象保留策略。
	ObjectRetentionPolicies featuregate.Feature = "ObjectRetentionPolicies"

	// 启用 TAS 中失败节点的替换。
	TASFailedNodeReplacement featuregate.Feature = "TASFailedNodeReplacement"

	// 如果 Kueue 在 TAS 中第一次尝试未能找到失败节点的替换，则驱逐 Workload。
	TASFailedNodeReplacementFailFast featuregate.Feature = "TASFailedNodeReplacementFailFast"
)

func init() {
	runtime.Must(utilfeature.DefaultMutableFeatureGate.AddVersioned(defaultVersionedFeatureGates))
}

// defaultVersionedFeatureGates consists of all known Kueue-specific feature keys.
// To add a new feature, define a key for it above and add it here. The features will be
// available throughout Kueue binaries.
//
// Entries are separated from each other with blank lines to avoid sweeping gofmt changes
// when adding or removing one entry.
var defaultVersionedFeatureGates = map[featuregate.Feature]featuregate.VersionedSpecs{
	PartialAdmission: {
		{Version: version.MustParse("0.4"), Default: false, PreRelease: featuregate.Alpha},
		{Version: version.MustParse("0.5"), Default: true, PreRelease: featuregate.Beta},
	},
	QueueVisibility: {
		{Version: version.MustParse("0.5"), Default: false, PreRelease: featuregate.Alpha},
		{Version: version.MustParse("0.9"), Default: false, PreRelease: featuregate.Deprecated},
	},
	FlavorFungibility: {
		{Version: version.MustParse("0.5"), Default: true, PreRelease: featuregate.Beta},
	},
	ProvisioningACC: {
		{Version: version.MustParse("0.5"), Default: true, PreRelease: featuregate.Beta},
	},
	VisibilityOnDemand: {
		{Version: version.MustParse("0.6"), Default: false, PreRelease: featuregate.Alpha},
		{Version: version.MustParse("0.9"), Default: true, PreRelease: featuregate.Beta},
	},
	PrioritySortingWithinCohort: {
		{Version: version.MustParse("0.6"), Default: true, PreRelease: featuregate.Beta},
	},
	MultiKueue: {
		{Version: version.MustParse("0.6"), Default: false, PreRelease: featuregate.Alpha},
		{Version: version.MustParse("0.9"), Default: true, PreRelease: featuregate.Beta},
	},
	LendingLimit: {
		{Version: version.MustParse("0.6"), Default: false, PreRelease: featuregate.Alpha},
		{Version: version.MustParse("0.9"), Default: true, PreRelease: featuregate.Beta},
	},
	MultiKueueBatchJobWithManagedBy: {
		{Version: version.MustParse("0.8"), Default: false, PreRelease: featuregate.Alpha},
	},
	MultiplePreemptions: {
		{Version: version.MustParse("0.8"), Default: false, PreRelease: featuregate.Alpha},
		{Version: version.MustParse("0.9"), Default: true, PreRelease: featuregate.Beta},
		{Version: version.MustParse("0.10"), Default: true, PreRelease: featuregate.GA, LockToDefault: true}, // remove in 0.12
	},
	TopologyAwareScheduling: {
		{Version: version.MustParse("0.9"), Default: false, PreRelease: featuregate.Alpha},
	},
	ConfigurableResourceTransformations: {
		{Version: version.MustParse("0.9"), Default: false, PreRelease: featuregate.Alpha},
		{Version: version.MustParse("0.10"), Default: true, PreRelease: featuregate.Beta},
	},
	WorkloadResourceRequestsSummary: {
		{Version: version.MustParse("0.9"), Default: true, PreRelease: featuregate.Alpha},
		{Version: version.MustParse("0.10"), Default: true, PreRelease: featuregate.Beta},
		{Version: version.MustParse("0.11"), Default: true, PreRelease: featuregate.GA, LockToDefault: true}, // remove in 0.13
	},
	ExposeFlavorsInLocalQueue: {
		{Version: version.MustParse("0.9"), Default: true, PreRelease: featuregate.Beta},
	},
	ManagedJobsNamespaceSelector: {
		{Version: version.MustParse("0.10"), Default: true, PreRelease: featuregate.Beta},
	},
	LocalQueueMetrics: {
		{Version: version.MustParse("0.10"), Default: false, PreRelease: featuregate.Alpha},
	},
	LocalQueueDefaulting: {
		{Version: version.MustParse("0.10"), Default: false, PreRelease: featuregate.Alpha},
		{Version: version.MustParse("0.12"), Default: true, PreRelease: featuregate.Beta},
	},
	TASProfileLeastFreeCapacity: {
		{Version: version.MustParse("0.10"), Default: false, PreRelease: featuregate.Alpha},
		{Version: version.MustParse("0.11"), Default: false, PreRelease: featuregate.Deprecated},
	},
	TASProfileMixed: {
		{Version: version.MustParse("0.10"), Default: false, PreRelease: featuregate.Alpha},
		{Version: version.MustParse("0.11"), Default: false, PreRelease: featuregate.Deprecated},
	},
	HierarchicalCohorts: {
		{Version: version.MustParse("0.11"), Default: true, PreRelease: featuregate.Beta},
	},
	AdmissionFairSharing: {
		{Version: version.MustParse("0.12"), Default: false, PreRelease: featuregate.Alpha},
	},
	ObjectRetentionPolicies: {
		{Version: version.MustParse("0.12"), Default: false, PreRelease: featuregate.Alpha},
	},
	TASFailedNodeReplacement: {
		{Version: version.MustParse("0.12"), Default: false, PreRelease: featuregate.Alpha},
	},
	TASFailedNodeReplacementFailFast: {
		{Version: version.MustParse("0.13"), Default: false, PreRelease: featuregate.Alpha},
	},
}

func SetFeatureGateDuringTest(tb testing.TB, f featuregate.Feature, value bool) {
	featuregatetesting.SetFeatureGateDuringTest(tb, utilfeature.DefaultFeatureGate, f, value)
}

// Enabled is helper for `utilfeature.DefaultFeatureGate.Enabled()`
func Enabled(f featuregate.Feature) bool {
	return utilfeature.DefaultFeatureGate.Enabled(f)
}

// SetEnable helper function that can be used to set the enabled value of a feature gate,
// it should only be used in integration test pending the merge of
// https://github.com/kubernetes/kubernetes/pull/118346
func SetEnable(f featuregate.Feature, v bool) error {
	return utilfeature.DefaultMutableFeatureGate.Set(fmt.Sprintf("%s=%v", f, v))
}

func LogFeatureGates(log logr.Logger) {
	features := make(map[featuregate.Feature]bool, len(defaultVersionedFeatureGates))
	for f := range utilfeature.DefaultMutableFeatureGate.GetAll() {
		if _, ok := defaultVersionedFeatureGates[f]; ok {
			features[f] = Enabled(f)
		}
	}
	log.V(2).Info("Loaded feature gates", "featureGates", features)
}
