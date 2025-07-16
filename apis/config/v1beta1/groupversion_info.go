// Package v1beta1 包含 config v1beta1 API 组的 API Schema 定义
// +kubebuilder:object:generate=true
// +groupName=config.kueue.x-k8s.io
package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion 是用于注册这些对象的组版本
	GroupVersion = schema.GroupVersion{Group: "config.kueue.x-k8s.io", Version: "v1beta1"}

	// SchemeBuilder 用于将 go 类型添加到 GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// localSchemeBuilder 用于注册自动生成的 conversion 和 defaults 函数
	// 它被 ./zz_generated.conversion.go 和 ./zz_generated.defaults.go 所需
	localSchemeBuilder = &SchemeBuilder.SchemeBuilder

	// AddToScheme 将此组版本中的类型添加到给定的 scheme。
	AddToScheme = SchemeBuilder.AddToScheme
)

func init() {
	SchemeBuilder.Register(&Configuration{})
	// We only register manually written functions here. The registration of the
	// generated functions takes place in the generated files. The separation
	// makes the code compile even when the generated files are missing.
	localSchemeBuilder.Register(RegisterDefaults)
}
