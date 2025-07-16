package workload

import (
	"context"
	"fmt"
	ctrl "sigs.k8s.io/controller-runtime"

	corev1 "k8s.io/api/core/v1"
	nodev1 "k8s.io/api/node/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/controller/core/over_indexer"
	"sigs.k8s.io/kueue/pkg/util/limitrange"
	"sigs.k8s.io/kueue/pkg/util/resource"
)

var (
	PodSetsPath = field.NewPath("spec").Child("podSets")
)

const (
	RequestsMustNotExceedLimitMessage = "requests must not exceed its limits"
)

// ValidateResources validates that requested resources are less or equal
// to limits.
func ValidateResources(wi *Info) field.ErrorList {
	// requests should be less than limits.
	var allErrors field.ErrorList
	for i := range wi.Obj.Spec.PodSets {
		ps := &wi.Obj.Spec.PodSets[i]
		podSpecPath := PodSetsPath.Index(i).Child("template").Child("spec")
		for i := range ps.Template.Spec.InitContainers {
			c := ps.Template.Spec.InitContainers[i]
			if resNames := resource.GetGreaterKeys(c.Resources.Requests, c.Resources.Limits); len(resNames) > 0 {
				allErrors = append(
					allErrors,
					field.Invalid(podSpecPath.Child("initContainers").Index(i), resNames, RequestsMustNotExceedLimitMessage),
				)
			}
		}

		for i := range ps.Template.Spec.Containers {
			c := ps.Template.Spec.Containers[i]
			if resNames := resource.GetGreaterKeys(c.Resources.Requests, c.Resources.Limits); len(resNames) > 0 {
				allErrors = append(
					allErrors,
					field.Invalid(podSpecPath.Child("containers").Index(i), resNames, RequestsMustNotExceedLimitMessage),
				)
			}
		}
	}
	return allErrors
}

// ValidateLimitRange validates that the requested resources fit into the namespace defined
// limitRanges.
func ValidateLimitRange(ctx context.Context, c client.Client, wi *Info) field.ErrorList {
	var allErrs field.ErrorList
	limitRanges := corev1.LimitRangeList{}
	if err := c.List(ctx, &limitRanges, &client.ListOptions{Namespace: wi.Obj.Namespace}); err != nil {
		allErrs = append(allErrs, field.InternalError(field.NewPath(""), err))
		return allErrs
	}
	if len(limitRanges.Items) == 0 {
		return nil
	}
	summary := limitrange.Summarize(limitRanges.Items...)

	// verify
	for i := range wi.Obj.Spec.PodSets {
		ps := &wi.Obj.Spec.PodSets[i]
		allErrs = append(allErrs, summary.ValidatePodSpec(&ps.Template.Spec, PodSetsPath.Index(i).Child("template").Child("spec"))...)
	}
	return allErrs
}

// AdjustResources 根据以下因素调整工作负载的资源需求：
// - PodOverhead
// - LimitRanges
// - Limits
func AdjustResources(ctx context.Context, cl client.Client, wl *kueue.Workload) {
	log := ctrl.LoggerFrom(ctx)
	for _, err := range handlePodOverhead(ctx, cl, wl) {
		log.Error(err, "Failures adjusting requests for pod overhead")
	}
	if err := handlePodLimitRange(ctx, cl, wl); err != nil {
		log.Error(err, "Failed adjusting requests for LimitRanges")
	}
	handleLimitsToRequests(wl)
}

// 我们在此处不验证 Pod 的运行时类合法性，因为这一操作将在准入控制器中完成。
// 因此，Pod 的开销并非总是准确无误的。例如，如果我们将一个不存在的运行时类名称设置为 `pod.Spec.RuntimeClassName`，
// 并且还设置了 `pod.Spec.Overhead`，在实际情况下，由于与运行时类的不匹配，Pod 的创建将会被拒绝。然而，我们假设在未来它们是正确的。
func handlePodOverhead(ctx context.Context, cl client.Client, wl *kueue.Workload) []error {
	var errs []error
	for i := range wl.Spec.PodSets {
		podSpec := &wl.Spec.PodSets[i].Template.Spec
		if podSpec.RuntimeClassName != nil && len(podSpec.Overhead) == 0 {
			var runtimeClass nodev1.RuntimeClass
			if err := cl.Get(ctx, types.NamespacedName{Name: *podSpec.RuntimeClassName}, &runtimeClass); err != nil {
				errs = append(errs, fmt.Errorf("in podSet %s: %w", wl.Spec.PodSets[i].Name, err))
				continue
			}
			if runtimeClass.Overhead != nil {
				podSpec.Overhead = runtimeClass.Overhead.PodFixed
			}
		}
	}
	return errs
}
func handlePodLimitRange(ctx context.Context, cl client.Client, wl *kueue.Workload) error {
	// get the list of limit ranges
	var limitRanges corev1.LimitRangeList
	if err := cl.List(ctx, &limitRanges, &client.ListOptions{Namespace: wl.Namespace}, client.MatchingFields{over_indexer.LimitRangeHasContainerType: "true"}); err != nil {
		return err
	}

	if len(limitRanges.Items) == 0 {
		return nil
	}
	summary := limitrange.Summarize(limitRanges.Items...)
	containerLimits, found := summary[corev1.LimitTypeContainer]
	if !found {
		return nil
	}

	for pi := range wl.Spec.PodSets {
		pod := &wl.Spec.PodSets[pi].Template.Spec
		for ci := range pod.InitContainers {
			res := &pod.InitContainers[ci].Resources
			res.Limits = resource.MergeResourceListKeepFirst(res.Limits, containerLimits.Default)
			res.Requests = resource.MergeResourceListKeepFirst(res.Requests, containerLimits.DefaultRequest)
		}
		for ci := range pod.Containers {
			res := &pod.Containers[ci].Resources
			res.Limits = resource.MergeResourceListKeepFirst(res.Limits, containerLimits.Default)
			res.Requests = resource.MergeResourceListKeepFirst(res.Requests, containerLimits.DefaultRequest)
		}
	}
	return nil
}
func handleLimitsToRequests(wl *kueue.Workload) {
	for pi := range wl.Spec.PodSets {
		UseLimitsAsMissingRequestsInPod(&wl.Spec.PodSets[pi].Template.Spec)
	}
}

// UseLimitsAsMissingRequestsInPod adjust the resource requests to the limits value
// for resources that only set limits.
func UseLimitsAsMissingRequestsInPod(pod *corev1.PodSpec) {
	for ci := range pod.InitContainers {
		res := &pod.InitContainers[ci].Resources
		res.Requests = resource.MergeResourceListKeepFirst(res.Requests, res.Limits)
	}
	for ci := range pod.Containers {
		res := &pod.Containers[ci].Resources
		res.Requests = resource.MergeResourceListKeepFirst(res.Requests, res.Limits)
	}
}
