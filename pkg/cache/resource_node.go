package cache

import (
	"maps"

	"k8s.io/apimachinery/pkg/util/sets"

	"sigs.k8s.io/kueue/pkg/hierarchy"
	"sigs.k8s.io/kueue/pkg/resources"
)

// ResourceNode 是 Quotas 和 Usage 的共享表示，由 ClusterQueues 和 Cohorts 使用。
type ResourceNode struct { // 所有 flavor 的资源情况
	Quotas       map[resources.FlavorResource]ResourceQuota // flavorName+resourceName: Quota
	SubtreeQuota resources.FlavorResourceQuantities         // 是节点配额的总和，以及受 LendingLimits(借给别人) 限制的其子节点可用资源。
	Usage        resources.FlavorResourceQuantities         // cohort 所有子 cq 不允许出借的资源
}

func NewResourceNode() ResourceNode {
	return ResourceNode{
		Quotas:       make(map[resources.FlavorResource]ResourceQuota),
		SubtreeQuota: make(resources.FlavorResourceQuantities),
		Usage:        make(resources.FlavorResourceQuantities),
	}
}

// Clone 克隆可变字段 Usage，同时返回 Quota 和 SubtreeQuota 的副本（这些在更新时会被新 map 替换）。
func (r ResourceNode) Clone() ResourceNode {
	return ResourceNode{
		Quotas:       r.Quotas,
		SubtreeQuota: r.SubtreeQuota,
		Usage:        maps.Clone(r.Usage),
	}
}

// hierarchicalResourceNode extends flatResourceNode
// with the ability to navigate to the parent node.
// hierarchicalResourceNode 扩展了 flatResourceNode，具备导航到父节点的能力。
type hierarchicalResourceNode interface {
	flatResourceNode
	HasParent() bool
	parentHRN() hierarchicalResourceNode
}

// flatResourceNode 通过提供对包含的 ResourceNode 的访问，实现对 ClusterQueues 和 Cohorts 的抽象。
type flatResourceNode interface {
	getResourceNode() ResourceNode
}

// LocalAvailable returns, for a given node and resource flavor,
// how much guaranteed quota in this flavor exceeds usage.
// This quota is available at this node but is not visible at its parent.
// LocalAvailable 返回对于给定节点和资源 flavor，该 flavor 的 guaranteed quota 超出使用量的部分。
// 这部分配额在本节点可用，但在父节点不可见。
func LocalAvailable(node flatResourceNode, fr resources.FlavorResource) int64 {
	return max(0, node.getResourceNode().guaranteedQuota(fr)-node.getResourceNode().Usage[fr])
}

// potentialAvailable returns the maximum capacity available to this node,
// assuming no usage, while respecting BorrowingLimits.
// potentialAvailable 返回在无使用量且遵守 BorrowingLimits 的情况下，此节点可用的最大容量。
func potentialAvailable(node hierarchicalResourceNode, fr resources.FlavorResource) int64 {
	r := node.getResourceNode()
	if !node.HasParent() {
		return r.SubtreeQuota[fr]
	}
	available := r.guaranteedQuota(fr) + potentialAvailable(node.parentHRN(), fr)
	if borrowingLimit := r.Quotas[fr].BorrowingLimit; borrowingLimit != nil {
		maxWithBorrowing := r.SubtreeQuota[fr] + *borrowingLimit
		available = min(maxWithBorrowing, available)
	}
	return available
}

// addUsage adds usage to the current node, and bubbles up usage to
// its Cohort when usage exceeds guaranteedQuota.
// addUsage 向当前节点添加使用量，当使用量超过 guaranteedQuota 时，将超出部分向 Cohort 逐级上报。
func addUsage(node hierarchicalResourceNode, fr resources.FlavorResource, val int64) {
	r := node.getResourceNode()
	localAvailable := LocalAvailable(node, fr)
	r.Usage[fr] += val
	if node.HasParent() && val > localAvailable {
		deltaParentUsage := val - localAvailable
		addUsage(node.parentHRN(), fr, deltaParentUsage)
	}
}

// removeUsage removes usage from the current node, and removes usage
// past guaranteedQuota that it was storing in its Cohort.
// removeUsage 从当前节点移除使用量，并从 Cohort 中移除超出 guaranteedQuota 的部分。
func removeUsage(node hierarchicalResourceNode, fr resources.FlavorResource, val int64) {
	r := node.getResourceNode()
	usageStoredInParent := r.Usage[fr] - r.guaranteedQuota(fr)
	r.Usage[fr] -= val
	if usageStoredInParent <= 0 || !node.HasParent() {
		return
	}
	deltaParentUsage := min(val, usageStoredInParent)
	removeUsage(node.parentHRN(), fr, deltaParentUsage)
}

// updateCohortTreeResources 从根节点遍历 Cohort 树，累加 SubtreeQuota 和 Usage。如果 Cohort 存在环，则返回错误。
func updateCohortTreeResources(cohort *cohort) error {
	if hierarchy.HasCycle(cohort) {
		return ErrCohortHasCycle
	}
	updateCohortResourceNode(cohort.getRootUnsafe())
	return nil
}

// QuantitiesFitInQuota returns if resource requests fit in quota on this node
// and the amount of resources that exceed the guaranteed
// quota on this node. It is assumed that subsequent call
// to this function will be on nodes parent with remainingRequests
// QuantitiesFitInQuota 返回资源请求是否适合此节点的配额，以及超出 guaranteed quota 的资源量。假定后续对该函数的调用将在父节点上进行，参数为 remainingRequests。
func QuantitiesFitInQuota(node flatResourceNode, requests resources.FlavorResourceQuantities) (bool, resources.FlavorResourceQuantities) {
	fits := true
	remainingRequests := make(resources.FlavorResourceQuantities, len(requests))
	for fr, v := range requests {
		if node.getResourceNode().Usage[fr]+v > node.getResourceNode().SubtreeQuota[fr] {
			fits = false
		}
		remainingRequests[fr] = max(0, v-LocalAvailable(node, fr))
	}
	return fits, remainingRequests
}

// IsWithinNominalInResources returns whether or not, the node quota usage exceeds its
// nominal quota in any resource flavor out of a set of resource flavours.
// IsWithinNominalInResources 返回节点配额使用量是否在任一资源 flavor 上超出其 nominal quota。
func IsWithinNominalInResources(node flatResourceNode, frs sets.Set[resources.FlavorResource]) bool {
	for fr := range frs {
		if node.getResourceNode().Usage[fr] > node.getResourceNode().SubtreeQuota[fr] {
			return false
		}
	}
	return true
}

// available 用于确定当前节点剩余多少容量，考虑了使用量和 BorrowingLimits。它查找本地存储的剩余容量。如果节点有父节点，则查询父节点的容量，并通过借用限制和节点在父节点中存储/使用的容量来限制该容量。
// This function may return a negative number in the case of
// overadmission - e.g. capacity was removed or the node moved to
// another Cohort.
// 如果出现超额分配（如容量被移除或节点移动到另一个 Cohort），此函数可能返回负数。
func available(node hierarchicalResourceNode, fr resources.FlavorResource) int64 {
	_ = node.(*ClusterQueueSnapshot).ClusterQueue.HasParent
	r := node.getResourceNode()
	if !node.HasParent() { // 没有配置 cohort
		return r.SubtreeQuota[fr] - r.Usage[fr] // 正常值 - 出借值
	}
	parentAvailable := available(node.parentHRN(), fr)

	if borrowingLimit := r.Quotas[fr].BorrowingLimit; borrowingLimit != nil {
		storedInParent := r.SubtreeQuota[fr] - r.guaranteedQuota(fr)
		usedInParent := max(0, r.Usage[fr]-r.guaranteedQuota(fr))
		withMaxFromParent := storedInParent - usedInParent + *borrowingLimit
		parentAvailable = min(withMaxFromParent, parentAvailable)
	}
	return LocalAvailable(node, fr) + parentAvailable
}

// updateCohortResourceNode 遍历 Cohort 树以累加 SubtreeQuota 和 Usage。通常应通过 updateCohortTree 调用，该方法从根节点开始并包含环检测。
func updateCohortResourceNode(cohort *cohort) { // ✅
	cohort.resourceNode.SubtreeQuota = make(resources.FlavorResourceQuantities, len(cohort.resourceNode.SubtreeQuota))
	cohort.resourceNode.Usage = make(resources.FlavorResourceQuantities, len(cohort.resourceNode.Usage))

	for fr, quota := range cohort.resourceNode.Quotas {
		cohort.resourceNode.SubtreeQuota[fr] = quota.Nominal // 正常值  ✅
	}
	for _, child := range cohort.ChildCohorts() {
		updateCohortResourceNode(child)    // ✅
		accumulateFromChild(cohort, child) // ✅
	}
	for _, child := range cohort.ChildCQs() {
		updateClusterQueueResourceNode(child) // 重置 cq SubtreeQuota
		accumulateFromChild(cohort, child)    // ✅
	}
}
func accumulateFromChild(parent *cohort, child flatResourceNode) {
	for fr, childQuota := range child.getResourceNode().SubtreeQuota { // cohort 可以供使用的资源
		parent.resourceNode.SubtreeQuota[fr] += childQuota - child.getResourceNode().guaranteedQuota(fr) // ✅
	}
	for fr, childUsage := range child.getResourceNode().Usage {
		parent.resourceNode.Usage[fr] += max(0, childUsage-child.getResourceNode().guaranteedQuota(fr)) // ✅
	}
}

func updateClusterQueueResourceNode(cq *clusterQueue) {
	cq.AllocatableResourceGeneration++
	cq.resourceNode.SubtreeQuota = make(resources.FlavorResourceQuantities, len(cq.resourceNode.Quotas))
	for fr, quota := range cq.resourceNode.Quotas {
		cq.resourceNode.SubtreeQuota[fr] = quota.Nominal
	}
}

// guaranteedQuota 是不会借给节点 Cohort 的容量。
func (r ResourceNode) guaranteedQuota(fr resources.FlavorResource) int64 {
	if lendingLimit := r.Quotas[fr].LendingLimit; lendingLimit != nil {
		return max(0, r.SubtreeQuota[fr]-*lendingLimit)
	}
	return 0
}
