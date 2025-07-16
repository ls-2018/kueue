package over_webhooks

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
)

type CohortWebhook struct{}

func setupWebhookForCohort(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&kueue.Cohort{}).
		WithValidator(&CohortWebhook{}).
		Complete()
}

func (w *CohortWebhook) Default(context.Context, runtime.Object) error {
	return nil
}

//+kubebuilder:webhook:path=/validate-kueue-x-k8s-io-v1beta1-cohort,mutating=false,failurePolicy=fail,sideEffects=None,groups=kueue.x-k8s.io,resources=cohorts,verbs=create;update,versions=v1beta1,name=vcohort.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &CohortWebhook{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (w *CohortWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	cohort := obj.(*kueue.Cohort)
	log := ctrl.LoggerFrom(ctx).WithName("cohort-webhook")
	log.V(5).Info("Validating Cohort create")
	return nil, validateCohort(cohort).ToAggregate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (w *CohortWebhook) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	cohort := newObj.(*kueue.Cohort)
	log := ctrl.LoggerFrom(ctx).WithName("cohort-webhook")
	log.V(5).Info("Validating Cohort update")
	return nil, validateCohort(cohort).ToAggregate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (w *CohortWebhook) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateCohort(cohort *kueue.Cohort) field.ErrorList {
	path := field.NewPath("spec")
	config := validationConfig{
		hasParent:                        cohort.Spec.ParentName != "",
		enforceNominalGreaterThanLending: false,
	}
	var allErrs field.ErrorList
	// 资源不能小于0
	// 名不能冲
	allErrs = append(allErrs, validateFairSharing(cohort.Spec.FairSharing, path.Child("fairSharing"))...)
	allErrs = append(allErrs, validateResourceGroups(cohort.Spec.ResourceGroups, config, path.Child("resourceGroups"), true)...)
	return allErrs
}
