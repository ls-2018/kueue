package jobframework

import (
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kueuealpha "sigs.k8s.io/kueue/apis/kueue/v1alpha1"
	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
)

type podSetTopologyRequestBuilder struct {
	request *kueue.PodSetTopologyRequest
}

func (p *podSetTopologyRequestBuilder) Build() *kueue.PodSetTopologyRequest {
	return p.request
}

func NewPodSetTopologyRequest(meta *metav1.ObjectMeta) *podSetTopologyRequestBuilder {
	psTopologyReq := &kueue.PodSetTopologyRequest{}
	requiredValue, requiredFound := meta.Annotations[kueuealpha.PodSetRequiredTopologyAnnotation]
	preferredValue, preferredFound := meta.Annotations[kueuealpha.PodSetPreferredTopologyAnnotation]
	unconstrained, unconstrainedFound := meta.Annotations[kueuealpha.PodSetUnconstrainedTopologyAnnotation]
	switch {
	case requiredFound:
		psTopologyReq.Required = &requiredValue
	case preferredFound:
		psTopologyReq.Preferred = &preferredValue
	case unconstrainedFound:
		unconstrained, _ := strconv.ParseBool(unconstrained)
		psTopologyReq.Unconstrained = &unconstrained
	default:
		psTopologyReq = nil
	}

	builder := &podSetTopologyRequestBuilder{request: psTopologyReq}
	return builder
}
func (p *podSetTopologyRequestBuilder) PodIndexLabel(podIndexLabel *string) *podSetTopologyRequestBuilder {
	if p.request != nil {
		p.request.PodIndexLabel = podIndexLabel
	}
	return p
}

func (p *podSetTopologyRequestBuilder) SubGroup(subGroupIndexLabel *string, subGroupCount *int32) *podSetTopologyRequestBuilder {
	if p.request != nil {
		p.request.SubGroupIndexLabel = subGroupIndexLabel
		p.request.SubGroupCount = subGroupCount
	}
	return p
}
