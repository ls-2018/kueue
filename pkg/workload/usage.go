package workload

import (
	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/over_resources"
)

type TASFlavorUsage []TopologyDomainRequests
type TASUsage map[kueue.ResourceFlavorReference]TASFlavorUsage

type Usage struct {
	Quota over_resources.FlavorResourceQuantities
	TAS   TASUsage
}
