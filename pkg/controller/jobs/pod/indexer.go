package pod

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	podconstants "sigs.k8s.io/kueue/pkg/controller/jobs/pod/constants"
)

const (
	PodGroupNameCacheKey = "PodGroupNameCacheKey"
)

func IndexPodGroupName(o client.Object) []string {
	pod, ok := o.(*corev1.Pod)
	if !ok {
		return nil
	}

	if labelValue, exists := pod.GetLabels()[podconstants.GroupNameLabel]; exists {
		return []string{labelValue}
	}
	return nil
}
