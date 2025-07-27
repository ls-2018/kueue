package multikueue

import (
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/kueue/pkg/constants"
	"sigs.k8s.io/kueue/pkg/controller/jobframework"
)

const (
	defaultGCInterval        = time.Minute
	defaultOrigin            = "multikueue"
	defaultWorkerLostTimeout = 5 * time.Minute
)

type SetupOptions struct {
	gcInterval        time.Duration
	origin            string
	workerLostTimeout time.Duration
	eventsBatchPeriod time.Duration
	adapters          map[string]jobframework.MultiKueueAdapter // gvk
}

type SetupOption func(o *SetupOptions)

// WithGCInterval - 设置两次垃圾回收运行之间的间隔。
// 如果为0，则禁用垃圾回收。
func WithGCInterval(i time.Duration) SetupOption {
	return func(o *SetupOptions) {
		o.gcInterval = i
	}
}

// WithOrigin - 设置此管理器使用的 multikueue-origin 标签值
func WithOrigin(origin string) SetupOption {
	return func(o *SetupOptions) {
		o.origin = origin
	}
}

// WithWorkerLostTimeout - 设置 multikueue
// admission check 在与接收 worker 集群的连接丢失后保持 Ready 状态的时间。
func WithWorkerLostTimeout(d time.Duration) SetupOption {
	return func(o *SetupOptions) {
		o.workerLostTimeout = d
	}
}

func SetupControllers(mgr ctrl.Manager, namespace string, opts ...SetupOption) error {
	options := &SetupOptions{
		gcInterval:        defaultGCInterval,
		origin:            defaultOrigin,
		workerLostTimeout: defaultWorkerLostTimeout,
		eventsBatchPeriod: constants.UpdatesBatchPeriod,
		adapters:          make(map[string]jobframework.MultiKueueAdapter),
	}

	for _, o := range opts {
		o(options)
	}

	helper, err := newMultiKueueStoreHelper(mgr.GetClient())
	if err != nil {
		return err
	}

	fsWatcher := newKubeConfigFSWatcher()
	err = mgr.Add(fsWatcher)
	if err != nil {
		return err
	}

	cRec := newClustersReconciler(mgr.GetClient(), namespace, options.gcInterval, options.origin, fsWatcher, options.adapters)
	err = cRec.setupWithManager(mgr)
	if err != nil {
		return err
	}

	acRec := newACReconciler(mgr.GetClient(), helper)
	err = acRec.setupWithManager(mgr)
	if err != nil {
		return err
	}

	wlRec := newWlReconciler(mgr.GetClient(), helper, cRec, options.origin,
		mgr.GetEventRecorderFor(constants.WorkloadControllerName),
		options.workerLostTimeout,
		options.eventsBatchPeriod,
		options.adapters,
	)
	return wlRec.setupWithManager(mgr)
}

// WithAdapters 设置或更新 MultiKueue 适配器。
func WithAdapters(adapters map[string]jobframework.MultiKueueAdapter) SetupOption {
	return func(o *SetupOptions) {
		o.adapters = adapters
	}
}
