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
	SubtreeQuota resources.FlavorResourceQuantities         // cq.SubtreeQuota=cq.Nominal ;cohort.SubtreeQuota = sum([flavor.lendingLimit])
	Usage        resources.FlavorResourceQuantities         // 当前cq 使用的资源，会超出 上限-借出     ； cohort 用的(借用总和)
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

// “available” 参数决定了当前节点剩余的可用容量大小，该值会综合考虑使用情况和借用限制等因素。
//
//	1、它会查找本地存储的剩余容量。
//	2、则会查询父节点的容量，并将此容量限制在借用限制范围内— 同时还要考虑到该节点在其父节点中所存储/使用的容量。
//
// 在出现超额入院的情况时（例如，病床容量被取消或该患者被转至其他病区），此函数可能会返回负数。
func available(node hierarchicalResourceNode, fr resources.FlavorResource) int64 {
	// 获取当前节点的资源信息
	r := node.getResourceNode()
	// 如果没有父节点（即没有配置 cohort），直接返回本节点可用资源（总配额 - 已用资源）
	if !node.HasParent() { // 没有配置 cohort
		return r.SubtreeQuota[fr] - r.Usage[fr] // 正常值 - 已用资源
	}
	// 递归获取父节点的可用资源（即 cohort 的可用资源）
	parentAvailable := available(node.parentHRN(), fr) // 获取  cohort
	_ = ResourceNode{}                                 // 占位，无实际作用
	// 检查当前资源 flavor 是否设置了 BorrowingLimit（可借用上限）
	if borrowingLimit := r.Quotas[fr].BorrowingLimit; borrowingLimit != nil {
		// 可以借出的上限
		storedInParentUpperLimit := r.SubtreeQuota[fr] - r.guaranteedQuota(fr)
		// 已经借出的？
		usedInParentCurrent := max(0, r.Usage[fr]-r.guaranteedQuota(fr))
		// 可以借出的上限 - 已经借出的 + 可以借别人的
		withMaxFromParent := storedInParentUpperLimit - usedInParentCurrent + *borrowingLimit
		// 父节点可用资源不能超过上述计算的最大值
		parentAvailable = min(withMaxFromParent, parentAvailable)
	}
	return LocalAvailable(node, fr) + parentAvailable
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
		return max(0, r.SubtreeQuota[fr]-*lendingLimit) // 可供自己用的
	}
	return 0
}

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

// LocalAvailable 返回对于给定节点和资源 flavor，该 flavor 的 guaranteed quota 超出使用量的部分。
// 这部分配额在本节点可用，但在父节点不可见。
func LocalAvailable(node flatResourceNode, fr resources.FlavorResource) int64 {
	return max(0, node.getResourceNode().guaranteedQuota(fr)-node.getResourceNode().Usage[fr])
}

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
func accumulateFromChild(parent *cohort, child flatResourceNode) {
	// 每一轮，都会清空SubtreeQuota  Usage
	for fr, childQuota := range child.getResourceNode().SubtreeQuota { // cohort 可以供使用的资源
		parent.resourceNode.SubtreeQuota[fr] += childQuota - child.getResourceNode().guaranteedQuota(fr)
	}
	for fr, childUsage := range child.getResourceNode().Usage {
		parent.resourceNode.Usage[fr] += max(0, childUsage-child.getResourceNode().guaranteedQuota(fr))
	}
}

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
