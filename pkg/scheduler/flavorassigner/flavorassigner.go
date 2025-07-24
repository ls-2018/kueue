package flavorassigner

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	corev1helpers "k8s.io/component-helpers/scheduling/corev1"
	"k8s.io/component-helpers/scheduling/corev1/nodeaffinity"
	"k8s.io/utils/ptr"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/cache"
	"sigs.k8s.io/kueue/pkg/features"
	"sigs.k8s.io/kueue/pkg/resources"
	"sigs.k8s.io/kueue/pkg/scheduler/preemption/classical"
	preemptioncommon "sigs.k8s.io/kueue/pkg/scheduler/preemption/common"
	"sigs.k8s.io/kueue/pkg/workload"
)

// Assignment 结构体用于保存分配结果，包括每个 PodSet 的分配信息、借用情况、上一次状态、资源使用情况等。
type Assignment struct {
	PodSets []PodSetAssignment
	// Borrowing 表示适应额外使用量所需的最小 cohort 树的高度。如果不需要借用，则为 0。
	Borrowing int
	LastState workload.AssignmentClusterQueueState

	// Usage 表示随着 pod sets 被分配 flavor 后的资源累计使用量。
	Usage workload.Usage

	// representativeMode 是该分配的代表性分配模式的缓存。
	representativeMode *FlavorAssignmentMode
}

// UpdateForTASResult 用 TAS 结果更新 Assignment
func (a *Assignment) UpdateForTASResult(result cache.TASAssignmentsResult) {
	for psName, psResult := range result {
		psAssignment := a.podSetAssignmentByName(psName)
		psAssignment.TopologyAssignment = psResult.TopologyAssignment
		if psResult.TopologyAssignment != nil && psAssignment.DelayedTopologyRequest != nil {
			psAssignment.DelayedTopologyRequest = ptr.To(kueue.DelayedTopologyRequestStateReady)
		}
	}
	a.Usage.TAS = a.ComputeTASNetUsage(nil)
}

// ComputeTASNetUsage 计算该分配的净 TAS 使用量
func (a *Assignment) ComputeTASNetUsage(prevAdmission *kueue.Admission) workload.TASUsage {
	result := make(workload.TASUsage)
	for i, psa := range a.PodSets {
		if psa.TopologyAssignment != nil {
			if prevAdmission != nil && prevAdmission.PodSetAssignments[i].TopologyAssignment != nil {
				continue
			}
			singlePodRequests := resources.NewRequests(psa.Requests).ScaledDown(int64(psa.Count))
			for _, flv := range psa.Flavors {
				if _, ok := result[flv.Name]; !ok {
					result[flv.Name] = make(workload.TASFlavorUsage, 0)
				}
				for _, domain := range psa.TopologyAssignment.Domains {
					result[flv.Name] = append(result[flv.Name], workload.TopologyDomainRequests{
						Values:            domain.Values,
						SinglePodRequests: singlePodRequests.Clone(),
						Count:             domain.Count,
					})
				}
			}
		}
	}
	return result
}

// Borrows 返回该分配是否需要借用。
func (a *Assignment) Borrows() int {
	return a.Borrowing
}

func (a *Assignment) podSetAssignmentByName(psName kueue.PodSetReference) *PodSetAssignment {
	if idx := slices.IndexFunc(a.PodSets, func(ps PodSetAssignment) bool { return ps.Name == psName }); idx != -1 {
		return &a.PodSets[idx]
	}
	return nil
}

func (a *Assignment) Message() string {
	var builder strings.Builder
	for _, ps := range a.PodSets {
		if ps.Status == nil {
			continue
		}
		if ps.Status.IsError() {
			return fmt.Sprintf("failed to assign flavors to pod set %s: %v", ps.Name, ps.Status.err)
		}
		if builder.Len() > 0 {
			builder.WriteString("; ")
		}
		builder.WriteString("couldn't assign flavors to pod set ")
		builder.WriteString(string(ps.Name))
		builder.WriteString(": ")
		builder.WriteString(ps.Status.Message())
	}
	return builder.String()
}

func (a *Assignment) ToAPI() []kueue.PodSetAssignment {
	psFlavors := make([]kueue.PodSetAssignment, len(a.PodSets))
	for i := range psFlavors {
		psFlavors[i] = a.PodSets[i].toAPI()
	}
	return psFlavors
}

// TotalRequestsFor - 返回该 workload 的总配额需求，考虑部分准入时的缩放。
func (a *Assignment) TotalRequestsFor(wl *workload.Info) resources.FlavorResourceQuantities {
	usage := make(resources.FlavorResourceQuantities)
	for i, ps := range wl.TotalRequests {
		// 如果是部分准入，则缩放数量
		aps := a.PodSets[i]
		if aps.Count != ps.Count {
			ps = *ps.ScaledTo(aps.Count)
		}
		for res, q := range ps.Requests {
			flv := aps.Flavors[res].Name
			usage[resources.FlavorResource{Flavor: flv, Resource: res}] += q
		}
	}
	return usage
}

type Status struct {
	reasons []string
	err     error
}

func (s *Status) IsError() bool {
	return s != nil && s.err != nil
}

func (s *Status) appendf(format string, args ...any) *Status {
	s.reasons = append(s.reasons, fmt.Sprintf(format, args...))
	return s
}

func (s *Status) Message() string {
	if s == nil {
		return ""
	}
	if s.err != nil {
		return s.err.Error()
	}
	sort.Strings(s.reasons)
	return strings.Join(s.reasons, ", ")
}

func (s *Status) Equal(o *Status) bool {
	if s == nil || o == nil {
		return s == o
	}
	if s.err != nil {
		return errors.Is(s.err, o.err)
	}
	return cmp.Equal(s.reasons, o.reasons, cmpopts.SortSlices(func(a, b string) bool {
		return a < b
	}))
}

// PodSetAssignment 保存每个 pod set 的已分配 flavor 及每个资源的状态信息。每个分配的 flavor 都有一个 AssignmentMode。
// 空的 .Flavors 可解释为所有资源的 NoFit 模式。
// 空的 .Status 可解释为所有资源的 Fit 模式。
// 一旦 PodSetAssignment 完全计算，.Flavors 和 .Status 不能同时为空。
type PodSetAssignment struct {
	Name     kueue.PodSetReference
	Flavors  ResourceAssignment
	Status   *Status
	Requests corev1.ResourceList
	Count    int32

	TopologyAssignment     *kueue.TopologyAssignment
	DelayedTopologyRequest *kueue.DelayedTopologyRequestState
}

// RepresentativeMode 计算该分配的代表性分配模式，即所有已分配 flavor 中最差的模式。
func (psa *PodSetAssignment) RepresentativeMode() FlavorAssignmentMode {
	if psa.Status == nil {
		return Fit
	}
	if len(psa.Flavors) == 0 {
		return NoFit
	}
	mode := Fit
	for _, flvAssignment := range psa.Flavors {
		if flvAssignment.Mode < mode {
			mode = flvAssignment.Mode
		}
	}
	return mode
}

func (psa *PodSetAssignment) updateMode(newMode FlavorAssignmentMode) {
	for _, flvAssignment := range psa.Flavors {
		flvAssignment.Mode = newMode
	}
}

func (psa *PodSetAssignment) reason(reason string) {
	if psa.Status == nil {
		psa.Status = &Status{}
	}
	psa.Status.reasons = append(psa.Status.reasons, reason)
}

func (psa *PodSetAssignment) error(err error) {
	if psa.Status == nil {
		psa.Status = &Status{}
	}
	psa.Status.err = err
}

type ResourceAssignment map[corev1.ResourceName]*FlavorAssignment

func (psa *PodSetAssignment) toAPI() kueue.PodSetAssignment {
	flavors := make(map[corev1.ResourceName]kueue.ResourceFlavorReference, len(psa.Flavors))
	for res, flvAssignment := range psa.Flavors {
		flavors[res] = flvAssignment.Name
	}
	return kueue.PodSetAssignment{
		Name:                   psa.Name,
		Flavors:                flavors,
		ResourceUsage:          psa.Requests,
		Count:                  ptr.To(psa.Count),
		TopologyAssignment:     psa.TopologyAssignment.DeepCopy(),
		DelayedTopologyRequest: psa.DelayedTopologyRequest,
	}
}

// FlavorAssignmentMode 描述 flavor 是否可以立即分配，或需要什么条件才能分配。
type FlavorAssignmentMode int

// 下面的 flavor 分配模式按优先级从低到高排序。
const (
	// NoFit 表示没有足够的配额分配该 flavor，或需要抢占但已在借用，且策略不允许。
	NoFit FlavorAssignmentMode = iota
	// Preempt 表示根据配额可以准入。
	// 但由于策略/限制/优先级，抢占可能不可行。
	Preempt
	// Fit 表示有足够的未使用配额分配给该 flavor，无需抢占，可能需要借用。
	Fit
)

func (m FlavorAssignmentMode) String() string {
	switch m {
	case NoFit:
		return "NoFit"
	case Preempt:
		return "Preempt"
	case Fit:
		return "Fit"
	}
	return "Unknown"
}

// granularMode 是 FlavorAssigner 的内部分配模式，用于区分基于优先级的抢占和 cohort 内的回收。
type granularMode int

func (mode granularMode) flavorAssignmentMode() FlavorAssignmentMode {
	switch mode {
	case noFit:
		return NoFit
	case noPreemptionCandidates:
		return Preempt
	case preempt:
		return Preempt
	case reclaim:
		return Preempt
	case fit:
		return Fit
	default:
		panic("illegal state")
	}
}

// isPreemptMode 表示该模式下找到了抢占目标。
func (mode granularMode) isPreemptMode() bool {
	return mode == preempt || mode == reclaim
}

// FlavorAssignment 结构体保存 flavor 分配的详细信息。
type FlavorAssignment struct {
	Name           kueue.ResourceFlavorReference
	Mode           FlavorAssignmentMode
	TriedFlavorIdx int
	borrow         int
}

type preemptionOracle interface {
	SimulatePreemption(log logr.Logger, cq *cache.ClusterQueueSnapshot, wl workload.Info, fr resources.FlavorResource, quantity int64) preemptioncommon.PreemptionPossibility
}

type FlavorAssigner struct {
	wl                *workload.Info
	cq                *cache.ClusterQueueSnapshot
	resourceFlavors   map[kueue.ResourceFlavorReference]*kueue.ResourceFlavor
	enableFairSharing bool
	oracle            preemptionOracle
}

// New 创建一个 FlavorAssigner 实例。
func New(wl *workload.Info, cq *cache.ClusterQueueSnapshot, resourceFlavors map[kueue.ResourceFlavorReference]*kueue.ResourceFlavor, enableFairSharing bool, oracle preemptionOracle) *FlavorAssigner {
	return &FlavorAssigner{
		wl:                wl,
		cq:                cq,
		resourceFlavors:   resourceFlavors,
		enableFairSharing: enableFairSharing,
		oracle:            oracle,
	}
}

// lastAssignmentOutdated 判断上次分配是否已过时。
func lastAssignmentOutdated(wl *workload.Info, cq *cache.ClusterQueueSnapshot) bool {
	return cq.AllocatableResourceGeneration > wl.LastAssignment.ClusterQueueGeneration
}

func (psa *PodSetAssignment) append(flavors ResourceAssignment, status *Status) {
	for resource, assignment := range flavors {
		psa.Flavors[resource] = assignment
	}
	if psa.Status == nil {
		psa.Status = status
	} else if status != nil {
		psa.Status.reasons = append(psa.Status.reasons, status.reasons...)
	}
}

func (a *Assignment) append(requests resources.Requests, psAssignment *PodSetAssignment) {
	flavorIdx := make(map[corev1.ResourceName]int, len(psAssignment.Flavors))
	a.PodSets = append(a.PodSets, *psAssignment)
	for resource, flvAssignment := range psAssignment.Flavors {
		if flvAssignment.borrow > a.Borrowing {
			a.Borrowing = flvAssignment.borrow
		}
		fr := resources.FlavorResource{Flavor: flvAssignment.Name, Resource: resource}
		a.Usage.Quota[fr] += requests[resource]
		flavorIdx[resource] = flvAssignment.TriedFlavorIdx
	}
	a.LastState.LastTriedFlavorIdx = append(a.LastState.LastTriedFlavorIdx, flavorIdx)
}

// filterRequestedResources 过滤请求资源，仅保留 allowList 中的资源。
func filterRequestedResources(req resources.Requests, allowList sets.Set[corev1.ResourceName]) resources.Requests {
	filtered := make(resources.Requests)
	for n, v := range req {
		if allowList.Has(n) {
			filtered[n] = v
		}
	}
	return filtered
}

// flavorSelector 生成 flavor 的节点亲和性选择器。
func flavorSelector(spec *corev1.PodSpec, allowedKeys sets.Set[string]) nodeaffinity.RequiredNodeAffinity {
	// 此函数大致复刻了 kube-scheduler v1.24 的 NodeAffinity Filter 插件实现。
	var specCopy corev1.PodSpec

	// 移除与无关 key 相关的亲和性约束。
	if len(spec.NodeSelector) != 0 {
		specCopy.NodeSelector = map[string]string{} // pod 上写的 NodeSelector 与flavor 的 NodeSelector   取交集 ✅
		for k, v := range spec.NodeSelector {
			if allowedKeys.Has(k) { //  // rg下flavor的 NodeLabels key
				specCopy.NodeSelector[k] = v
			}
		}
	}
	// 节点亲和性 的NodeSelector 也使用 flavor 的NodeSelector
	affinity := spec.Affinity
	if affinity != nil && affinity.NodeAffinity != nil && affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
		var termsCopy []corev1.NodeSelectorTerm
		for _, t := range affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
			var expCopy []corev1.NodeSelectorRequirement
			for _, e := range t.MatchExpressions {
				if allowedKeys.Has(e.Key) {
					expCopy = append(expCopy, e)
				}
			}
			// 如果 term 变为空，则表示节点亲和性匹配任意 flavor，因为这些 term 是 OR 关系，
			// 匹配将退化为 spec.NodeSelector
			if len(expCopy) == 0 {
				termsCopy = nil
				break
			}
			termsCopy = append(termsCopy, corev1.NodeSelectorTerm{MatchExpressions: expCopy})
		}
		if len(termsCopy) != 0 {
			specCopy.Affinity = &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: termsCopy,
					},
				},
			}
		}
	}
	return nodeaffinity.GetRequiredNodeAffinity(&corev1.Pod{Spec: specCopy})
}

// fitsResourceQuota 返回该 flavor 是否可以分配给资源，
// 依据 ClusterQueue 和 cohort 中剩余配额。
// 如果适合，还会返回是否需要借用。同样，在抢占时也会返回是否需要借用的信息。
// 如果 flavor 不能立即满足限制（等待或抢占可能有帮助），则返回包含原因的 Status。
func (a *FlavorAssigner) fitsResourceQuota(log logr.Logger, fr resources.FlavorResource, val int64, rQuota cache.ResourceQuota) (mode granularMode, brow int, s *Status) {
	var status Status

	available := a.cq.Available(fr)            // ✅
	maxCapacity := a.cq.PotentialAvailable(fr) // ✅

	// No Fit
	if val > maxCapacity {
		status.appendf("insufficient quota for %s in flavor %s, request > maximum capacity (%s > %s)",
			fr.Resource, fr.Flavor, resources.ResourceQuantityString(fr.Resource, val), resources.ResourceQuantityString(fr.Resource, maxCapacity))
		return noFit, 0, &status
	}

	borrow, mayReclaimInHierarchy := classical.FindHeightOfLowestSubtreeThatFits(a.cq, fr, val)
	// Fit
	if val <= available {
		return fit, borrow, nil
	}

	// Preempt
	status.appendf("insufficient unused quota for %s in flavor %s, %s more needed", fr.Resource, fr.Flavor, resources.ResourceQuantityString(fr.Resource, val-available))

	if val <= rQuota.Nominal || mayReclaimInHierarchy || a.canPreemptWhileBorrowing() {
		mode := fromPreemptionPossibility(a.oracle.SimulatePreemption(log, a.cq, *a.wl, fr, val))
		return mode, borrow, &status
	}
	return noFit, borrow, &status
}

// canPreemptWhileBorrowing 判断在借用时是否可以抢占。
func (a *FlavorAssigner) canPreemptWhileBorrowing() bool {
	// 局部开启、全局开启
	return (a.cq.Preemption.BorrowWithinCohort != nil && a.cq.Preemption.BorrowWithinCohort.Policy != kueue.BorrowWithinCohortPolicyNever) ||
		(a.enableFairSharing && a.cq.Preemption.ReclaimWithinCohort != kueue.PreemptionPolicyNever)
}

const (
	noFit                  granularMode = iota
	noPreemptionCandidates              // 表示可以通过抢占准入，但模拟未找到抢占目标。
	preempt
	reclaim
	fit
)

func fromPreemptionPossibility(preemptionPossibility preemptioncommon.PreemptionPossibility) granularMode {
	switch preemptionPossibility {
	case preemptioncommon.NoCandidates:
		return noPreemptionCandidates
	case preemptioncommon.Preempt:
		return preempt
	case preemptioncommon.Reclaim:
		return reclaim
	}
	panic("illegal state")
}

// shouldTryNextFlavor 判断是否应尝试下一个 flavor。
func shouldTryNextFlavor(representativeMode granularMode, flavorFungibility kueue.FlavorFungibility, needsBorrowing bool) bool {
	policyPreempt := flavorFungibility.WhenCanPreempt
	policyBorrow := flavorFungibility.WhenCanBorrow
	if representativeMode.isPreemptMode() && policyPreempt == kueue.Preempt {
		if !needsBorrowing || policyBorrow == kueue.Borrow {
			return false
		}
	}

	if representativeMode == fit && needsBorrowing && policyBorrow == kueue.Borrow {
		return false
	}

	if representativeMode == fit && !needsBorrowing {
		return false
	}

	return true
}

// findFlavorForPodSetResource 查找能满足 podSet 请求的 flavor，适用于 resName 所在的资源组的所有资源。
// 返回选择的 flavor 及需要借用的资源信息。
// 如果 flavor 不能立即分配，则返回包含原因或失败信息的 status。
func (a *FlavorAssigner) findFlavorForPodSetResource(
	log logr.Logger,
	psID int,
	requests resources.Requests,
	resName corev1.ResourceName,
	assignmentUsage resources.FlavorResourceQuantities,
) (ResourceAssignment, *Status) {
	resourceGroup := a.cq.RGByResource(resName)
	if resourceGroup == nil {
		return nil, &Status{
			reasons: []string{fmt.Sprintf("resource %s unavailable in ClusterQueue", resName)},
		}
	}

	status := &Status{}
	requests = filterRequestedResources(requests, resourceGroup.CoveredResources) // 处理在 CoveredResources 中存在的资源
	ps := &a.wl.Obj.Spec.PodSets[psID]
	podSpec := &ps.Template.Spec

	var bestAssignment ResourceAssignment
	bestAssignmentMode := noFit

	// 只检查资源的 flavor 标签。
	selector := flavorSelector(podSpec, resourceGroup.LabelKeys) // rg下所有 flavor的 NodeLabels key
	attemptedFlavorIdx := -1
	idx := a.wl.LastAssignment.NextFlavorToTryForPodSetResource(psID, resName)
	for ; idx < len(resourceGroup.Flavors); idx++ {
		attemptedFlavorIdx = idx
		fName := resourceGroup.Flavors[idx]
		flavor, exist := a.resourceFlavors[fName]
		if !exist {
			log.Error(nil, "Flavor not found", "Flavor", fName)
			status.appendf("flavor %s not found", fName)
			continue
		}
		if features.Enabled(features.TopologyAwareScheduling) {
			// 检查podSet的拓扑 与 flavor 的拓扑是否一致
			if message := checkPodSetAndFlavorMatchForTAS(a.cq, ps, flavor); message != nil {
				log.Error(nil, *message)
				status.appendf("%s", *message)
				continue
			}
		}
		// pod 的容忍度、flavor 的容忍度   是否可以忽略   污点
		taint, untolerated := corev1helpers.FindMatchingUntoleratedTaint(flavor.Spec.NodeTaints, append(podSpec.Tolerations, flavor.Spec.Tolerations...), func(t *corev1.Taint) bool {
			return t.Effect == corev1.TaintEffectNoSchedule || t.Effect == corev1.TaintEffectNoExecute
		})
		if untolerated {
			status.appendf("untolerated taint %s in flavor %s", taint, fName)
			continue
		}
		//    flavor1 k1=v1 ,flavor2 k2=v2       selector k1     flavor2
		if match, err := selector.Match(&corev1.Node{ObjectMeta: metav1.ObjectMeta{Labels: flavor.Spec.NodeLabels}}); !match || err != nil {
			//之所有有这个选项，是因为可以借其他flavor 的资源使用；  所有他也要满足其他flavor 的 nodelabel
			if err != nil {
				status.err = err
				return nil, status
			}
			status.appendf("flavor %s doesn't match node affinity", fName)
			continue
		}
		needsBorrowing := false
		assignments := make(ResourceAssignment, len(requests))
		// 计算该分配的代表性模式，即所有请求中最差的模式。
		representativeMode := fit
		for rName, val := range requests { // 判断这个flavor 是否满足资源需求
			resQuota := a.cq.QuotaFor(resources.FlavorResource{Flavor: fName, Resource: rName})
			// 考虑前面 pod set 的 flavor 使用量。
			fr := resources.FlavorResource{Flavor: fName, Resource: rName}
			// borrow 最大借几层
			mode, borrow, s := a.fitsResourceQuota(log, fr, val+assignmentUsage[fr], resQuota)
			if s != nil {
				status.reasons = append(status.reasons, s.reasons...)
			}
			if mode < representativeMode {
				representativeMode = mode
			}
			needsBorrowing = needsBorrowing || (borrow > 0) // 借用
			if representativeMode == noFit {
				// 该 flavor 不适合，无需检查其他资源。
				break
			}

			assignments[rName] = &FlavorAssignment{
				Name:   fName,
				Mode:   mode.flavorAssignmentMode(),
				borrow: borrow,
			}
		}

		if features.Enabled(features.FlavorFungibility) {
			//todo  太复杂了
			if !shouldTryNextFlavor(representativeMode, a.cq.FlavorFungibility, needsBorrowing) {
				bestAssignment = assignments
				bestAssignmentMode = representativeMode
				break
			}
			if representativeMode > bestAssignmentMode {
				bestAssignment = assignments
				bestAssignmentMode = representativeMode
			}
		} else if representativeMode > bestAssignmentMode {
			bestAssignment = assignments
			bestAssignmentMode = representativeMode
			if bestAssignmentMode == fit {
				// 所有资源都适合该 cohort，无需检查更多 flavor。
				return bestAssignment, nil
			}
		}
	}

	if features.Enabled(features.FlavorFungibility) {
		for _, assignment := range bestAssignment {
			if attemptedFlavorIdx == len(resourceGroup.Flavors)-1 {
				// 已到最后一个 flavor，下次从第一个 flavor 开始尝试
				assignment.TriedFlavorIdx = -1
			} else {
				assignment.TriedFlavorIdx = attemptedFlavorIdx
			}
		}
		if bestAssignmentMode == fit {
			return bestAssignment, nil
		}
	}
	return bestAssignment, status
}

func (a *FlavorAssigner) assignFlavors(log logr.Logger, counts []int32) Assignment {
	var requests []workload.PodSetResources
	if len(counts) == 0 {
		requests = a.wl.TotalRequests
	} else {
		requests = make([]workload.PodSetResources, len(a.wl.TotalRequests))
		for i := range a.wl.TotalRequests {
			requests[i] = *a.wl.TotalRequests[i].ScaledTo(counts[i])
		}
	}
	assignment := Assignment{ // 转让的资源
		PodSets: make([]PodSetAssignment, 0, len(requests)),
		Usage: workload.Usage{
			Quota: make(resources.FlavorResourceQuantities),
		},
		LastState: workload.AssignmentClusterQueueState{
			LastTriedFlavorIdx:     make([]map[corev1.ResourceName]int, 0, len(requests)),
			ClusterQueueGeneration: a.cq.AllocatableResourceGeneration,
		},
	}

	for i, podSet := range requests {
		if a.cq.RGByResource(corev1.ResourcePods) != nil {
			podSet.Requests[corev1.ResourcePods] = int64(podSet.Count)
		}

		psAssignment := PodSetAssignment{
			Name:     podSet.Name,
			Flavors:  make(ResourceAssignment, len(podSet.Requests)),
			Requests: podSet.Requests.ToResourceList(),
			Count:    podSet.Count,
		}

		if features.Enabled(features.TopologyAwareScheduling) {
			//当工作负载被分配时，会进行相应的填充操作。   尊重现有的分配设置。 如果这是调度器的第二次运行，则 PodSet 分配可能已经设置好。
			for resName, fName := range podSet.Flavors {
				psAssignment.Flavors[resName] = &FlavorAssignment{
					Name: fName,
					Mode: Fit,
				}
			}
			if podSet.DelayedTopologyRequest != nil {
				psAssignment.DelayedTopologyRequest = ptr.To(*podSet.DelayedTopologyRequest)
			}
			if podSet.TopologyRequest != nil {
				psAssignment.TopologyAssignment = a.wl.Obj.Status.Admission.PodSetAssignments[i].TopologyAssignment
			}
		}

		for resName := range podSet.Requests {
			if _, found := psAssignment.Flavors[resName]; found {
				// 此资源被赋予了与其资源组相同的属性。因此无需再次进行计算。
				continue
			}
			flavors, status := a.findFlavorForPodSetResource(log, i, podSet.Requests, resName, assignment.Usage.Quota)
			if status.IsError() || len(flavors) == 0 {
				psAssignment.Flavors = nil
				psAssignment.Status = status
				break
			}
			psAssignment.append(flavors, status)
		}

		assignment.append(podSet.Requests, &psAssignment)
		if psAssignment.Status.IsError() || (len(podSet.Requests) > 0 && len(psAssignment.Flavors) == 0) {
			return assignment
		}
	}
	if assignment.RepresentativeMode() == NoFit {
		return assignment
	}

	if features.Enabled(features.TopologyAwareScheduling) {
		tasRequests := assignment.WorkloadsTopologyRequests(a.wl, a.cq)
		if assignment.RepresentativeMode() == Fit {
			result := a.cq.FindTopologyAssignmentsForWorkload(tasRequests, false, a.wl.Obj)
			if failure := result.Failure(); failure != nil {
				// 至少有一个 PodSet 不满足
				psAssignment := assignment.podSetAssignmentByName(failure.PodSetName)
				psAssignment.reason(failure.Reason)
				// 更新所有 flavor 和代表性模式的分配模式
				psAssignment.updateMode(Preempt)
				assignment.representativeMode = ptr.To(Preempt)
			} else {
				// 所有 PodSets 都满足，我们只更新 TopologyAssignments
				assignment.UpdateForTASResult(result)
			}
		}
		if assignment.RepresentativeMode() == Preempt && !workload.HasNodeToReplace(a.wl.Obj) {
			// 如果正在寻找失败节点替换，则不抢占其他 workload
			result := a.cq.FindTopologyAssignmentsForWorkload(tasRequests, true, nil)
			if failure := result.Failure(); failure != nil {
				// 即使所有 workload 都抢占，至少有一个 PodSet 也不满足
				psAssignment := assignment.podSetAssignmentByName(failure.PodSetName)
				// 更新所有 flavor 和代表性模式的分配模式
				psAssignment.updateMode(NoFit)
				assignment.representativeMode = ptr.To(NoFit)
			}
		}
	}
	return assignment
}

// Assign 为每个 pod set 请求的每个资源分配 flavor。
// 每个 pod set 的结果都包含 flavor 不能立即分配的原因。
// 每个分配的 flavor 都有一个 FlavorAssignmentMode。
func (a *FlavorAssigner) Assign(log logr.Logger, counts []int32) Assignment {
	if a.wl.LastAssignment != nil && lastAssignmentOutdated(a.wl, a.cq) {
		if logV := log.V(6); logV.Enabled() {
			keysValues := []any{
				"cq.AllocatableResourceGeneration", a.cq.AllocatableResourceGeneration,
				"wl.LastAssignment.ClusterQueueGeneration", a.wl.LastAssignment.ClusterQueueGeneration,
			}
			logV.Info("Clearing Workload's last assignment because it was outdated", keysValues...)
		}
		a.wl.LastAssignment = nil
	}
	return a.assignFlavors(log, counts)
}

// RepresentativeMode 计算该分配的代表性分配模式，即所有 pod set 中最差的分配模式。
func (a *Assignment) RepresentativeMode() FlavorAssignmentMode {
	if len(a.PodSets) == 0 {
		// 没有计算分配。
		return NoFit
	}
	if a.representativeMode != nil {
		return *a.representativeMode
	}
	mode := Fit
	for _, ps := range a.PodSets {
		psMode := ps.RepresentativeMode()
		if psMode < mode {
			mode = psMode
		}
	}
	a.representativeMode = &mode
	return mode
}
