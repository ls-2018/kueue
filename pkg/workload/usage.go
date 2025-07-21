package workload

import (
	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/resources"
)

type TASFlavorUsage []TopologyDomainRequests
type TASUsage map[kueue.ResourceFlavorReference]TASFlavorUsage

type Usage struct {
	Quota resources.FlavorResourceQuantities
	TAS   TASUsage
}
