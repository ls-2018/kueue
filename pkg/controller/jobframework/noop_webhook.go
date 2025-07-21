package jobframework

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type noopWebhook struct {
}

func setupNoopWebhook(mgr ctrl.Manager, apiType runtime.Object) error {
	wh := &noopWebhook{}
	return ctrl.NewWebhookManagedBy(mgr).
		For(apiType).
		WithDefaulter(wh).
		WithValidator(wh).
		Complete()
}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (w *noopWebhook) Default(context.Context, runtime.Object) error {
	return nil
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (w *noopWebhook) ValidateCreate(context.Context, runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (w *noopWebhook) ValidateUpdate(context.Context, runtime.Object, runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (w *noopWebhook) ValidateDelete(context.Context, runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
