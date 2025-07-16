package multikueue

import (
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/kueue/pkg/controller/jobframework"
	"sigs.k8s.io/kueue/pkg/over_constants"
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
	adapters          map[string]jobframework.MultiKueueAdapter
}

type SetupOption func(o *SetupOptions)

// WithGCInterval - sets the interval between two garbage collection runs.
// If 0 the garbage collection is disabled.
func WithGCInterval(i time.Duration) SetupOption {
	return func(o *SetupOptions) {
		o.gcInterval = i
	}
}

// WithOrigin - sets the multikueue-origin label value used by this manager
func WithOrigin(origin string) SetupOption {
	return func(o *SetupOptions) {
		o.origin = origin
	}
}

// WithWorkerLostTimeout - sets the time for which the multikueue
// admission check is kept in Ready state after the connection to
// the admitting worker cluster is lost.
func WithWorkerLostTimeout(d time.Duration) SetupOption {
	return func(o *SetupOptions) {
		o.workerLostTimeout = d
	}
}

// WithEventsBatchPeriod - sets the delay used when adding remote triggered
// events to the workload's reconcile queue.
func WithEventsBatchPeriod(d time.Duration) SetupOption {
	return func(o *SetupOptions) {
		o.eventsBatchPeriod = d
	}
}

// WithAdapters sets or updates the adapters of the MultiKueue adapters.
func WithAdapters(adapters map[string]jobframework.MultiKueueAdapter) SetupOption {
	return func(o *SetupOptions) {
		o.adapters = adapters
	}
}

func SetupControllers(mgr ctrl.Manager, namespace string, opts ...SetupOption) error {
	options := &SetupOptions{
		gcInterval:        defaultGCInterval,
		origin:            defaultOrigin,
		workerLostTimeout: defaultWorkerLostTimeout,
		eventsBatchPeriod: over_constants.UpdatesBatchPeriod,
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

	wlRec := newWlReconciler(mgr.GetClient(), helper, cRec, options.origin, mgr.GetEventRecorderFor(over_constants.WorkloadControllerName), options.workerLostTimeout, options.eventsBatchPeriod, options.adapters)
	return wlRec.setupWithManager(mgr)
}
