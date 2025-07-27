package hierarchy

import (
	"k8s.io/apimachinery/pkg/util/sets"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
)

//lint:ignore U1000 due to https://github.com/dominikh/go-tools/issues/1602.
type Cohort[CQ clusterQueueNode[C], C nodeBase[kueue.CohortReference]] struct {
	parent       C
	childCohorts sets.Set[C]
	childCqs     sets.Set[CQ]
	explicit     bool // 表示此群组是否由一个 API 对象支持。
}

func (c *Cohort[CQ, C]) Parent() C {
	return c.parent
}

func (c *Cohort[CQ, C]) HasParent() bool {
	var zero C
	return c.Parent() != zero
}

func (c *Cohort[CQ, C]) ChildCQs() []CQ {
	return c.childCqs.UnsortedList()
}

// ChildCount returns number of Cohorts + ClusterQueues.
func (c *Cohort[CQ, C]) ChildCount() int {
	return c.childCohorts.Len() + c.childCqs.Len()
}

func NewCohort[CQ clusterQueueNode[C], C nodeBase[kueue.CohortReference]]() Cohort[CQ, C] {
	return Cohort[CQ, C]{
		childCohorts: sets.New[C](),
		childCqs:     sets.New[CQ](),
	}
}

// implement cohortNode interface

func (c *Cohort[CQ, C]) setParent(cohort C) {
	c.parent = cohort
}

func (c *Cohort[CQ, C]) insertCohortChild(cohort C) {
	c.childCohorts.Insert(cohort)
}

func (c *Cohort[CQ, C]) deleteCohortChild(cohort C) {
	c.childCohorts.Delete(cohort)
}

func (c *Cohort[CQ, C]) insertClusterQueue(cq CQ) {
	c.childCqs.Insert(cq)
}

func (c *Cohort[CQ, C]) hasChildren() bool {
	return c.childCqs.Len()+c.childCohorts.Len() > 0
}

func (c *Cohort[CQ, C]) isExplicit() bool {
	return c.explicit
}

func (c *Cohort[CQ, C]) markExplicit() {
	c.explicit = true
}
func (c *Cohort[CQ, C]) deleteClusterQueue(cq CQ) {
	c.childCqs.Delete(cq)
}
func (c *Cohort[CQ, C]) ChildCohorts() []C {
	return c.childCohorts.UnsortedList()
}
