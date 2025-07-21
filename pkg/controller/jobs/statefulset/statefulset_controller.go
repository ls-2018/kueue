package statefulset

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/kueue/pkg/controller/jobframework"
)

var (
	gvk = appsv1.SchemeGroupVersion.WithKind("StatefulSet")
)

const (
	FrameworkName = "statefulset"
)

func init() {
	utilruntime.Must(jobframework.RegisterIntegration(FrameworkName, jobframework.IntegrationCallbacks{
		SetupIndexes:   SetupIndexes,
		NewReconciler:  NewReconciler,
		SetupWebhook:   SetupWebhook,
		JobType:        &appsv1.StatefulSet{},
		AddToScheme:    appsv1.AddToScheme,
		DependencyList: []string{"pod"},
		GVK:            gvk,
	}))
}

type StatefulSet appsv1.StatefulSet

func fromObject(o runtime.Object) *StatefulSet {
	return (*StatefulSet)(o.(*appsv1.StatefulSet))
}

func (d *StatefulSet) Object() client.Object {
	return (*appsv1.StatefulSet)(d)
}

func (d *StatefulSet) GVK() schema.GroupVersionKind {
	return gvk
}

func SetupIndexes(context.Context, client.FieldIndexer) error {
	return nil
}
