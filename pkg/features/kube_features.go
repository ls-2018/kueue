package features

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/version"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/component-base/featuregate"
)

const (
	// owner: @trasc
	// kep: https://github.com/kubernetes-sigs/kueue/tree/main/keps/420-partial-admission
	//
	// 启用部分准入。
	PartialAdmission featuregate.Feature = "PartialAdmission"

	// owner: @stuton
	// kep: https://github.com/kubernetes-sigs/kueue/tree/main/keps/168-pending-workloads-visibility
	//
	// 启用队列可见性。
	QueueVisibility featuregate.Feature = "QueueVisibility"

	// owner: @KunWuLuan
	// kep: https://github.com/kubernetes-sigs/kueue/tree/main/keps/582-preempt-based-on-flavor-order
	//
	// 启用风味可替代性。
	FlavorFungibility featuregate.Feature = "FlavorFungibility"

	// owner: @trasc
	// kep: https://github.com/kubernetes-sigs/kueue/tree/main/keps/1136-provisioning-request-support
	//
	// 启用供应准入检查控制器。
	ProvisioningACC featuregate.Feature = "ProvisioningACC"

	// owner: @pbundyra
	// kep: https://github.com/kubernetes-sigs/kueue/tree/main/keps/168-2-pending-workloads-visibility
	//
	// 启用按需 Kueue 可见性
	VisibilityOnDemand featuregate.Feature = "VisibilityOnDemand"

	// owner: @yaroslava-serdiuk
	// kep: https://github.com/kubernetes-sigs/kueue/issues/1283
	//
	// 启用队列组内的优先级排序。
	PrioritySortingWithinCohort featuregate.Feature = "PrioritySortingWithinCohort"

	// owner: @trasc
	// kep: https://github.com/kubernetes-sigs/kueue/tree/main/keps/693-multikueue
	//
	// 启用 MultiKueue 支持。
	MultiKueue featuregate.Feature = "MultiKueue"

	// owners: @B1F030, @kerthcet
	// kep: https://github.com/kubernetes-sigs/kueue/tree/main/keps/1224-lending-limit
	//
	// 启用借贷限制。
	LendingLimit featuregate.Feature = "LendingLimit"

	// owner: @trasc
	// kep: https://github.com/kubernetes-sigs/kueue/tree/main/keps/693-multikueue
	//
	// 启用 batch.Job spec.managedBy 字段在 MultiKueue 集成中的使用。
	MultiKueueBatchJobWithManagedBy featuregate.Feature = "MultiKueueBatchJobWithManagedBy"

	// owner: @gabesaba
	// kep: https://github.com/kubernetes-sigs/kueue/issues/2596
	//
	// 启用多个共享风味的工作负载在同一队列组内抢占，只要抢占目标不重叠。
	MultiplePreemptions featuregate.Feature = "MultiplePreemptions"

	// owner: @mimowo
	//
	// 启用拓扑感知调度，优化 Pod 的放置，使其位于相近的节点上（例如同一机架或区块内）。
	TopologyAwareScheduling featuregate.Feature = "TopologyAwareScheduling"

	// owner: @dgrove-oss
	// kep: https://github.com/kubernetes-sigs/kueue/tree/main/keps/2937-resource-transformer
	//
	// 启用在计算 Workload 资源请求时应用可配置的资源转换。
	ConfigurableResourceTransformations featuregate.Feature = "ConfigurableResourceTransformations"

	// owner: @dgrove-oss
	// kep: https://github.com/kubernetes-sigs/kueue/tree/main/keps/2937-resource-transformer
	//
	// 汇总未被接纳的 Workload 的资源请求到 Workload.Status.resourceRequest 字段，以提升可观测性。
	WorkloadResourceRequestsSummary featuregate.Feature = "WorkloadResourceRequestsSummary"

	// owner: @mbobrovskyi
	//
	// 启用 LocalQueue 的 Flavors 状态字段，允许用户查看 LocalQueue 当前可用的所有 ResourceFlavor。
	ExposeFlavorsInLocalQueue featuregate.Feature = "ExposeFlavorsInLocalQueue"

	// owner: @dgrove-oss
	// kep: https://github.com/kubernetes-sigs/kueue/tree/main/keps/3589-manage-jobs-selectively
	//
	// 启用所有 Job 集成的 manageJobsWithoutQueueNames 的基于命名空间的控制。
	ManagedJobsNamespaceSelector featuregate.Feature = "ManagedJobsNamespaceSelector"

	// owner: @kpostoffice
	// kep: https://github.com/kubernetes-sigs/kueue/tree/main/keps/1833-metrics-for-local-queue
	//
	// 启用收集 LocalQueue 指标。
	LocalQueueMetrics featuregate.Feature = "LocalQueueMetrics"

	// owner: @yaroslava-serdiuk
	// kep: https://github.com/kubernetes-sigs/kueue/tree/main/keps/2936-local-queue-defaulting
	//
	// 启用设置默认 LocalQueue。
	LocalQueueDefaulting featuregate.Feature = "LocalQueueDefaulting"

	// owner: @pbundyra
	// kep: https://github.com/kubernetes-sigs/kueue/tree/main/keps/2724-topology-aware-scheduling
	//
	// 启用为 TAS 使用 MostFreeCapacity 算法。
	TASProfileMostFreeCapacity featuregate.Feature = "TASProfileMostFreeCapacity"

	// owner: @pbundyra
	// kep: https://github.com/kubernetes-sigs/kueue/tree/main/keps/2724-topology-aware-scheduling
	//
	// 启用为 TAS 使用 LeastFreeCapacity 算法。
	TASProfileLeastFreeCapacity featuregate.Feature = "TASProfileLeastFreeCapacity"

	// owner: @pbundyra
	// kep: https://github.com/kubernetes-sigs/kueue/tree/main/keps/2724-topology-aware-scheduling
	//
	// 启用为 TAS 使用混合算法（BestFit 或 LeastFreeCapacity），根据 TAS 需求级别切换算法。
	TASProfileMixed featuregate.Feature = "TASProfileMixed"

	// owner: @mwielgus
	// kep: https://github.com/kubernetes-sigs/kueue/tree/main/keps/79-hierarchical-cohorts
	//
	// 启用分层队列组。
	HierarchicalCohorts featuregate.Feature = "HierarchicalCohorts"

	// owner: @pbundyra
	// kep: https://github.com/kubernetes-sigs/kueue/tree/main/keps/4136-admission-fair-sharing
	//
	// 启用准入公平共享。
	AdmissionFairSharing featuregate.Feature = "AdmissionFairSharing"

	// owner: @mwysokin @mykysha @mbobrovskyi
	// kep: https://github.com/kubernetes-sigs/kueue/tree/main/keps/1618-optional-gc-of-workloads
	//
	// 启用对象保留策略。
	ObjectRetentionPolicies featuregate.Feature = "ObjectRetentionPolicies"

	// owner: @pajakd
	// kep: https://github.com/kubernetes-sigs/kueue/tree/main/keps/2724-topology-aware-scheduling
	//
	// 启用 TAS 中失败节点的替换。
	TASFailedNodeReplacement featuregate.Feature = "TASFailedNodeReplacement"
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
		{Version: version.MustParse("0.5"), Default: true, PreRelease: featuregate.Beta},
	},
	QueueVisibility: {
		{Version: version.MustParse("0.9"), Default: true, PreRelease: featuregate.Deprecated},
	},
	FlavorFungibility: {
		{Version: version.MustParse("0.5"), Default: true, PreRelease: featuregate.Beta},
	},
	ProvisioningACC: {
		{Version: version.MustParse("0.5"), Default: true, PreRelease: featuregate.Beta},
	},
	VisibilityOnDemand: {
		{Version: version.MustParse("0.9"), Default: true, PreRelease: featuregate.Beta},
	},
	PrioritySortingWithinCohort: {
		{Version: version.MustParse("0.6"), Default: true, PreRelease: featuregate.Beta},
	},
	MultiKueue: {
		{Version: version.MustParse("0.9"), Default: true, PreRelease: featuregate.Beta},
	},
	LendingLimit: {
		{Version: version.MustParse("0.9"), Default: true, PreRelease: featuregate.Beta},
	},
	MultiKueueBatchJobWithManagedBy: {
		{Version: version.MustParse("0.8"), Default: true, PreRelease: featuregate.Alpha},
	},
	MultiplePreemptions: {
		{Version: version.MustParse("0.10"), Default: true, PreRelease: featuregate.GA, LockToDefault: true}, // remove in 0.12
	},
	TopologyAwareScheduling: {
		{Version: version.MustParse("0.9"), Default: true, PreRelease: featuregate.Alpha},
	},
	ConfigurableResourceTransformations: {
		{Version: version.MustParse("0.10"), Default: true, PreRelease: featuregate.Beta},
	},
	WorkloadResourceRequestsSummary: {
		{Version: version.MustParse("0.11"), Default: true, PreRelease: featuregate.GA, LockToDefault: true}, // remove in 0.13
	},
	ExposeFlavorsInLocalQueue: {
		{Version: version.MustParse("0.9"), Default: true, PreRelease: featuregate.Beta},
	},
	ManagedJobsNamespaceSelector: {
		{Version: version.MustParse("0.10"), Default: true, PreRelease: featuregate.Beta},
	},
	LocalQueueMetrics: {
		{Version: version.MustParse("0.10"), Default: true, PreRelease: featuregate.Alpha},
	},
	LocalQueueDefaulting: {
		{Version: version.MustParse("0.12"), Default: true, PreRelease: featuregate.Beta},
	},

	HierarchicalCohorts: {
		{Version: version.MustParse("0.11"), Default: true, PreRelease: featuregate.Beta},
	},
	AdmissionFairSharing: {
		{Version: version.MustParse("0.12"), Default: true, PreRelease: featuregate.Alpha},
	},
	ObjectRetentionPolicies: {
		{Version: version.MustParse("0.12"), Default: true, PreRelease: featuregate.Alpha},
	},
	TASFailedNodeReplacement: {
		{Version: version.MustParse("0.12"), Default: false, PreRelease: featuregate.Alpha},
	},
	TASProfileMostFreeCapacity: {
		{Version: version.MustParse("0.11"), Default: false, PreRelease: featuregate.Deprecated},
	},
	TASProfileLeastFreeCapacity: {
		{Version: version.MustParse("0.11"), Default: false, PreRelease: featuregate.Deprecated},
	},
	TASProfileMixed: {
		{Version: version.MustParse("0.11"), Default: true, PreRelease: featuregate.Deprecated},
	},
}

// Enabled is helper for `utilfeature.DefaultFeatureGate.Enabled()`
func Enabled(f featuregate.Feature) bool {
	return utilfeature.DefaultFeatureGate.Enabled(f)
}

// SetEnable helper function that can be used to set the enabled value of a feature gate,
// it should only be used in integration test pending the merge of
// https://github.com/kubernetes/kubernetes/pull/118346

func LogFeatureGates(log logr.Logger) {
	features := make(map[featuregate.Feature]bool, len(defaultVersionedFeatureGates))
	for f := range utilfeature.DefaultMutableFeatureGate.GetAll() {
		if _, ok := defaultVersionedFeatureGates[f]; ok {
			features[f] = Enabled(f)
		}
	}
	log.V(2).Info("Loaded feature gates", "featureGates", features)
}
