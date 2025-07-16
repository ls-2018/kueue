package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

func init() {
	localSchemeBuilder.Register(addDefaultingFuncs)
}

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

//nolint:revive // format required by generated code for defaulting
func SetDefaults_PendingWorkloadOptions(obj *PendingWorkloadOptions) {
	defaultPendingWorkloadsLimit := int64(1000)
	if obj.Limit == 0 {
		obj.Limit = defaultPendingWorkloadsLimit
	}
}
