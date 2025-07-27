package cache

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"math"
	"sigs.k8s.io/kueue/pkg/resources"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
)

var (
	oneQuantity = resource.MustParse("1")
)

// dominantResourceShareNode 是 Cohort 树中可用于计算主导资源份额的节点。
type dominantResourceShareNode interface {
	fairWeight() *resource.Quantity
	hierarchicalResourceNode
}

// calculateLendable 聚合所有 FlavorResource 的资源容量。
func calculateLendable(node hierarchicalResourceNode) map[corev1.ResourceName]int64 {
	// walk to root
	root := node
	for root.HasParent() {
		root = root.parentHRN()
	}

	lendable := make(map[corev1.ResourceName]int64, len(root.getResourceNode().SubtreeQuota))
	// The root's SubtreeQuota contains all FlavorResources,
	// as we accumulate even 0s in accumulateFromChild.
	for fr := range root.getResourceNode().SubtreeQuota {
		lendable[fr.Resource] += potentialAvailable(node, fr)
	}
	return lendable
}

// parseFairWeight parses FairSharing.Weight if it exists,
// or otherwise returns the default value of 1.
// parseFairWeight 解析 FairSharing.Weight，如果不存在则返回默认值 1。
func parseFairWeight(fs *kueue.FairSharing) resource.Quantity {
	if fs == nil || fs.Weight == nil {
		return oneQuantity
	}
	return *fs.Weight
}

// dominantResourceShare 返回一个 0 到 1,000,000 的值，
// 表示所有资源中超出 nominal quota 的使用量与 cohort 可借用资源的最大比值，并除以权重。
// 如果为 0，表示 ClusterQueue 的使用量低于 nominal quota。函数还返回导致该值的资源名。
// 当 FairSharing 权重为 0 且 ClusterQueue 或 Cohort 处于借用状态时，返回 math.MaxInt。
func dominantResourceShare(node dominantResourceShareNode, wlReq resources.FlavorResourceQuantities) (int, corev1.ResourceName) {
	if !node.HasParent() {
		return 0, ""
	}

	borrowing := make(map[corev1.ResourceName]int64, len(node.getResourceNode().SubtreeQuota))
	for fr, quota := range node.getResourceNode().SubtreeQuota {
		amountBorrowed := wlReq[fr] + node.getResourceNode().Usage[fr] - quota
		if amountBorrowed > 0 {
			borrowing[fr.Resource] += amountBorrowed
		}
	}
	if len(borrowing) == 0 {
		return 0, ""
	}

	var drs int64 = -1
	var dRes corev1.ResourceName

	lendable := calculateLendable(node.parentHRN())
	for rName, b := range borrowing {
		if lr := lendable[rName]; lr > 0 {
			ratio := b * 1000 / lr
			// Use alphabetical order to get a deterministic resource name.
			if ratio > drs || (ratio == drs && rName < dRes) {
				drs = ratio
				dRes = rName
			}
		}
	}

	if node.fairWeight().IsZero() {
		return math.MaxInt, dRes
	}

	dws := drs * 1000 / node.fairWeight().MilliValue()
	return int(dws), dRes
}
