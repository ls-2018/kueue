package over_queue

import (
	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/over_hierarchy"
)

// cohort is a set of ClusterQueues that can borrow resources from
// each other.
type cohort struct {
	Name kueue.CohortReference
	over_hierarchy.Cohort[*ClusterQueue, *cohort]
}

func newCohort(name kueue.CohortReference) *cohort {
	return &cohort{
		name,
		over_hierarchy.NewCohort[*ClusterQueue, *cohort](),
	}
}

func (c *cohort) GetName() kueue.CohortReference {
	return c.Name
}

// CCParent satisfies the CycleCheckable interface.
func (c *cohort) CCParent() over_hierarchy.CycleCheckable {
	return c.Parent()
}

func (c *cohort) getRootUnsafe() *cohort {
	if !c.HasParent() {
		return c
	}
	return c.Parent().getRootUnsafe()
}
