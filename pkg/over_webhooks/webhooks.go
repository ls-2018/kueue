package over_webhooks

import (
	ctrl "sigs.k8s.io/controller-runtime"
)

// Setup sets up the webhooks for core controllers. It returns the name of the
// webhook that failed to create and an error, if any.
func Setup(mgr ctrl.Manager) (string, error) {
	if err := setupWebhookForWorkload(mgr); err != nil {
		return "Workload", err
	}

	if err := setupWebhookForResourceFlavor(mgr); err != nil {
		return "ResourceFlavor", err
	}

	if err := setupWebhookForClusterQueue(mgr); err != nil {
		return "ClusterQueue", err
	}

	if err := setupWebhookForCohort(mgr); err != nil {
		return "Cohort", err
	}

	return "", nil
}
