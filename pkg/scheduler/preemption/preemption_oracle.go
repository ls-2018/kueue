package preemption

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/sets"

	"sigs.k8s.io/kueue/pkg/cache"
	"sigs.k8s.io/kueue/pkg/resources"
	preemptioncommon "sigs.k8s.io/kueue/pkg/scheduler/preemption/common"
	"sigs.k8s.io/kueue/pkg/workload"
)

func NewOracle(preemptor *Preemptor, snapshot *cache.Snapshot) *PreemptionOracle {
	return &PreemptionOracle{preemptor, snapshot}
}

type PreemptionOracle struct {
	preemptor *Preemptor
	snapshot  *cache.Snapshot
}

// SimulatePreemption 模拟和判断在 Kubernetes 调度系统中某个资源是否可以通过抢占（preemption）或回收（reclaim）来满足调度需求。
func (p *PreemptionOracle) SimulatePreemption(log logr.Logger, cq *cache.ClusterQueueSnapshot, wl workload.Info, fr resources.FlavorResource, quantity int64) preemptioncommon.PreemptionPossibility {
	//1 获取抢占候选
	//调用 preemptor.getTargets，传入当前上下文，获取可以被抢占的候选工作负载列表
	candidates := p.preemptor.getTargets(&preemptionCtx{ // 1 ✅
		log:               log,
		preemptor:         wl,
		preemptorCQ:       p.snapshot.ClusterQueue(wl.ClusterQueue),
		snapshot:          p.snapshot,
		frsNeedPreemption: sets.New(fr),
		workloadUsage:     workload.Usage{Quota: resources.FlavorResourceQuantities{fr: quantity}},
	})
	//2 无候选
	//如果没有候选（len(candidates) == 0），返回 NoCandidates，表示无法通过抢占或回收满足需求。
	if len(candidates) == 0 {
		return preemptioncommon.NoCandidates
	}
	//3 判断候选归属
	//遍历候选，如果有候选的 ClusterQueue 与目标 cq 相同，返回 Preempt，表示可以通过抢占本队列的工作负载来满足需求。
	for _, candidate := range candidates {
		if candidate.WorkloadInfo.ClusterQueue == cq.Name {
			return preemptioncommon.Preempt
		}
	}
	//4 如果没有同队列的候选，则返回 Reclaim，表示只能通过回收其他队列的资源来满足需求。
	return preemptioncommon.Reclaim
}
