package flavorassigner

import (
	"errors"
	"fmt"

	"k8s.io/utils/ptr"

	kueuealpha "sigs.k8s.io/kueue/apis/kueue/v1alpha1"
	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/cache"
	"sigs.k8s.io/kueue/pkg/workload"
)

// WorkloadsTopologyRequests - 返回该 workload 的 TopologyRequests
func (a *Assignment) WorkloadsTopologyRequests(wl *workload.Info, cq *cache.ClusterQueueSnapshot) cache.WorkloadTASRequests {
	tasRequests := make(cache.WorkloadTASRequests)
	for i, podSet := range wl.Obj.Spec.PodSets {
		if isTASRequested(&podSet, cq) {
			psAssignment := a.podSetAssignmentByName(podSet.Name)
			if psAssignment.Status.IsError() {
				// PodSet 没有资源配额分配，无需检查 TAS。
				continue
			}
			if psAssignment.TopologyAssignment != nil && !psAssignment.HasFailedNode(wl) {
				// 如果已计算且不需要重新计算则跳过
				// 如果已有分配但因节点失败需要重新计算
				// 则将其加入 TASRequests 列表
				continue
			}
			isTASImplied := isTASImplied(&podSet, cq)
			psTASRequest, err := podSetTopologyRequest(psAssignment, wl, cq, isTASImplied, i)
			if err != nil {
				psAssignment.error(err)
			} else if psTASRequest != nil {
				tasRequests[psTASRequest.Flavor] = append(tasRequests[psTASRequest.Flavor], *psTASRequest)
			}
		}
	}
	return tasRequests
}

// HasFailedNode 判断该 PodSetAssignment 是否有需要替换的失败节点。
func (psa *PodSetAssignment) HasFailedNode(wl *workload.Info) bool {
	if !workload.HasNodeToReplace(wl.Obj) {
		return false
	}
	failedNode := wl.Obj.Annotations[kueuealpha.NodeToReplaceAnnotation]
	for _, domain := range psa.TopologyAssignment.Domains {
		if domain.Values[len(domain.Values)-1] == failedNode {
			return true
		}
	}
	return false
}

// podSetTopologyRequest 构造 PodSet 的拓扑请求。
func podSetTopologyRequest(psAssignment *PodSetAssignment,
	wl *workload.Info,
	cq *cache.ClusterQueueSnapshot,
	isTASImplied bool,
	podSetIndex int) (*cache.TASPodSetRequests, error) {
	if len(cq.TASFlavors) == 0 {
		return nil, errors.New("workload requires Topology, but there is no TAS cache information")
	}
	psResources := wl.TotalRequests[podSetIndex]
	singlePodRequests := psResources.SinglePodRequests()
	podCount := psAssignment.Count
	tasFlvr, err := onlyFlavor(psAssignment.Flavors)
	if err != nil {
		return nil, err
	}
	if !workload.HasQuotaReservation(wl.Obj) && cq.HasProvRequestAdmissionCheck(*tasFlvr) {
		// 由于这是第一次调度，且 flavor 使用了 ProvisioningRequest admission check，延迟 TAS。
		psAssignment.DelayedTopologyRequest = ptr.To(kueue.DelayedTopologyRequestStatePending)
		return nil, nil
	}
	if cq.TASFlavors[*tasFlvr] == nil {
		return nil, errors.New("workload requires Topology, but there is no TAS cache information for the assigned flavor")
	}
	podSet := &wl.Obj.Spec.PodSets[podSetIndex]
	var podSetUpdates []*kueue.PodSetUpdate
	for _, ac := range wl.Obj.Status.AdmissionChecks {
		if ac.State == kueue.CheckStateReady {
			for _, psUpdate := range ac.PodSetUpdates {
				if psUpdate.Name == podSet.Name {
					podSetUpdates = append(podSetUpdates, &psUpdate)
				}
			}
		}
	}
	return &cache.TASPodSetRequests{
		Count:             podCount,
		SinglePodRequests: singlePodRequests,
		PodSet:            podSet,
		PodSetUpdates:     podSetUpdates,
		Flavor:            *tasFlvr,
		Implied:           isTASImplied,
	}, nil
}

// onlyFlavor 检查 ResourceAssignment 是否只分配了一个 flavor，并返回该 flavor。
func onlyFlavor(ra ResourceAssignment) (*kueue.ResourceFlavorReference, error) {
	var result *kueue.ResourceFlavorReference
	for _, v := range ra {
		if result == nil {
			result = &v.Name
		} else if *result != v.Name {
			return nil, fmt.Errorf("more than one flavor assigned: %s, %s", v.Name, *result)
		}
	}
	if result != nil {
		return result, nil
	}
	return nil, errors.New("no flavor assigned")
}

// checkPodSetAndFlavorMatchForTAS 检查 PodSet 和 flavor 是否匹配 TAS。
func checkPodSetAndFlavorMatchForTAS(cq *cache.ClusterQueueSnapshot, ps *kueue.PodSet, flavor *kueue.ResourceFlavor) *string {
	// 对于需要 TAS 的 PodSet，跳过不支持 TAS 的 resource flavor
	if ps.TopologyRequest != nil {
		if flavor.Spec.TopologyName == nil {
			return ptr.To(fmt.Sprintf("Flavor %q does not support TopologyAwareScheduling", flavor.Name))
		}
		s := cq.TASFlavors[kueue.ResourceFlavorReference(flavor.Name)]
		if s == nil {
			// 如果没有 TAS 信息则跳过该 Flavor。一般不应发生，但在 ResourceFlavor API 对象刚添加还未缓存时可能出现。
			return ptr.To(fmt.Sprintf("Flavor %q information missing in TAS cache", flavor.Name))
		}
		// 判断 podSet指定的topology 是不是存在
		if !s.HasLevel(ps.TopologyRequest) { // ps.TopologyRequest 是写在声明的
			// 跳过不包含请求 level 的 flavor
			return ptr.To(fmt.Sprintf("Flavor %q does not contain the requested level", flavor.Name))
		}
	}
	// 如果 CQ 仅支持 TAS，则没有 TopologyRequest 也可以
	if isTASImplied(ps, cq) {
		return nil
	}
	// 对于不使用 TAS 的 PodSet，跳过仅支持 TAS 的 resource flavor
	if ps.TopologyRequest == nil && flavor.Spec.TopologyName != nil {
		return ptr.To(fmt.Sprintf("Flavor %q supports only TopologyAwareScheduling", flavor.Name))
	}
	return nil
}

// isTASImplied 判断是否隐式请求了 TAS（即没有显式 TopologyRequest 且 CQ 仅支持 TAS）。
func isTASImplied(ps *kueue.PodSet, cq *cache.ClusterQueueSnapshot) bool {
	return ps.TopologyRequest == nil && cq.IsTASOnly() // cq 使用的 flavor 都是支持 tas的
}

// isTASRequested 检查输入 PodSet 是否请求了 TAS（显式或隐式）。
func isTASRequested(ps *kueue.PodSet, cq *cache.ClusterQueueSnapshot) bool {
	return ps.TopologyRequest != nil || cq.IsTASOnly()
}
