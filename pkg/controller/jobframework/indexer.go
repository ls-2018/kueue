package jobframework

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
)

func SetupWorkloadOwnerIndex(ctx context.Context, indexer client.FieldIndexer, gvk schema.GroupVersionKind) error {
	return indexer.IndexField(ctx, &kueue.Workload{}, GetOwnerKey(gvk), func(o client.Object) []string {
		// grab the Workload object, extract the owner...
		wl := o.(*kueue.Workload)
		if len(wl.OwnerReferences) == 0 {
			return nil
		}
		owners := make([]string, 0, len(wl.OwnerReferences))
		for i := range wl.OwnerReferences {
			owner := &wl.OwnerReferences[i]
			if owner.Kind == gvk.Kind && owner.APIVersion == gvk.GroupVersion().String() {
				owners = append(owners, owner.Name)
			}
		}
		return owners
	})
}
