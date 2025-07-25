// Package v1alpha1 contains API Schema definitions for the kueue v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=kueue.x-k8s.io
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: "kueue.x-k8s.io", Version: "v1alpha1"}

	// SchemeGroupVersion is alias to GroupVersion for client-go libraries.
	// It is required by pkg/client/informers/externalversions/...
	SchemeGroupVersion = GroupVersion

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// Resource is required by pkg/client/listers/...
func Resource(resource string) schema.GroupResource {
	return GroupVersion.WithResource(resource).GroupResource()
}
