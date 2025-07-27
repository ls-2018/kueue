package webhooks

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
)

func validateResourceName(name corev1.ResourceName, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	for _, msg := range validation.IsQualifiedName(string(name)) {
		allErrs = append(allErrs, field.Invalid(fldPath, name, msg))
	}
	return allErrs
}

type validationConfig struct {
	hasParent                        bool
	enforceNominalGreaterThanLending bool
}

// validateFairSharing validates the FairSharing config for both ClusterQueues and Cohorts.
func validateFairSharing(fs *kueue.FairSharing, fldPath *field.Path) field.ErrorList {
	if fs == nil {
		return nil
	}
	var allErrs field.ErrorList
	if fs.Weight != nil && fs.Weight.Cmp(resource.Quantity{}) < 0 {
		allErrs = append(allErrs, field.Invalid(fldPath, fs.Weight.String(), apimachineryvalidation.IsNegativeErrorMsg))
	}
	return allErrs
}
