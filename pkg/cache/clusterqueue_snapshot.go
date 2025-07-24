package cache

import (
	"iter"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/features"
	"sigs.k8s.io/kueue/pkg/hierarchy"
	"sigs.k8s.io/kueue/pkg/metrics"
	"sigs.k8s.io/kueue/pkg/resources"
	utiltas "sigs.k8s.io/kueue/pkg/util/tas"
	"sigs.k8s.io/kueue/pkg/workload"
)

// ClusterQueueSnapshot 表示某一时刻 ClusterQueue 的快照，
// 用于调度和资源分配决策。该结构体包含了当前队列的资源、工作负载、
// 命名空间选择器、抢占策略、权重、AdmissionChecks 等信息。
// 通过快照可以模拟资源的增减、判断资源可用性等。
type ClusterQueueSnapshot struct {
	Name                                    kueue.ClusterQueueReference                                               // ClusterQueue 的唯一标识引用
	ResourceGroups                          []ResourceGroup                                                           // 该队列可用的资源组列表
	Workloads                               map[string]*workload.Info                                                 // 当前在队列中的工作负载信息，key 为 workload 名称
	WorkloadsNotReady                       sets.Set[string]                                                          // 尚未就绪的工作负载名称集合
	NamespaceSelector                       labels.Selector                                                           // 命名空间选择器，决定哪些命名空间可用此队列
	Preemption                              kueue.ClusterQueuePreemption                                              // 抢占策略配置
	FairWeight                              resource.Quantity                                                         // 队列的公平调度权重
	FlavorFungibility                       kueue.FlavorFungibility                                                   // Flavor 可替代性配置
	AdmissionChecks                         map[kueue.AdmissionCheckReference]sets.Set[kueue.ResourceFlavorReference] // AdmissionChecks 及其适用的 ResourceFlavor 集合
	Status                                  metrics.ClusterQueueStatus                                                // 队列当前状态
	AllocatableResourceGeneration           int64                                                                     // 可分配资源的版本号（变更时递增）
	ResourceNode                            ResourceNode                                                              // 1、flavorName+ResourceName: quota
	hierarchy.ClusterQueue[*CohortSnapshot]                                                                           // 队列的层级结构信息
	TASFlavors                              map[kueue.ResourceFlavorReference]*TASFlavorSnapshot                      // TAS 相关的 ResourceFlavor 快照
	tasOnly                                 bool                                                                      // 是否仅 TAS 模式
	flavorsForProvReqACs                    sets.Set[kueue.ResourceFlavorReference]                                   // 需要 AdmissionCheck 的 ResourceFlavor 集合
}

// RGByResource returns the ResourceGroup which contains capacity
// for the resource, or nil if the CQ doesn't provide this resource.
// RGByResource 返回包含该资源容量的 ResourceGroup，如果 CQ 不提供该资源则返回 nil。
func (c *ClusterQueueSnapshot) RGByResource(resource corev1.ResourceName) *ResourceGroup {
	for i := range c.ResourceGroups {
		if c.ResourceGroups[i].CoveredResources.Has(resource) {
			return &c.ResourceGroups[i]
		}
	}
	return nil
}

// SimulateWorkloadRemoval modifies the snapshot by removing the usage
// corresponding to the list of workloads. It returns a function which
// can be used to restore the usage.
// SimulateWorkloadRemoval 通过移除 workloads 列表对应的使用量来修改快照。返回一个可用于恢复使用量的函数。
func (c *ClusterQueueSnapshot) SimulateWorkloadRemoval(workloads []*workload.Info) func() {
	usage := make([]workload.Usage, 0, len(workloads))
	for _, w := range workloads {
		usage = append(usage, w.Usage())
	}
	for _, u := range usage {
		c.RemoveUsage(u)
	}
	return func() {
		for _, u := range usage {
			c.AddUsage(u)
		}
	}
}

// SimulateUsageRemoval 通过移除使用量来修改快照，并返回一个用于恢复使用量的函数。
func (c *ClusterQueueSnapshot) SimulateUsageRemoval(usage workload.Usage) func() {
	c.RemoveUsage(usage)
	return func() {
		c.AddUsage(usage)
	}
}

// RemoveUsage 从快照中移除资源使用量。
func (c *ClusterQueueSnapshot) RemoveUsage(usage workload.Usage) {
	for fr, q := range usage.Quota {
		removeUsage(c, fr, q)
	}
	c.updateTASUsage(usage.TAS, subtract)
}

// Fits 检查快照是否有足够的资源容量容纳指定的 usage。
func (c *ClusterQueueSnapshot) Fits(usage workload.Usage) bool {
	for fr, q := range usage.Quota {
		if c.Available(fr) < q {
			return false
		}
	}
	for tasFlavor, flvUsage := range usage.TAS {
		// We assume the `tasFlavor` is already in the snapshot as this was
		// already checked earlier during flavor assignment, and the set of
		// flavors is immutable in snapshot.
		if !c.TASFlavors[tasFlavor].Fits(flvUsage) {
			return false
		}
	}
	return true
}

// Borrowing 判断当前资源 flavor 是否处于借用状态。
func (c *ClusterQueueSnapshot) Borrowing(fr resources.FlavorResource) bool {
	return c.BorrowingWith(fr, 0)
}

// BorrowingWith 判断加上 val 后资源 flavor 是否处于借用状态。
func (c *ClusterQueueSnapshot) BorrowingWith(fr resources.FlavorResource, val int64) bool {
	return c.ResourceNode.Usage[fr]+val > c.QuotaFor(fr).Nominal
}

// Available 返回当前可用容量（包括本地和 Cohort 借用的容量），如果处于债务状态则返回 0。
func (c *ClusterQueueSnapshot) Available(fr resources.FlavorResource) int64 {
	return max(0, available(c, fr)) // ✅
}

// PotentialAvailable 返回该 ClusterQueue 理论上可接纳的最大 workload。
func (c *ClusterQueueSnapshot) PotentialAvailable(fr resources.FlavorResource) int64 {
	return potentialAvailable(c, fr) // ✅
}

// GetName 返回 ClusterQueue 的名称。
func (c *ClusterQueueSnapshot) GetName() kueue.ClusterQueueReference {
	return c.Name
}

// Implements dominantResourceShareNode interface.

// fairWeight 返回公平权重。
func (c *ClusterQueueSnapshot) fairWeight() *resource.Quantity {
	return &c.FairWeight
}

// implement flatResourceNode/hierarchicalResourceNode interfaces

// getResourceNode 返回资源节点。
func (c *ClusterQueueSnapshot) getResourceNode() ResourceNode {
	return c.ResourceNode
}

// parentHRN 返回父节点。
func (c *ClusterQueueSnapshot) parentHRN() hierarchicalResourceNode {
	return c.Parent()
}

// DominantResourceShare 返回主导资源份额。
func (c *ClusterQueueSnapshot) DominantResourceShare() int {
	share, _ := dominantResourceShare(c, nil)
	return share
}

type WorkloadTASRequests map[kueue.ResourceFlavorReference]FlavorTASRequests

// IsTASOnly 判断是否仅为 TAS。
func (c *ClusterQueueSnapshot) IsTASOnly() bool {
	return c.tasOnly
}

// HasProvRequestAdmissionCheck 判断指定 flavor 是否有 ProvisioningRequest AdmissionCheck。
func (c *ClusterQueueSnapshot) HasProvRequestAdmissionCheck(rf kueue.ResourceFlavorReference) bool {
	return c.flavorsForProvReqACs.Has(rf)
}

// PathParentToRoot 返回所有祖先（从父到根）。
func (c *ClusterQueueSnapshot) PathParentToRoot() iter.Seq[*CohortSnapshot] {
	return func(yield func(*CohortSnapshot) bool) {
		a := c.Parent()
		for a != nil {
			if !yield(a) {
				return
			}
			a = a.Parent()
		}
	}
}

// QuotaFor 返回指定资源 flavor 的配额。
func (c *ClusterQueueSnapshot) QuotaFor(fr resources.FlavorResource) ResourceQuota {
	return c.ResourceNode.Quotas[fr]
}

// SimulateUsageAddition 通过增加使用量来修改快照，并返回一个用于恢复使用量的函数。
func (c *ClusterQueueSnapshot) SimulateUsageAddition(usage workload.Usage) func() {
	c.AddUsage(usage)
	return func() {
		c.RemoveUsage(usage)
	}
}

// AddUsage 向快照中添加资源使用量。
func (c *ClusterQueueSnapshot) AddUsage(usage workload.Usage) {
	for fr, q := range usage.Quota {
		addUsage(c, fr, q)
	}
	c.updateTASUsage(usage.TAS, add)
}

// updateTASUsage 更新 TAS 相关的资源使用量。
func (c *ClusterQueueSnapshot) updateTASUsage(usage workload.TASUsage, op usageOp) {
	if features.Enabled(features.TopologyAwareScheduling) {
		for tasFlavor, tasUsage := range usage {
			if tasFlvCache := c.TASFlavors[tasFlavor]; tasFlvCache != nil {
				for _, tr := range tasUsage {
					domainID := utiltas.DomainID(tr.Values)
					tasFlvCache.updateTASUsage(domainID, tr.TotalRequests(), op, tr.Count)
				}
			}
		}
	}
}

// FindTopologyAssignmentsForWorkload 为 workload 查找拓扑分配。
func (c *ClusterQueueSnapshot) FindTopologyAssignmentsForWorkload(tasRequestsByFlavor WorkloadTASRequests,
	simulateEmpty bool, wl *kueue.Workload) TASAssignmentsResult {
	result := make(TASAssignmentsResult)
	for tasFlavor, flavorTASRequests := range tasRequestsByFlavor {
		// 我们假定“tasFlavor”已在快照中存在，因为这一情况在之前分配口味时已经进行了检查，并且快照中的口味集合是不可变的。
		tasFlavorCache := c.TASFlavors[tasFlavor]
		flvResult := tasFlavorCache.FindTopologyAssignmentsForFlavor(flavorTASRequests, simulateEmpty, wl)
		for psName, psAssignment := range flvResult {
			result[psName] = psAssignment
		}
	}
	return result
}
