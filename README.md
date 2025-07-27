ResourceFlavor 定了匹配的节点、容忍度等信息

CQ -> LQ -> Workload

cohort  -> CQ
            -> Flavor
            -> 抢占策略
        -> Flavor -> 可以出借


- AddClusterQueue

- ReconcileGenericJob:
    - 删除多余的workload
    - 创建更新workload
    - 暂停启动 job、pod


- (r *Reconciler) constructWorkload
    - NewGroupWorkload
        - NewWorkload

- (r *JobReconciler) constructWorkload
    - NewGroupWorkload
        - NewWorkload
- (p *Pod) ConstructComposableWorkload
    - ConstructWorkload
        - NewWorkload
    - NewGroupWorkload
        - NewWorkload

job\pod ---> workload ---> localqueue
                      ---> clusterqueue



FindTopologyAssignmentsForWorkload







- func (w *wlReconciler) Reconcile(
    - grp, err := w.readGroup
        - reconcileGroup


















