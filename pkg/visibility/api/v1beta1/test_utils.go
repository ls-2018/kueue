package v1beta1

import (
	visibility "sigs.k8s.io/kueue/apis/visibility/v1beta1"
)

type req struct {
	nsName      string
	queueName   string
	queryParams *visibility.PendingWorkloadOptions
}

type resp struct {
	wantErr              error
	wantPendingWorkloads []visibility.PendingWorkload
}
