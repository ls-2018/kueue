package jobframework

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/cache"
	"sigs.k8s.io/kueue/pkg/controller/constants"
	"sigs.k8s.io/kueue/pkg/features"
	"sigs.k8s.io/kueue/pkg/queue"
)

func ApplyDefaultForManagedBy(jobOrPod GenericJob, queues *queue.Manager, cache *cache.Cache, log logr.Logger) {
	if managedJob, ok := jobOrPod.(JobWithManagedBy); ok {
		if managedJob.CanDefaultManagedBy() { // 是否被 k8s 管理
			localQueueName, found := jobOrPod.Object().GetLabels()[constants.QueueLabel]
			if !found {
				return
			} //ToDo 启用了 default queue   会加上这个label
			clusterQueueName, ok := queues.ClusterQueueFromLocalQueue(queue.NewLocalQueueReference(jobOrPod.Object().GetNamespace(), kueue.LocalQueueName(localQueueName)))
			if !ok {
				log.V(5).Info("Cluster queue for local queue not found", "localQueueName", localQueueName)
				return
			}
			for _, admissionCheck := range cache.AdmissionChecksForClusterQueue(clusterQueueName) {
				if admissionCheck.Controller == kueue.MultiKueueControllerName {
					log.V(5).Info("Defaulting ManagedBy", "oldManagedBy", managedJob.ManagedBy(), "managedBy", kueue.MultiKueueControllerName)
					managedJob.SetManagedBy(ptr.To(kueue.MultiKueueControllerName))
					return
				}
			}
		}
	}
}

func ApplyDefaultLocalQueue(jobObj client.Object, defaultQueueExist func(string) bool) {
	if !features.Enabled(features.LocalQueueDefaulting) || !defaultQueueExist(jobObj.GetNamespace()) {
		return
	}
	if QueueNameForObject(jobObj) == "" {
		// Do not default the queue-name for a job whose owner is already managed by Kueue
		if IsOwnerManagedByKueueForObject(jobObj) {
			return
		}
		labels := jobObj.GetLabels()
		if labels == nil {
			labels = make(map[string]string, 1)
		}
		labels[constants.QueueLabel] = string(constants.DefaultLocalQueueName)
		jobObj.SetLabels(labels)
	}
}

func ApplyDefaultForSuspend(ctx context.Context, jobOrPod GenericJob, k8sClient client.Client,
	manageJobsWithoutQueueName bool, managedJobsNamespaceSelector labels.Selector) error {
	//暂停
	suspend, err := WorkloadShouldBeSuspended(ctx, jobOrPod.Object(), k8sClient, manageJobsWithoutQueueName, managedJobsNamespaceSelector)
	if err != nil {
		return err
	}
	if suspend && !jobOrPod.IsSuspended() {
		jobOrPod.Suspend()
	}
	return nil
}

// WorkloadShouldBeSuspended 决定在创建时是否应将 jobObj 设置为默认暂停状态
func WorkloadShouldBeSuspended(ctx context.Context, jobObj client.Object, k8sClient client.Client, manageJobsWithoutQueueName bool, managedJobsNamespaceSelector labels.Selector) (bool, error) {
	// 不要对那些其父项已被 Kueue 管理的作业进行暂停操作
	ancestorJob, err := FindAncestorJobManagedByKueue(ctx, k8sClient, jobObj, manageJobsWithoutQueueName)
	if err != nil || ancestorJob != nil {
		return false, err
	}

	// Jobs with queue names whose parents are not managed by Kueue are default suspended
	if QueueNameForObject(jobObj) != "" {
		return true, nil
	}

	// Logic for managing jobs without queue names.
	if manageJobsWithoutQueueName {
		if features.Enabled(features.ManagedJobsNamespaceSelector) && managedJobsNamespaceSelector != nil {
			// Default suspend the job if the namespace selector matches
			ns := corev1.Namespace{}
			err := k8sClient.Get(ctx, client.ObjectKey{Name: jobObj.GetNamespace()}, &ns)
			if err != nil {
				return false, fmt.Errorf("failed to get namespace: %w", err)
			}
			return managedJobsNamespaceSelector.Matches(labels.Set(ns.GetLabels())), nil
		} else {
			// Namespace filtering is disabled; unconditionally default suspend
			return true, nil
		}
	}
	return false, nil
}
