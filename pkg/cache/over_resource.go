package cache

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/features"
	"sigs.k8s.io/kueue/pkg/resources"
)

type ResourceGroup struct {
	CoveredResources sets.Set[corev1.ResourceName]
	Flavors          []kueue.ResourceFlavorReference
	// 所有 flavor 的 key label 集合。
	// 这些 key 定义了 workload 的亲和性条件，可用于与 flavor 匹配。
	LabelKeys sets.Set[string]
}

func (rg *ResourceGroup) Clone() ResourceGroup {
	return ResourceGroup{
		CoveredResources: rg.CoveredResources.Clone(),
		Flavors:          rg.Flavors,
		LabelKeys:        rg.LabelKeys.Clone(),
	}
}

type ResourceQuota struct { // 分类为 cq 的资源; cohort 可以使用的资源
	Nominal        int64  // 规定的大小
	BorrowingLimit *int64 // 借入
	LendingLimit   *int64 // 借出
}

func createResourceQuotas(kueueRgs []kueue.ResourceGroup) map[resources.FlavorResource]ResourceQuota {
	frCount := 0
	for _, rg := range kueueRgs {
		frCount += len(rg.Flavors) * len(rg.CoveredResources)
	}
	quotas := make(map[resources.FlavorResource]ResourceQuota, frCount)
	for _, kueueRg := range kueueRgs {
		for _, kueueFlavor := range kueueRg.Flavors {
			for _, kueueQuota := range kueueFlavor.Resources {
				quota := ResourceQuota{
					Nominal: resources.ResourceValue(kueueQuota.Name, kueueQuota.NominalQuota),
				}
				if kueueQuota.BorrowingLimit != nil {
					quota.BorrowingLimit = ptr.To(resources.ResourceValue(kueueQuota.Name, *kueueQuota.BorrowingLimit))
				}
				if features.Enabled(features.LendingLimit) && kueueQuota.LendingLimit != nil {
					quota.LendingLimit = ptr.To(resources.ResourceValue(kueueQuota.Name, *kueueQuota.LendingLimit))
				}
				quotas[resources.FlavorResource{Flavor: kueueFlavor.Name, Resource: kueueQuota.Name}] = quota
			}
		}
	}
	return quotas
}
