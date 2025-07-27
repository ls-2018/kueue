package multikueue

import (
	"context"
	"errors"
	"fmt"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/controller/jobframework"
	"sigs.k8s.io/kueue/pkg/util/admissioncheck"
	"sigs.k8s.io/kueue/pkg/util/api"
	utilmaps "sigs.k8s.io/kueue/pkg/util/maps"
	"sigs.k8s.io/kueue/pkg/workload"
)

var (
	realClock           = clock.RealClock{}
	errNoActiveClusters = errors.New("no active clusters")
)

type wlReconciler struct {
	localclient       client.Client
	helper            *multiKueueStoreHelper
	clusters          *clustersReconciler
	origin            string
	workerLostTimeout time.Duration
	deletedWlCache    *utilmaps.SyncMap[string, *kueue.Workload]
	eventsBatchPeriod time.Duration
	adapters          map[string]jobframework.MultiKueueAdapter //   group/version/kind  :    todo  jobframework.RegisterIntegration(
	recorder          record.EventRecorder
	clock             clock.Clock
}

var _ reconcile.Reconciler = (*wlReconciler)(nil)

type wlGroup struct {
	local           *kueue.Workload
	remoteWorkloads map[string]*kueue.Workload
	remoteClients   map[string]*remoteClient
	acName          kueue.AdmissionCheckReference
	jobAdapter      jobframework.MultiKueueAdapter
	controllerKey   types.NamespacedName
}

type options struct {
	clock clock.Clock
}

type Option func(*options)

var defaultOptions = options{
	clock: realClock,
}

// FirstReserving 返回 true 表示有 workload 正在保留配额，字符串标识远程集群。
func (g *wlGroup) FirstReserving() (bool, string) {
	found := false
	bestMatch := ""
	var bestTime time.Time
	for remote, wl := range g.remoteWorkloads {
		if wl == nil {
			continue
		}
		c := apimeta.FindStatusCondition(wl.Status.Conditions, kueue.WorkloadQuotaReserved)
		if c != nil && c.Status == metav1.ConditionTrue && (!found || bestTime.IsZero() || c.LastTransitionTime.Time.Before(bestTime)) {
			found = true
			bestMatch = remote
			bestTime = c.LastTransitionTime.Time
		}
	}
	return found, bestMatch
}

func (w *wlReconciler) updateACS(ctx context.Context, wl *kueue.Workload, acs *kueue.AdmissionCheckState, status kueue.CheckState, message string) error {
	acs.State = status
	acs.Message = message
	acs.LastTransitionTime = metav1.NewTime(w.clock.Now())
	wlPatch := workload.BaseSSAWorkload(wl)
	workload.SetAdmissionCheckState(&wlPatch.Status.AdmissionChecks, *acs, w.clock)
	return w.localclient.Status().Patch(ctx, wlPatch, client.Apply, client.FieldOwner(kueue.MultiKueueControllerName), client.ForceOwnership)
}

func (w *wlReconciler) Create(_ event.CreateEvent) bool {
	return true
}

func (w *wlReconciler) Delete(de event.DeleteEvent) bool {
	if wl, isWl := de.Object.(*kueue.Workload); isWl && !de.DeleteStateUnknown {
		w.deletedWlCache.Add(client.ObjectKeyFromObject(wl).String(), wl)
	}
	return true
}

func (w *wlReconciler) Update(_ event.UpdateEvent) bool {
	return true
}

func (w *wlReconciler) Generic(_ event.GenericEvent) bool {
	return true
}

func newWlReconciler(c client.Client, helper *multiKueueStoreHelper, cRec *clustersReconciler, origin string, recorder record.EventRecorder, workerLostTimeout, eventsBatchPeriod time.Duration, adapters map[string]jobframework.MultiKueueAdapter, opts ...Option) *wlReconciler {
	options := defaultOptions

	for _, opt := range opts {
		opt(&options)
	}

	return &wlReconciler{
		localclient:       c,
		helper:            helper,
		clusters:          cRec,
		origin:            origin,
		workerLostTimeout: workerLostTimeout,
		deletedWlCache:    utilmaps.NewSyncMap[string, *kueue.Workload](0),
		eventsBatchPeriod: eventsBatchPeriod,
		adapters:          adapters,
		recorder:          recorder,
		clock:             options.clock,
	}
}

func cloneForCreate(orig *kueue.Workload, origin string) *kueue.Workload {
	remoteWl := &kueue.Workload{}
	remoteWl.ObjectMeta = api.CloneObjectMetaForCreation(&orig.ObjectMeta)
	if remoteWl.Labels == nil {
		remoteWl.Labels = make(map[string]string)
	}
	remoteWl.Labels[kueue.MultiKueueOriginLabel] = origin
	orig.Spec.DeepCopyInto(&remoteWl.Spec)
	return remoteWl
}

func (w *wlReconciler) setupWithManager(mgr ctrl.Manager) error {
	syncHndl := handler.Funcs{
		GenericFunc: func(_ context.Context, e event.GenericEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			q.AddAfter(reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: e.Object.GetNamespace(),
				Name:      e.Object.GetName(),
			}}, w.eventsBatchPeriod)
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named("multikueue_workload").
		For(&kueue.Workload{}).
		WatchesRawSource(source.Channel(w.clusters.wlUpdateCh, syncHndl)).
		WithEventFilter(w).
		Complete(w)
}
func (w *wlReconciler) multikueueAC(ctx context.Context, local *kueue.Workload) (*kueue.AdmissionCheckState, error) {
	// 正确的检查
	relevantChecks, err := admissioncheck.FilterForController(ctx, w.localclient, local.Status.AdmissionChecks, kueue.MultiKueueControllerName) // ✅
	if err != nil {
		return nil, err
	}

	if len(relevantChecks) == 0 {
		return nil, nil
	}
	return workload.FindAdmissionCheck(local.Status.AdmissionChecks, relevantChecks[0]), nil
}

func (w *wlReconciler) adapter(local *kueue.Workload) (jobframework.MultiKueueAdapter, *metav1.OwnerReference) {
	if controller := metav1.GetControllerOf(local); controller != nil {
		adapterKey := schema.FromAPIVersionAndKind(controller.APIVersion, controller.Kind).String()
		return w.adapters[adapterKey], controller
	} else if refs := local.GetOwnerReferences(); len(refs) > 0 {
		// For workloads without a controller but with owner references,
		// use the first owner reference to find the adapter. This supports composable workloads.
		adapterKey := schema.FromAPIVersionAndKind(refs[0].APIVersion, refs[0].Kind).String()
		return w.adapters[adapterKey], &refs[0]
	}
	return nil, nil
}

func (w *wlReconciler) readGroup(ctx context.Context, local *kueue.Workload, acName kueue.AdmissionCheckReference, adapter jobframework.MultiKueueAdapter, controllerName string) (*wlGroup, error) {
	rClients, err := w.remoteClientsForAC(ctx, acName)
	if err != nil {
		return nil, fmt.Errorf("admission check %q: %w", acName, err)
	}

	grp := wlGroup{
		local:           local,
		remoteWorkloads: make(map[string]*kueue.Workload, len(rClients)),
		remoteClients:   rClients,
		acName:          acName,
		jobAdapter:      adapter,
		controllerKey:   types.NamespacedName{Name: controllerName, Namespace: local.Namespace},
	}

	for clusterName, rClient := range rClients {
		wl := &kueue.Workload{}
		err := rClient.remoteClient.Get(ctx, client.ObjectKeyFromObject(local), wl)
		if client.IgnoreNotFound(err) != nil {
			return nil, err
		}
		if err != nil {
			wl = nil
		}
		grp.remoteWorkloads[clusterName] = wl
	}
	return &grp, nil
}

func (w *wlReconciler) remoteClientsForAC(ctx context.Context, acName kueue.AdmissionCheckReference) (map[string]*remoteClient, error) {
	cfg, err := w.helper.ConfigForAdmissionCheck(ctx, acName)
	if err != nil {
		return nil, err
	}
	clients := make(map[string]*remoteClient, len(cfg.Spec.Clusters))
	for _, clusterName := range cfg.Spec.Clusters {
		if client, found := w.clusters.controllerFor(clusterName); found {
			// Skip the localclient if its reconnect is ongoing.
			if !client.connecting.Load() {
				clients[clusterName] = client
			}
		}
	}
	if len(clients) == 0 {
		return nil, errNoActiveClusters
	}
	return clients, nil
}

// IsFinished 返回 true 表示本地 workload 已完成。
func (g *wlGroup) IsFinished() bool {
	return apimeta.IsStatusConditionTrue(g.local.Status.Conditions, kueue.WorkloadFinished) // ✅
}

func (g *wlGroup) RemoveRemoteObjects(ctx context.Context, cluster string) error {
	remWl := g.remoteWorkloads[cluster]
	if remWl == nil {
		return nil
	}
	if err := g.jobAdapter.DeleteRemoteObject(ctx, g.remoteClients[cluster].remoteClient, g.controllerKey); err != nil {
		return fmt.Errorf("deleting remote controller object: %w", err)
	}

	if controllerutil.RemoveFinalizer(remWl, kueue.ResourceInUseFinalizerName) {
		if err := g.remoteClients[cluster].remoteClient.Update(ctx, remWl); err != nil {
			return fmt.Errorf("removing remote workloads finalizer: %w", err)
		}
	}

	err := g.remoteClients[cluster].remoteClient.Delete(ctx, remWl)
	if client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("deleting remote workload: %w", err)
	}
	g.remoteWorkloads[cluster] = nil
	return nil
}

func (w *wlReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	// 子集群的workload 发生了变化
	log := ctrl.LoggerFrom(ctx)
	log.V(2).Info("Reconcile Workload")

	wl := &kueue.Workload{}
	isDeleted := false
	err := w.localclient.Get(ctx, req.NamespacedName, wl)
	switch {
	case client.IgnoreNotFound(err) != nil:
		return reconcile.Result{}, err
	case err != nil:
		oldWl, found := w.deletedWlCache.Get(req.String())
		if !found {
			return reconcile.Result{}, nil
		}
		wl = oldWl
		isDeleted = true

	default:
		isDeleted = !wl.DeletionTimestamp.IsZero()
	}

	mkAc, err := w.multikueueAC(ctx, wl)
	if err != nil {
		return reconcile.Result{}, err
	}

	if mkAc == nil || mkAc.State == kueue.CheckStateRejected {
		log.V(2).Info("Skip Workload", "isDeleted", isDeleted)
		if isDeleted {
			// Delete the workload from the cache considering the following case:
			//    The workload is not admitted by MultiKueue and so there are
			//    no workloads on worker clusters created, we can safely drop it
			//    from the cache.
			//    TODO(#3840): Ideally, we would not add it to the cache in the
			//    first place.
			w.deletedWlCache.Delete(req.String())
		}
		return reconcile.Result{}, nil
	}

	adapter, owner := w.adapter(wl) // workload 对应的 owner
	if adapter == nil {
		// Reject the workload since there is no chance for it to run.
		var rejectionMessage string
		if owner != nil {
			rejectionMessage = fmt.Sprintf("No multikueue adapter found for owner kind %q", schema.FromAPIVersionAndKind(owner.APIVersion, owner.Kind).String())
		} else {
			rejectionMessage = "No multikueue adapter found"
		}
		return reconcile.Result{}, w.updateACS(ctx, wl, mkAc, kueue.CheckStateRejected, rejectionMessage)
	}

	// 如果 workload 被删除，其 owner 也有可能被删除。在这种情况下，跳过调用 `IsJobManagedByKueue`，因为其输出不可靠。
	if !isDeleted {
		managed, unmanagedReason, err := adapter.IsJobManagedByKueue(ctx, w.localclient, types.NamespacedName{Name: owner.Name, Namespace: wl.Namespace})
		if err != nil {
			return reconcile.Result{}, err
		}

		if !managed {
			return reconcile.Result{}, w.updateACS(ctx, wl, mkAc, kueue.CheckStateRejected, fmt.Sprintf("The owner is not managed by Kueue: %s", unmanagedReason))
		}
	}

	grp, err := w.readGroup(ctx, wl, mkAc.Name, adapter, owner.Name) // 每个集群的 workload 的状态
	if err != nil {
		return reconcile.Result{}, err
	}

	if isDeleted {
		for cluster := range grp.remoteWorkloads {
			err := grp.RemoveRemoteObjects(ctx, cluster)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		w.deletedWlCache.Delete(req.String())
		return reconcile.Result{}, nil
	}

	return w.reconcileGroup(ctx, grp)
}

func (w *wlReconciler) reconcileGroup(ctx context.Context, group *wlGroup) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx).WithValues("op", "reconcileGroup")
	log.V(3).Info("Reconcile Workload Group")

	acs := workload.FindAdmissionCheck(group.local.Status.AdmissionChecks, group.acName)

	// 1. 当本地 workload 已完成或没有配额预留时，删除所有远程 workload
	if group.IsFinished() || !workload.HasQuotaReservation(group.local) {
		var errs []error
		for rem := range group.remoteWorkloads {
			if err := group.RemoveRemoteObjects(ctx, rem); err != nil { // ✅
				errs = append(errs, err)
				log.V(2).Error(err, "Deleting remote workload", "workerCluster", rem)
			}
		}
		return reconcile.Result{}, errors.Join(errs...)
	}

	if remoteFinishedCond, remote := group.RemoteFinishedCondition(); remoteFinishedCond != nil { // ✅
		// NOTE: we can have a race condition setting the wl status here and it being updated by the job controller
		// it should not be problematic but the "From remote xxxx:" could be lost ....

		if group.jobAdapter != nil {
			if err := group.jobAdapter.SyncJob(ctx, w.localclient, group.remoteClients[remote].remoteClient, group.controllerKey, group.local.Name, w.origin); err != nil {
				log.V(2).Error(err, "copying remote controller status", "workerCluster", remote)
				// we should retry this
				return reconcile.Result{}, err
			}
		} else {
			log.V(3).Info("Group with no adapter, skip owner status copy", "workerCluster", remote)
		}

		// copy the status to the local one
		wlPatch := workload.BaseSSAWorkload(group.local)
		apimeta.SetStatusCondition(&wlPatch.Status.Conditions, metav1.Condition{Type: kueue.WorkloadFinished, Status: metav1.ConditionTrue,
			Reason:  remoteFinishedCond.Reason,
			Message: remoteFinishedCond.Message,
		})
		return reconcile.Result{}, w.localclient.Status().Patch(ctx, wlPatch, client.Apply, client.FieldOwner(kueue.MultiKueueControllerName+"-finish"), client.ForceOwnership)
	}

	// 2. 删除所有不同步或不在选定 worker 的 workload
	for rem, remWl := range group.remoteWorkloads {
		if remWl != nil && !equality.Semantic.DeepEqual(group.local.Spec, remWl.Spec) {
			if err := client.IgnoreNotFound(group.RemoveRemoteObjects(ctx, rem)); err != nil {
				log.V(2).Error(err, "Deleting out of sync remote objects", "remote", rem)
				return reconcile.Result{}, err
			}
			group.remoteWorkloads[rem] = nil
		}
	}

	// 3. 获取第一个 reserving
	hasReserving, reservingRemote := group.FirstReserving()
	if hasReserving {
		// remove the non-reserving worker workloads
		for rem, remWl := range group.remoteWorkloads {
			if remWl != nil && rem != reservingRemote {
				if err := client.IgnoreNotFound(group.RemoveRemoteObjects(ctx, rem)); err != nil {
					log.V(2).Error(err, "Deleting out of sync remote objects", "remote", rem)
					return reconcile.Result{}, err
				}
				group.remoteWorkloads[rem] = nil
			}
		}

		acs := workload.FindAdmissionCheck(group.local.Status.AdmissionChecks, group.acName)
		if err := group.jobAdapter.SyncJob(ctx, w.localclient, group.remoteClients[reservingRemote].remoteClient, group.controllerKey, group.local.Name, w.origin); err != nil {
			log.V(2).Error(err, "creating remote controller object", "remote", reservingRemote)
			// We'll retry this in the next reconcile.
			return reconcile.Result{}, err
		}

		if acs.State != kueue.CheckStateRetry && acs.State != kueue.CheckStateRejected {
			if group.jobAdapter.KeepAdmissionCheckPending() {
				acs.State = kueue.CheckStatePending
			} else {
				acs.State = kueue.CheckStateReady
			}
			// update the message
			acs.Message = fmt.Sprintf("The workload got reservation on %q", reservingRemote)
			// update the transition time since is used to detect the lost worker state.
			acs.LastTransitionTime = metav1.NewTime(w.clock.Now())

			wlPatch := workload.BaseSSAWorkload(group.local)
			workload.SetAdmissionCheckState(&wlPatch.Status.AdmissionChecks, *acs, w.clock)
			err := w.localclient.Status().Patch(ctx, wlPatch, client.Apply, client.FieldOwner(kueue.MultiKueueControllerName), client.ForceOwnership)
			if err != nil {
				return reconcile.Result{}, err
			}
			w.recorder.Eventf(wlPatch, corev1.EventTypeNormal, "MultiKueue", acs.Message)
		}
		return reconcile.Result{RequeueAfter: w.workerLostTimeout}, nil
	} else if acs.State == kueue.CheckStateReady {
		// If there is no reserving and the AC is ready, the connection with the reserving remote might
		// be lost, keep the workload admitted for keepReadyTimeout and put it back in the queue after that.
		remainingWaitTime := w.workerLostTimeout - time.Since(acs.LastTransitionTime.Time)
		if remainingWaitTime > 0 {
			log.V(3).Info("Reserving remote lost, retry", "retryAfter", remainingWaitTime)
			return reconcile.Result{RequeueAfter: remainingWaitTime}, nil
		} else {
			acs.State = kueue.CheckStateRetry
			acs.Message = "Reserving remote lost"
			acs.LastTransitionTime = metav1.NewTime(w.clock.Now())
			wlPatch := workload.BaseSSAWorkload(group.local)
			workload.SetAdmissionCheckState(&wlPatch.Status.AdmissionChecks, *acs, w.clock)
			return reconcile.Result{}, w.localclient.Status().Patch(ctx, wlPatch, client.Apply, client.FieldOwner(kueue.MultiKueueControllerName), client.ForceOwnership)
		}
	}

	// finally - 创建缺失的 workload
	var errs []error
	for rem, remWl := range group.remoteWorkloads {
		if remWl == nil {
			clone := cloneForCreate(group.local, group.remoteClients[rem].origin)
			err := group.remoteClients[rem].remoteClient.Create(ctx, clone)
			if err != nil {
				// just log the error for a single remote
				log.V(2).Error(err, "creating remote object", "remote", rem)
				errs = append(errs, err)
			}
		}
	}
	return reconcile.Result{}, errors.Join(errs...)
}

func (g *wlGroup) RemoteFinishedCondition() (*metav1.Condition, string) {
	var bestMatch *metav1.Condition
	bestMatchRemote := ""
	for remote, wl := range g.remoteWorkloads {
		if wl == nil {
			continue
		}
		c := apimeta.FindStatusCondition(wl.Status.Conditions, kueue.WorkloadFinished)
		if c != nil && c.Status == metav1.ConditionTrue && (bestMatch == nil || c.LastTransitionTime.Before(&bestMatch.LastTransitionTime)) {
			bestMatch = c
			bestMatchRemote = remote
		}
	}
	return bestMatch, bestMatchRemote
}
