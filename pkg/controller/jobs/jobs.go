package jobs

// Reference the job framework integration packages to ensure linking.
import (
	_ "sigs.k8s.io/kueue/pkg/controller/jobs/deployment"
	_ "sigs.k8s.io/kueue/pkg/controller/jobs/job"
	_ "sigs.k8s.io/kueue/pkg/controller/jobs/leaderworkerset"
	_ "sigs.k8s.io/kueue/pkg/controller/jobs/pod"
	_ "sigs.k8s.io/kueue/pkg/controller/jobs/statefulset"
)
