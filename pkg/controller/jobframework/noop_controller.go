package jobframework

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	_ JobReconcilerInterface = (*NoopReconciler)(nil)
)

type NoopReconciler struct {
	gvk schema.GroupVersionKind
}

func (r NoopReconciler) Reconcile(context.Context, reconcile.Request) (reconcile.Result, error) {
	return ctrl.Result{}, nil
}

func (r NoopReconciler) SetupWithManager(ctrl.Manager) error {
	ctrl.Log.V(3).Info("Skipped reconciler setup", "gvk", r.gvk)
	return nil
}

func NewNoopReconcilerFactory(gvk schema.GroupVersionKind) ReconcilerFactory {
	return func(client client.Client, record record.EventRecorder, opts ...Option) JobReconcilerInterface {
		return &NoopReconciler{gvk: gvk}
	}
}
