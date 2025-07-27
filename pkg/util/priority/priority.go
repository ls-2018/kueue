package priority

import (
	"context"

	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/constants"
)

// Priority 返回给定工作负载的优先级。
func Priority(w *kueue.Workload) int32 {
	// 当正在运行的工作负载的优先级为 nil 时，表示其创建时没有全局默认优先级类，且 pod 的优先级类名称为空。因此，使用静态默认优先级。
	return ptr.Deref(w.Spec.Priority, constants.DefaultPriority)
}

// GetPriorityFromWorkloadPriorityClass 返回从工作负载优先级类获取的优先级。如果未指定，则返回 0。
// 此函数内部不会调用 DefaultPriority，因为应接着检查 k8s 的优先级类。
func GetPriorityFromWorkloadPriorityClass(ctx context.Context, client client.Client,
	workloadPriorityClass string) (string, string, int32, error) {
	wpc := &kueue.WorkloadPriorityClass{}
	if err := client.Get(ctx, types.NamespacedName{Name: workloadPriorityClass}, wpc); err != nil {
		return "", "", 0, err
	}
	return wpc.Name, constants.WorkloadPriorityClassSource, wpc.Value, nil
}

// GetPriorityFromPriorityClass 返回从优先级类获取的优先级。如果未指定，则优先级为默认值，
// 如果没有默认值则为 0。
func GetPriorityFromPriorityClass(ctx context.Context, client client.Client,
	priorityClass string) (string, string, int32, error) {
	if len(priorityClass) == 0 {
		return getDefaultPriority(ctx, client)
	}

	pc := &schedulingv1.PriorityClass{}
	if err := client.Get(ctx, types.NamespacedName{Name: priorityClass}, pc); err != nil {
		return "", "", 0, err
	}

	return pc.Name, constants.PodPriorityClassSource, pc.Value, nil
}
func getDefaultPriority(ctx context.Context, client client.Client) (string, string, int32, error) {
	dpc, err := getDefaultPriorityClass(ctx, client)
	if err != nil {
		return "", "", 0, err
	}
	if dpc != nil {
		return dpc.Name, constants.PodPriorityClassSource, dpc.Value, nil
	}
	return "", "", int32(constants.DefaultPriority), nil
}

func getDefaultPriorityClass(ctx context.Context, client client.Client) (*schedulingv1.PriorityClass, error) {
	pcs := schedulingv1.PriorityClassList{}
	err := client.List(ctx, &pcs)
	if err != nil {
		return nil, err
	}

	// 如果由于竞争条件导致添加了多个全局默认优先级类，则选择优先级值最低的一个。
	var defaultPC *schedulingv1.PriorityClass
	for _, item := range pcs.Items {
		if item.GlobalDefault {
			if defaultPC == nil || defaultPC.Value > item.Value {
				defaultPC = &item
			}
		}
	}

	return defaultPC, nil
}
