package multikueue

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/controller/jobframework"
)

const (
	eventChBufferSize = 10

	// 这个集合将提供 0 到 5m20s 之间的等待时间
	retryIncrement = 5 * time.Second
	retryMaxSteps  = 7
)

// retryAfter 返回一个指数递增的间隔，范围在 0 到 2^(retryMaxSteps-1) * retryIncrement 之间
func retryAfter(failedAttempts uint) time.Duration {
	if failedAttempts == 0 {
		return 0
	}
	return (1 << (min(failedAttempts, retryMaxSteps) - 1)) * retryIncrement
}

type clientWithWatchBuilder func(config []byte, options client.Options) (client.WithWatch, error)

type remoteClient struct {
	clusterName  string
	localClient  client.Client
	remoteClient client.WithWatch
	wlUpdateCh   chan<- event.GenericEvent
	watchEndedCh chan<- event.GenericEvent
	watchCancel  func()
	kubeconfig   []byte
	origin       string
	adapters     map[string]jobframework.MultiKueueAdapter

	connecting         atomic.Bool
	failedConnAttempts uint

	// 仅用于单元测试。单元测试中无需创建完全功能的远程客户端，且创建有效的 kubeconfig 内容并不简单。
	// 完整的客户端创建和使用已在集成和 e2e 测试中验证。
	builderOverride clientWithWatchBuilder
}

type workloadKueueWatcher struct{}

var _ jobframework.MultiKueueWatcher = (*workloadKueueWatcher)(nil)

func (*workloadKueueWatcher) GetEmptyList() client.ObjectList {
	return &kueue.WorkloadList{}
}

func (*workloadKueueWatcher) WorkloadKeyFor(o runtime.Object) (types.NamespacedName, error) {
	wl, isWl := o.(*kueue.Workload)
	if !isWl {
		return types.NamespacedName{}, errors.New("not a workload")
	}
	return client.ObjectKeyFromObject(wl), nil
}

func (rc *remoteClient) startWatcher(ctx context.Context, kind string, w jobframework.MultiKueueWatcher) error {
	log := ctrl.LoggerFrom(ctx).WithValues("watchKind", kind)
	newWatcher, err := rc.remoteClient.Watch(ctx, w.GetEmptyList(), client.MatchingLabels{kueue.MultiKueueOriginLabel: rc.origin}) // ✅
	if err != nil {
		return err
	}

	go func() {
		log.V(2).Info("Starting watch")
		for r := range newWatcher.ResultChan() {
			switch r.Type {
			case watch.Error:
				switch s := r.Object.(type) {
				case *metav1.Status:
					log.V(3).Info("Watch error", "status", s.Status, "message", s.Message, "reason", s.Reason)
				default:
					log.V(3).Info("Watch error with unexpected type", "type", fmt.Sprintf("%T", s))
				}
			default:
				wlKey, err := w.WorkloadKeyFor(r.Object)
				if err != nil {
					log.Error(err, "Cannot get workload key", "jobKind", r.Object.GetObjectKind().GroupVersionKind())
				} else {
					rc.queueWorkloadEvent(ctx, wlKey)
				}
			}
		}
		log.V(2).Info("Watch ended", "ctxErr", ctx.Err())
		// If the context is not yet Done , queue a reconcile to attempt reconnection
		if ctx.Err() == nil {
			oldConnecting := rc.connecting.Swap(true)
			// reconnect if this is the first watch failing.
			if !oldConnecting {
				log.V(2).Info("Queue reconcile for reconnect", "cluster", rc.clusterName)
				rc.queueWatchEndedEvent(ctx) // ✅
			}
		}
	}()
	return nil
}

func (rc *remoteClient) StopWatchers() {
	if rc.watchCancel != nil {
		rc.watchCancel()
	}
}

// runGC - 列出所有具有相同 multikueue-origin 的远程 workload，并移除那些不再有本地对应项（缺失或等待删除）的 workload。如果远程 workload 被 job 拥有，也会删除该 job。
func (rc *remoteClient) runGC(ctx context.Context) {
	log := ctrl.LoggerFrom(ctx)

	if rc.connecting.Load() {
		log.V(5).Info("Skip disconnected localclient")
		return
	}

	lst := &kueue.WorkloadList{}
	err := rc.remoteClient.List(ctx, lst, client.MatchingLabels{kueue.MultiKueueOriginLabel: rc.origin}) // ✅
	if err != nil {
		log.Error(err, "Listing remote workloads")
		return
	}

	for _, remoteWl := range lst.Items {
		localWl := &kueue.Workload{}
		wlLog := log.WithValues("remoteWl", klog.KObj(&remoteWl))
		err := rc.localClient.Get(ctx, client.ObjectKeyFromObject(&remoteWl), localWl)
		if client.IgnoreNotFound(err) != nil {
			wlLog.Error(err, "Reading local workload")
			continue
		}

		if err == nil && localWl.DeletionTimestamp.IsZero() {
			// The local workload exists and isn't being deleted, so the remote workload is still relevant.
			continue
		}

		// if the remote wl has a controller(owning Job), delete the job
		if controller := metav1.GetControllerOf(&remoteWl); controller != nil {
			ownerKey := klog.KRef(remoteWl.Namespace, controller.Name)
			adapterKey := schema.FromAPIVersionAndKind(controller.APIVersion, controller.Kind).String()
			if adapter, found := rc.adapters[adapterKey]; !found {
				wlLog.V(2).Info("No adapter found", "adapterKey", adapterKey, "ownerKey", ownerKey)
			} else {
				wlLog.V(5).Info("MultiKueueGC deleting workload owner", "ownerKey", ownerKey, "ownnerKind", controller)
				err := adapter.DeleteRemoteObject(ctx, rc.remoteClient, types.NamespacedName{Name: controller.Name, Namespace: remoteWl.Namespace})
				if client.IgnoreNotFound(err) != nil {
					wlLog.Error(err, "Deleting remote workload's owner", "ownerKey", ownerKey)
				}
			}
		}
		wlLog.V(5).Info("MultiKueueGC deleting remote workload")
		if err := rc.remoteClient.Delete(ctx, &remoteWl); client.IgnoreNotFound(err) != nil {
			wlLog.Error(err, "Deleting remote workload")
		}
	}
}

// clustersReconciler 实现了所有 MultiKueueCluster 的 reconciler。
// 其主要任务是维护与每个 MultiKueueCluster 关联的远程客户端列表。
type clustersReconciler struct {
	localClient     client.Client
	configNamespace string

	lock sync.RWMutex
	// The list of remote remoteClients, indexed by the cluster name.
	remoteClients map[string]*remoteClient //  admissionCheck Name:
	wlUpdateCh    chan event.GenericEvent

	// gcInterval - 两次 GC 运行之间的等待时间。
	gcInterval time.Duration

	// multikueue-origin 使用的值
	origin string

	// rootContext - 保存 controller-runtime 在 Start 时传递的 context。
	// 用于为 MultiKueueClusters 客户端 watch 协程创建子 context，
	// 当 controller-manager 停止时可优雅结束。
	rootContext context.Context

	// 仅用于单元测试。单元测试中无需创建完全功能的远程客户端，且创建有效的 kubeconfig 内容并不简单。
	// 完整的客户端创建和使用已在集成和 e2e 测试中验证。
	builderOverride clientWithWatchBuilder

	// watchEndedCh - 一个事件通道，用于请求对 watch 循环已结束（连接丢失）的集群进行调和。
	watchEndedCh chan event.GenericEvent

	fsWatcher *KubeConfigFSWatcher

	adapters map[string]jobframework.MultiKueueAdapter
}

var _ manager.Runnable = (*clustersReconciler)(nil)
var _ reconcile.Reconciler = (*clustersReconciler)(nil)

func (c *clustersReconciler) stopAndRemoveCluster(clusterName string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if rc, found := c.remoteClients[clusterName]; found {
		rc.StopWatchers()
		delete(c.remoteClients, clusterName)
	}
}

func (c *clustersReconciler) controllerFor(acName string) (*remoteClient, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	rc, f := c.remoteClients[acName]
	return rc, f
}

func (c *clustersReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	cluster := &kueue.MultiKueueCluster{}

	err := c.localClient.Get(ctx, req.NamespacedName, cluster)
	if client.IgnoreNotFound(err) != nil {
		return reconcile.Result{}, err
	}

	log := ctrl.LoggerFrom(ctx)
	log.V(2).Info("Reconcile MultiKueueCluster")

	if err != nil || !cluster.DeletionTimestamp.IsZero() {
		c.stopAndRemoveCluster(req.Name)
		return reconcile.Result{}, nil //nolint:nilerr // nil is intentional, as either the cluster is deleted, or not found
	}

	// get the kubeconfig
	kubeConfig, retry, err := c.getKubeConfig(ctx, &cluster.Spec.KubeConfig)
	if retry {
		return reconcile.Result{}, err
	}
	if err != nil {
		log.Error(err, "reading kubeconfig")
		c.stopAndRemoveCluster(req.Name)
		return reconcile.Result{}, c.updateStatus(ctx, cluster, false, "BadConfig", err.Error())
	}

	if retryAfter, err := c.setRemoteClientConfig(ctx, cluster.Name, kubeConfig, c.origin); err != nil {
		log.Error(err, "setting kubeconfig", "retryAfter", retryAfter)
		if err := c.updateStatus(ctx, cluster, false, "ClientConnectionFailed", err.Error()); err != nil {
			return reconcile.Result{}, err
		} else {
			return reconcile.Result{RequeueAfter: ptr.Deref(retryAfter, 0)}, nil
		}
	}
	return reconcile.Result{}, c.updateStatus(ctx, cluster, true, "Active", "Connected")
}

func (c *clustersReconciler) getKubeConfigFromSecret(ctx context.Context, secretName string) ([]byte, bool, error) {
	sec := corev1.Secret{}
	secretObjKey := types.NamespacedName{
		Namespace: c.configNamespace,
		Name:      secretName,
	}
	err := c.localClient.Get(ctx, secretObjKey, &sec)
	if err != nil {
		return nil, !apierrors.IsNotFound(err), err
	}

	kconfigBytes, found := sec.Data[kueue.MultiKueueConfigSecretKey]
	if !found {
		return nil, false, fmt.Errorf("key %q not found in secret %q", kueue.MultiKueueConfigSecretKey, secretName)
	}

	return kconfigBytes, false, nil
}

type secretHandler struct {
	client client.Client
}

var _ handler.EventHandler = (*secretHandler)(nil)

func (s *secretHandler) Create(ctx context.Context, event event.CreateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	secret, isSecret := event.Object.(*corev1.Secret)
	if !isSecret {
		ctrl.LoggerFrom(ctx).V(5).Error(errors.New("not a secret"), "Failure on create event")
		return
	}
	if err := s.queue(ctx, secret, q); err != nil {
		ctrl.LoggerFrom(ctx).V(5).Error(err, "Failure on create event", "secret", klog.KObj(event.Object))
	}
}

func (s *secretHandler) Update(ctx context.Context, event event.UpdateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	secret, isSecret := event.ObjectNew.(*corev1.Secret)
	if !isSecret {
		ctrl.LoggerFrom(ctx).V(5).Error(errors.New("not a secret"), "Failure on update event")
		return
	}
	if err := s.queue(ctx, secret, q); err != nil {
		ctrl.LoggerFrom(ctx).V(5).Error(err, "Failure on update event", "secret", klog.KObj(event.ObjectOld))
	}
}

func (s *secretHandler) Delete(ctx context.Context, event event.DeleteEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	secret, isSecret := event.Object.(*corev1.Secret)
	if !isSecret {
		ctrl.LoggerFrom(ctx).V(5).Error(errors.New("not a secret"), "Failure on delete event")
		return
	}
	if err := s.queue(ctx, secret, q); err != nil {
		ctrl.LoggerFrom(ctx).V(5).Error(err, "Failure on delete event", "secret", klog.KObj(event.Object))
	}
}

func (s *secretHandler) Generic(ctx context.Context, event event.GenericEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	secret, isSecret := event.Object.(*corev1.Secret)
	if !isSecret {
		ctrl.LoggerFrom(ctx).V(5).Error(errors.New("not a secret"), "Failure on generic event")
		return
	}
	if err := s.queue(ctx, secret, q); err != nil {
		ctrl.LoggerFrom(ctx).V(5).Error(err, "Failure on generic event", "secret", klog.KObj(event.Object))
	}
}

func (s *secretHandler) queue(ctx context.Context, secret *corev1.Secret, q workqueue.TypedRateLimitingInterface[reconcile.Request]) error {
	users := &kueue.MultiKueueClusterList{}
	if err := s.client.List(ctx, users, client.MatchingFields{UsingKubeConfigs: strings.Join([]string{secret.Namespace, secret.Name}, "/")}); err != nil {
		return err
	}

	for _, user := range users.Items {
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: user.Name,
			},
		}
		q.Add(req)
	}
	return nil
}

// +kubebuilder:rbac:groups="",resources=events,verbs=create;watch;update
// +kubebuilder:rbac:groups=kueue.x-k8s.io,resources=multikueueclusters,verbs=get;list;watch
// +kubebuilder:rbac:groups=kueue.x-k8s.io,resources=multikueueclusters/status,verbs=get;update;patch

func newClustersReconciler(c client.Client, namespace string, gcInterval time.Duration, origin string, fsWatcher *KubeConfigFSWatcher, adapters map[string]jobframework.MultiKueueAdapter) *clustersReconciler {
	return &clustersReconciler{
		localClient:     c,
		configNamespace: namespace,
		remoteClients:   make(map[string]*remoteClient),
		wlUpdateCh:      make(chan event.GenericEvent, eventChBufferSize),
		gcInterval:      gcInterval,
		origin:          origin,
		watchEndedCh:    make(chan event.GenericEvent, eventChBufferSize),
		fsWatcher:       fsWatcher,
		adapters:        adapters,
	}
}
func (c *clustersReconciler) Start(ctx context.Context) error {
	c.rootContext = ctx
	go c.runGC(ctx)
	return nil
}

func (c *clustersReconciler) runGC(ctx context.Context) {
	log := ctrl.LoggerFrom(ctx).WithName("MultiKueueGC")
	if c.gcInterval == 0 {
		log.V(2).Info("Garbage Collection is disabled")
		return
	}
	log.V(2).Info("Starting Garbage Collector")
	for {
		select {
		case <-ctx.Done():
			log.V(2).Info("Garbage Collector Stopped")
			return
		case <-time.After(c.gcInterval):
			log.V(4).Info("Run Garbage Collection for Lost Remote Workloads")
			for _, rc := range c.getRemoteClients() {
				rc.runGC(ctrl.LoggerInto(ctx, log.WithValues("multiKueueCluster", rc.clusterName)))
			}
		}
	}
}
func (c *clustersReconciler) getRemoteClients() []*remoteClient {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return slices.Collect(maps.Values(c.remoteClients))
}

func (c *clustersReconciler) setRemoteClientConfig(ctx context.Context, clusterName string, kubeconfig []byte, origin string) (*time.Duration, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	client, found := c.remoteClients[clusterName]
	if !found {
		client = newRemoteClient(c.localClient, c.wlUpdateCh, c.watchEndedCh, origin, clusterName, c.adapters)
		if c.builderOverride != nil {
			client.builderOverride = c.builderOverride
		}
		c.remoteClients[clusterName] = client
	}

	clientLog := ctrl.LoggerFrom(c.rootContext).WithValues("clusterName", clusterName)
	clientCtx := ctrl.LoggerInto(c.rootContext, clientLog)

	if retryAfter, err := client.setConfig(clientCtx, kubeconfig); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "failed to set kubeConfig in the remote localclient")
		return retryAfter, err
	}
	return nil, nil
}
func newRemoteClient(localClient client.Client, wlUpdateCh, watchEndedCh chan<- event.GenericEvent, origin, clusterName string, adapters map[string]jobframework.MultiKueueAdapter) *remoteClient {
	rc := &remoteClient{
		clusterName:  clusterName,
		wlUpdateCh:   wlUpdateCh,
		watchEndedCh: watchEndedCh,
		localClient:  localClient,
		origin:       origin,
		adapters:     adapters,
	}
	rc.connecting.Store(true)
	return rc
}

// setConfig - 如果新配置与当前使用的不同，或请求了重连，将尝试重新创建 k8s 客户端并重启 watch。
// 如果遇到的错误不是永久性的，则返回重试的时间间隔。
func (rc *remoteClient) setConfig(watchCtx context.Context, kubeconfig []byte) (*time.Duration, error) {
	configChanged := !equality.Semantic.DeepEqual(kubeconfig, rc.kubeconfig)
	if !configChanged && !rc.connecting.Load() {
		return nil, nil
	}

	rc.StopWatchers()
	if configChanged {
		rc.kubeconfig = kubeconfig
		rc.failedConnAttempts = 0
	}

	builder := newClientWithWatch
	if rc.builderOverride != nil {
		builder = rc.builderOverride
	}
	remoteClient, err := builder(kubeconfig, client.Options{Scheme: rc.localClient.Scheme()})
	if err != nil {
		return nil, err
	}

	rc.remoteClient = remoteClient

	watchCtx, rc.watchCancel = context.WithCancel(watchCtx)
	err = rc.startWatcher(watchCtx, kueue.GroupVersion.WithKind("Workload").GroupKind().String(), &workloadKueueWatcher{})
	if err != nil {
		rc.failedConnAttempts++
		return ptr.To(retryAfter(rc.failedConnAttempts)), err
	}

	// add a watch for all the adapters implementing multiKueueWatcher
	for kind, adapter := range rc.adapters {
		watcher, implementsWatcher := adapter.(jobframework.MultiKueueWatcher)
		if !implementsWatcher {
			continue
		}
		err := rc.startWatcher(watchCtx, kind, watcher)
		if err != nil {
			// not being able to setup a watcher is not ideal but we can function with only the wl watcher.
			ctrl.LoggerFrom(watchCtx).Error(err, "Unable to start the watcher", "kind", kind)
			// however let's not accept this for now.
			rc.failedConnAttempts++
			return ptr.To(retryAfter(rc.failedConnAttempts)), err
		}
	}

	rc.connecting.Store(false)
	rc.failedConnAttempts = 0
	return nil, nil
}

func newClientWithWatch(kubeconfig []byte, options client.Options) (client.WithWatch, error) {
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	return client.NewWithWatch(restConfig, options)
}
func (rc *remoteClient) queueWatchEndedEvent(ctx context.Context) {
	cluster := &kueue.MultiKueueCluster{}
	if err := rc.localClient.Get(ctx, types.NamespacedName{Name: rc.clusterName}, cluster); err == nil {
		rc.watchEndedCh <- event.GenericEvent{Object: cluster}
	} else {
		ctrl.LoggerFrom(ctx).Error(err, "sending watch ended event")
	}
}

func (rc *remoteClient) queueWorkloadEvent(ctx context.Context, wlKey types.NamespacedName) {
	localWl := &kueue.Workload{}
	if err := rc.localClient.Get(ctx, wlKey, localWl); err == nil {
		rc.wlUpdateCh <- event.GenericEvent{Object: localWl}
	} else if !apierrors.IsNotFound(err) {
		ctrl.LoggerFrom(ctx).Error(err, "reading local workload")
	}
}

func (c *clustersReconciler) setupWithManager(mgr ctrl.Manager) error {
	err := mgr.Add(c)
	if err != nil {
		return err
	}

	syncHndl := handler.Funcs{
		GenericFunc: func(_ context.Context, e event.GenericEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
				Name: e.Object.GetName(),
			}})
		},
	}

	fsWatcherHndl := handler.Funcs{
		GenericFunc: func(_ context.Context, e event.GenericEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			// batch the events
			q.AddAfter(reconcile.Request{NamespacedName: types.NamespacedName{
				Name: e.Object.GetName(),
			}}, 100*time.Millisecond)
		},
	}

	filterLog := mgr.GetLogger().WithName("MultiKueueCluster filter")
	filter := predicate.Funcs{
		CreateFunc: func(ce event.CreateEvent) bool {
			if cluster, isCluster := ce.Object.(*kueue.MultiKueueCluster); isCluster {
				if cluster.Spec.KubeConfig.LocationType == kueue.PathLocationType {
					err := c.fsWatcher.AddOrUpdate(cluster.Name, cluster.Spec.KubeConfig.Location)
					if err != nil {
						filterLog.Error(err, "AddOrUpdate FS watch", "cluster", klog.KObj(cluster))
					}
				}
			}
			return true
		},
		UpdateFunc: func(ue event.UpdateEvent) bool {
			clusterNew, isClusterNew := ue.ObjectNew.(*kueue.MultiKueueCluster)
			clusterOld, isClusterOld := ue.ObjectOld.(*kueue.MultiKueueCluster)
			if !isClusterNew || !isClusterOld {
				return true
			}

			if clusterNew.Spec.KubeConfig.LocationType == kueue.SecretLocationType && clusterOld.Spec.KubeConfig.LocationType == kueue.PathLocationType {
				err := c.fsWatcher.Remove(clusterOld.Name)
				if err != nil {
					filterLog.Error(err, "Remove FS watch", "cluster", klog.KObj(clusterOld))
				}
			}

			if clusterNew.Spec.KubeConfig.LocationType == kueue.PathLocationType {
				err := c.fsWatcher.AddOrUpdate(clusterNew.Name, clusterNew.Spec.KubeConfig.Location)
				if err != nil {
					filterLog.Error(err, "AddOrUpdate FS watch", "cluster", klog.KObj(clusterNew))
				}
			}
			return true
		},
		DeleteFunc: func(de event.DeleteEvent) bool {
			if cluster, isCluster := de.Object.(*kueue.MultiKueueCluster); isCluster {
				if cluster.Spec.KubeConfig.LocationType == kueue.PathLocationType {
					err := c.fsWatcher.Remove(cluster.Name)
					if err != nil {
						filterLog.Error(err, "Remove FS watch", "cluster", klog.KObj(cluster))
					}
				}
			}
			return true
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&kueue.MultiKueueCluster{}).
		Watches(&corev1.Secret{}, &secretHandler{client: c.localClient}).
		WatchesRawSource(source.Channel(c.watchEndedCh, syncHndl)).
		WatchesRawSource(source.Channel(c.fsWatcher.reconcile, fsWatcherHndl)).
		WithEventFilter(filter).
		Complete(c)
}

func (c *clustersReconciler) updateStatus(ctx context.Context, cluster *kueue.MultiKueueCluster, active bool, reason, message string) error {
	newCondition := metav1.Condition{
		Type:               kueue.MultiKueueClusterActive,
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: cluster.Generation,
	}
	if active {
		newCondition.Status = metav1.ConditionTrue
	}

	// if the condition is up-to-date
	oldCondition := apimeta.FindStatusCondition(cluster.Status.Conditions, kueue.MultiKueueClusterActive)
	if cmpConditionState(oldCondition, &newCondition) {
		return nil
	}

	apimeta.SetStatusCondition(&cluster.Status.Conditions, newCondition)
	return c.localClient.Status().Update(ctx, cluster)
}

func (c *clustersReconciler) getKubeConfigFromPath(path string) ([]byte, bool, error) {
	content, err := os.ReadFile(path)
	return content, false, err
}
func (c *clustersReconciler) getKubeConfig(ctx context.Context, ref *kueue.KubeConfig) ([]byte, bool, error) {
	if ref.LocationType == kueue.SecretLocationType {
		return c.getKubeConfigFromSecret(ctx, ref.Location)
	}
	// Otherwise it's path
	return c.getKubeConfigFromPath(ref.Location)
}
