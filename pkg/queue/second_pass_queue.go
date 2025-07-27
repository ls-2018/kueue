package queue

import (
	"sync"

	"k8s.io/apimachinery/pkg/util/sets"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/workload"
)

type secondPassQueue struct {
	sync.RWMutex

	prequeued sets.Set[string]
	queued    map[string]*workload.Info
}

func newSecondPassQueue() *secondPassQueue {
	return &secondPassQueue{
		prequeued: sets.New[string](),
		queued:    make(map[string]*workload.Info),
	}
}

func (q *secondPassQueue) takeAllReady() []workload.Info {
	q.Lock()
	defer q.Unlock()

	var result []workload.Info
	for _, v := range q.queued {
		result = append(result, *v)
	}
	q.queued = make(map[string]*workload.Info)
	return result
}

func (q *secondPassQueue) prequeue(obj *kueue.Workload) {
	q.Lock()
	defer q.Unlock()

	q.prequeued.Insert(workload.Key(obj))
}

func (q *secondPassQueue) queue(w *workload.Info) bool {
	q.Lock()
	defer q.Unlock()

	key := workload.Key(w.Obj)
	enqueued := q.prequeued.Has(key) && workload.NeedsSecondPass(w.Obj)
	if enqueued {
		q.queued[key] = w
	}
	q.prequeued.Delete(key)
	return enqueued
}

func (q *secondPassQueue) deleteByKey(key string) {
	q.Lock()
	defer q.Unlock()

	delete(q.queued, key)
	q.prequeued.Delete(key)
}
