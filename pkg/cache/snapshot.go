package cache

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/features"
	"sigs.k8s.io/kueue/pkg/hierarchy"
	utilmaps "sigs.k8s.io/kueue/pkg/util/maps"
	"sigs.k8s.io/kueue/pkg/workload"
)

type Snapshot struct {
	hierarchy.Manager[*ClusterQueueSnapshot, *CohortSnapshot]
	ResourceFlavors          map[kueue.ResourceFlavorReference]*kueue.ResourceFlavor
	InactiveClusterQueueSets sets.Set[kueue.ClusterQueueReference]
}

// RemoveWorkload removes a workload from its corresponding ClusterQueue and
// updates resource usage.
// RemoveWorkload 从对应的 ClusterQueue 移除 workload 并更新资源使用量。
func (s *Snapshot) RemoveWorkload(wl *workload.Info) {
	cq := s.ClusterQueue(wl.ClusterQueue)
	delete(cq.Workloads, workload.Key(wl.Obj))
	cq.RemoveUsage(wl.Usage())
}

// AddWorkload adds a workload from its corresponding ClusterQueue and
// updates resource usage.
// AddWorkload 将 workload 添加到对应的 ClusterQueue 并更新资源使用量。
func (s *Snapshot) AddWorkload(wl *workload.Info) {
	cq := s.ClusterQueue(wl.ClusterQueue)
	cq.Workloads[workload.Key(wl.Obj)] = wl
	cq.AddUsage(wl.Usage())
}

// Log 打印快照信息。
func (s *Snapshot) Log(log logr.Logger) {
	for name, cq := range s.ClusterQueues() {
		cohortName := "<none>"
		if cq.HasParent() {
			cohortName = string(cq.Parent().Name)
		}

		log.Info("Found ClusterQueue",
			"clusterQueue", klog.KRef("", string(name)),
			"cohort", cohortName,
			"resourceGroups", cq.ResourceGroups,
			"usage", cq.ResourceNode.Usage,
			"workloads", slices.Collect(maps.Keys(cq.Workloads)),
		)
	}
	for name, cohort := range s.Cohorts() {
		log.Info("Found cohort",
			"cohort", name,
			"resources", cohort.ResourceNode.SubtreeQuota,
			"usage", cohort.ResourceNode.Usage,
		)
	}

	// Dump TAS snapshots if the feature is enabled
	if features.Enabled(features.TopologyAwareScheduling) {
		for cqName, cq := range s.ClusterQueues() {
			for tasFlavor, tasSnapshot := range cq.TASFlavors {
				freeCapacityPerDomain, err := tasSnapshot.SerializeFreeCapacityPerDomain()
				if err != nil {
					log.Error(err, "Failed to serialize TAS snapshot free capacity",
						"clusterQueue", cqName,
						"resourceFlavor", tasFlavor,
					)
					continue
				}

				log.Info("TAS Snapshot Free Capacity",
					"clusterQueue", cqName,
					"resourceFlavor", tasFlavor,
					"freeCapacityPerDomain", freeCapacityPerDomain,
				)
			}
		}
	}
}

var snap sync.Once

// Snapshot 生成当前缓存的快照。
func (c *Cache) Snapshot(ctx context.Context) (*Snapshot, error) {
	snap.Do(func() {
		go func() {
			for {
				time.Sleep(5 * time.Second)
				fmt.Println(c)
			}
		}()
	})
	c.RLock()
	defer c.RUnlock()

	snap := Snapshot{
		Manager:                  hierarchy.NewManager(newCohortSnapshot),
		ResourceFlavors:          make(map[kueue.ResourceFlavorReference]*kueue.ResourceFlavor, len(c.resourceFlavors)),
		InactiveClusterQueueSets: sets.New[kueue.ClusterQueueReference](),
	}
	for _, cohort := range c.hm.Cohorts() {
		if hierarchy.HasCycle(cohort) {
			continue
		}
		snap.AddCohort(cohort.Name)
		snap.Cohort(cohort.Name).ResourceNode = cohort.resourceNode.Clone()
		snap.Cohort(cohort.Name).FairWeight = cohort.FairWeight
		if cohort.HasParent() {
			snap.UpdateCohortEdge(cohort.Name, cohort.Parent().Name)
		}
	}
	tasSnapshots := make(map[kueue.ResourceFlavorReference]*TASFlavorSnapshot)
	if features.Enabled(features.TopologyAwareScheduling) {
		for flavor, cache := range c.tasCache.Clone() {
			s, err := cache.snapshot(ctx)
			if err != nil {
				return nil, fmt.Errorf("%w: failed to construct snapshot for TAS flavor: %q", err, flavor)
			} else {
				tasSnapshots[flavor] = s
			}
		}
	}
	for _, cq := range c.hm.ClusterQueues() {
		if !cq.Active() || (cq.HasParent() && hierarchy.HasCycle(cq.Parent())) {
			snap.InactiveClusterQueueSets.Insert(cq.Name)
			continue
		}
		cqSnapshot := snapshotClusterQueue(cq)
		snap.AddClusterQueue(cqSnapshot)
		if cq.HasParent() {
			snap.UpdateClusterQueueEdge(cq.Name, cq.Parent().Name)
		}
		if features.Enabled(features.TopologyAwareScheduling) {
			for tasFlv, s := range tasSnapshots {
				if cq.flavorInUse(tasFlv) {
					cqSnapshot.TASFlavors[tasFlv] = s
				}
			}
		}
	}
	// Shallow copy is enough
	maps.Copy(snap.ResourceFlavors, c.resourceFlavors)
	return &snap, nil
}

// snapshotClusterQueue 创建 ClusterQueue 的快照副本。
func snapshotClusterQueue(c *clusterQueue) *ClusterQueueSnapshot {
	cc := &ClusterQueueSnapshot{
		Name:                          c.Name,
		ResourceGroups:                make([]ResourceGroup, len(c.ResourceGroups)),
		FlavorFungibility:             c.FlavorFungibility,
		FairWeight:                    c.FairWeight,
		AllocatableResourceGeneration: c.AllocatableResourceGeneration,
		Workloads:                     maps.Clone(c.Workloads),
		Preemption:                    c.Preemption,
		NamespaceSelector:             c.NamespaceSelector,
		Status:                        c.Status,
		AdmissionChecks:               utilmaps.DeepCopySets(c.AdmissionChecks),
		ResourceNode:                  c.resourceNode.Clone(),
		TASFlavors:                    make(map[kueue.ResourceFlavorReference]*TASFlavorSnapshot),
		tasOnly:                       c.isTASOnly(),
		flavorsForProvReqACs:          c.flavorsWithProvReqAdmissionCheck(),
	}
	for i, rg := range c.ResourceGroups {
		cc.ResourceGroups[i] = rg.Clone()
	}
	return cc
}

// newCohortSnapshot 创建新的 CohortSnapshot。
func newCohortSnapshot(name kueue.CohortReference) *CohortSnapshot {
	return &CohortSnapshot{
		Name:   name,
		Cohort: hierarchy.NewCohort[*ClusterQueueSnapshot](),
	}
}
