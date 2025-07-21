package jobframework

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metavalidation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	kueuealpha "sigs.k8s.io/kueue/apis/kueue/v1alpha1"
)

func ValidateTASPodSetRequest(replicaPath *field.Path, replicaMetadata *metav1.ObjectMeta) field.ErrorList {
	var allErrs field.ErrorList
	requiredValue, requiredFound := replicaMetadata.Annotations[kueuealpha.PodSetRequiredTopologyAnnotation]
	preferredValue, preferredFound := replicaMetadata.Annotations[kueuealpha.PodSetPreferredTopologyAnnotation]
	_, unconstrainedFound := replicaMetadata.Annotations[kueuealpha.PodSetUnconstrainedTopologyAnnotation]

	// validate no more than 1 annotation
	asInt := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}
	annotationFoundCount := asInt(requiredFound) + asInt(preferredFound) + asInt(unconstrainedFound)
	annotationsPath := replicaPath.Child("annotations")
	if annotationFoundCount > 1 {
		allErrs = append(allErrs, field.Invalid(annotationsPath, field.OmitValueType{},
			fmt.Sprintf("must not contain more than one topology annotation: [%q, %q, %q]",
				kueuealpha.PodSetRequiredTopologyAnnotation,
				kueuealpha.PodSetPreferredTopologyAnnotation,
				kueuealpha.PodSetUnconstrainedTopologyAnnotation),
		))
	}

	// validate labels
	if requiredFound {
		allErrs = append(allErrs, metavalidation.ValidateLabelName(requiredValue, annotationsPath.Key(kueuealpha.PodSetRequiredTopologyAnnotation))...)
	}
	if preferredFound {
		allErrs = append(allErrs, metavalidation.ValidateLabelName(preferredValue, annotationsPath.Key(kueuealpha.PodSetPreferredTopologyAnnotation))...)
	}
	return allErrs
}
