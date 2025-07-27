package resources

import (
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
)

type FlavorResource struct {
	Flavor   kueue.ResourceFlavorReference
	Resource corev1.ResourceName
}

func (fr FlavorResource) String() string {
	return fmt.Sprintf(`{"Flavor":"%s","Resource":"%s"}`, string(fr.Flavor), string(fr.Resource))
}

type FlavorResourceQuantities map[FlavorResource]int64

func (q FlavorResourceQuantities) MarshalJSON() ([]byte, error) {
	temp := make(map[string]int64, len(q))
	for flavourResource, num := range q {
		temp[flavourResource.String()] = num
	}
	return json.Marshal(temp)
}

func (frq FlavorResourceQuantities) FlattenFlavors() Requests {
	result := Requests{}
	for key, val := range frq {
		result[key.Resource] += val
	}
	return result
}
