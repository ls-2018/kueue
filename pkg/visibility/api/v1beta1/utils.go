package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	visibility "sigs.k8s.io/kueue/apis/visibility/v1beta1"
	"sigs.k8s.io/kueue/pkg/workload"
)

func newPendingWorkload(wlInfo *workload.Info, positionInLq int32, positionInCq int) *visibility.PendingWorkload {
	ownerReferences := make([]metav1.OwnerReference, 0, len(wlInfo.Obj.OwnerReferences))
	for _, ref := range wlInfo.Obj.OwnerReferences {
		ownerReferences = append(ownerReferences, metav1.OwnerReference{
			APIVersion: ref.APIVersion,
			Kind:       ref.Kind,
			Name:       ref.Name,
			UID:        ref.UID,
		})
	}
	return &visibility.PendingWorkload{
		ObjectMeta: metav1.ObjectMeta{
			Name:              wlInfo.Obj.Name,
			Namespace:         wlInfo.Obj.Namespace,
			OwnerReferences:   ownerReferences,
			CreationTimestamp: wlInfo.Obj.CreationTimestamp,
		},
		PositionInClusterQueue: int32(positionInCq),
		Priority:               *wlInfo.Obj.Spec.Priority,
		LocalQueueName:         wlInfo.Obj.Spec.QueueName,
		PositionInLocalQueue:   positionInLq,
	}
}
