package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"

	visibility "sigs.k8s.io/kueue/apis/visibility/v1beta1"
)

// LqREST type is used only to install localqueues/ resource, so we can install localqueues/pending_workloads subresource.
// It implements the necessary interfaces for genericapiserver but does not provide any actual functionalities.
type LqREST struct{}

// Those interfaces are necessary for genericapiserver to work properly
var _ rest.Storage = &LqREST{}
var _ rest.Scoper = &LqREST{}
var _ rest.SingularNameProvider = &LqREST{}

func NewLqREST() *LqREST {
	return &LqREST{}
}

// New implements rest.Storage interface
func (m *LqREST) New() runtime.Object {
	return &visibility.PendingWorkloadsSummary{}
}

// Destroy implements rest.Storage interface
func (m *LqREST) Destroy() {}

// NamespaceScoped implements rest.Scoper interface
func (m *LqREST) NamespaceScoped() bool {
	return true
}

// GetSingularName implements rest.SingularNameProvider interface
func (m *LqREST) GetSingularName() string {
	return "localqueue"
}
