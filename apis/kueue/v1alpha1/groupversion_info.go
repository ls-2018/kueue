// Package v1alpha1 包含 kueue v1alpha1 API 组的 API Schema 定义
// +kubebuilder:object:generate=true
// +groupName=kueue.x-k8s.io
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion 是用于注册这些对象的组版本。
	GroupVersion = schema.GroupVersion{Group: "kueue.x-k8s.io", Version: "v1alpha1"}

	// SchemeGroupVersion 是 client-go 库的 GroupVersion 别名。
	// 它被 pkg/client/informers/externalversions/... 所需。
	SchemeGroupVersion = GroupVersion

	// SchemeBuilder 用于将 go 类型添加到 GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme 将此组版本中的类型添加到给定的 scheme。
	AddToScheme = SchemeBuilder.AddToScheme
)

// Resource 被 pkg/client/listers/... 所需。
func Resource(resource string) schema.GroupResource {
	return GroupVersion.WithResource(resource).GroupResource()
}
