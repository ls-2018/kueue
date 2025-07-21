package hierarchy

import (
	"k8s.io/apimachinery/pkg/util/sets"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
)

type CycleCheckable interface {
	GetName() kueue.CohortReference
	HasParent() bool
	CCParent() CycleCheckable
}

func HasCycle(cohort CycleCheckable) bool {
	return hasCycle(cohort, sets.New[kueue.CohortReference]())
}

func hasCycle(cohort CycleCheckable, seen sets.Set[kueue.CohortReference]) bool {
	if !cohort.HasParent() {
		return false
	}
	if seen.Has(cohort.GetName()) {
		return true
	}
	seen.Insert(cohort.GetName())
	return hasCycle(cohort.CCParent(), seen)
}
