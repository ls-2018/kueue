package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	genericapiserver "k8s.io/apiserver/pkg/server"

	visibilityv1beta1 "sigs.k8s.io/kueue/apis/visibility/v1beta1"
	"sigs.k8s.io/kueue/pkg/queue"
	apiv1beta1 "sigs.k8s.io/kueue/pkg/visibility/api/v1beta1"
)

var (
	Scheme         = runtime.NewScheme()
	Codecs         = serializer.NewCodecFactory(Scheme)
	ParameterCodec = runtime.NewParameterCodec(Scheme)
)

func init() {
	utilruntime.Must(visibilityv1beta1.AddToScheme(Scheme))
	utilruntime.Must(Scheme.SetVersionPriority(visibilityv1beta1.GroupVersion))
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})
}

// Install installs API scheme and registers storages
func Install(server *genericapiserver.GenericAPIServer, kueueMgr *queue.Manager) error {
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(visibilityv1beta1.GroupVersion.Group, Scheme, ParameterCodec, Codecs)
	apiGroupInfo.VersionedResourcesStorageMap[visibilityv1beta1.GroupVersion.Version] = apiv1beta1.NewStorage(kueueMgr)
	return server.InstallAPIGroups(&apiGroupInfo)
}
