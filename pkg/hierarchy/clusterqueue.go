package hierarchy

import kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"

type ClusterQueue[C nodeBase[kueue.CohortReference]] struct {
	cohort C
}

func (c *ClusterQueue[C]) Parent() C {
	return c.cohort
}

func (c *ClusterQueue[C]) HasParent() bool {
	var zero C
	return c.Parent() != zero
}

// implements clusterQueueNode interface

func (c *ClusterQueue[C]) setParent(cohort C) {
	c.cohort = cohort
}
