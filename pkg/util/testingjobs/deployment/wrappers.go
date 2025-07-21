package deployment

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"

	controllerconstants "sigs.k8s.io/kueue/pkg/controller/over_constants"
	"sigs.k8s.io/kueue/pkg/over_constants"
)

// DeploymentWrapper wraps a Deployment.
type DeploymentWrapper struct {
	appsv1.Deployment
}

// MakeDeployment creates a wrapper for a Deployment with a single container.

// Obj returns the inner Deployment.
func (d *DeploymentWrapper) Obj() *appsv1.Deployment {
	return &d.Deployment
}

// Label sets the label of the Deployment
func (d *DeploymentWrapper) Label(k, v string) *DeploymentWrapper {
	if d.Labels == nil {
		d.Labels = make(map[string]string)
	}
	d.Labels[k] = v
	return d
}

// Queue updates the queue name of the Deployment
func (d *DeploymentWrapper) Queue(q string) *DeploymentWrapper {
	return d.Label(controllerconstants.QueueLabel, q)
}

// Name updated the name of the Deployment
func (d *DeploymentWrapper) Name(n string) *DeploymentWrapper {
	d.ObjectMeta.Name = n
	return d
}

// UID updates the uid of the Deployment.
func (d *DeploymentWrapper) UID(uid string) *DeploymentWrapper {
	d.ObjectMeta.UID = types.UID(uid)
	return d
}

// Image sets an image to the default container.
func (d *DeploymentWrapper) Image(image string, args []string) *DeploymentWrapper {
	d.Spec.Template.Spec.Containers[0].Image = image
	d.Spec.Template.Spec.Containers[0].Args = args
	return d
}

// Request adds a resource request to the default container.
func (d *DeploymentWrapper) Request(r corev1.ResourceName, v string) *DeploymentWrapper {
	if d.Spec.Template.Spec.Containers[0].Resources.Requests == nil {
		d.Spec.Template.Spec.Containers[0].Resources.Requests = corev1.ResourceList{}
	}
	d.Spec.Template.Spec.Containers[0].Resources.Requests[r] = resource.MustParse(v)
	return d
}

// Limit adds a resource limit to the default container.
func (d *DeploymentWrapper) Limit(r corev1.ResourceName, v string) *DeploymentWrapper {
	if d.Spec.Template.Spec.Containers[0].Resources.Limits == nil {
		d.Spec.Template.Spec.Containers[0].Resources.Limits = corev1.ResourceList{}
	}
	d.Spec.Template.Spec.Containers[0].Resources.Limits[r] = resource.MustParse(v)
	return d
}

// RequestAndLimit adds a resource request and limit to the default container.
func (d *DeploymentWrapper) RequestAndLimit(r corev1.ResourceName, v string) *DeploymentWrapper {
	return d.Request(r, v).Limit(r, v)
}

// Replicas updated the replicas of the Deployment
func (d *DeploymentWrapper) Replicas(replicas int32) *DeploymentWrapper {
	d.Spec.Replicas = &replicas
	return d
}

// ReadyReplicas updated the readyReplicas of the Deployment
func (d *DeploymentWrapper) ReadyReplicas(readyReplicas int32) *DeploymentWrapper {
	d.Status.ReadyReplicas = readyReplicas
	return d
}

// PodTemplateSpecLabel sets the label of the pod template spec of the Deployment
func (d *DeploymentWrapper) PodTemplateSpecLabel(k, v string) *DeploymentWrapper {
	if d.Spec.Template.Labels == nil {
		d.Spec.Template.Labels = make(map[string]string, 1)
	}
	d.Spec.Template.Labels[k] = v
	return d
}

// PodTemplateAnnotation sets the annotation of the pod template
func (d *DeploymentWrapper) PodTemplateAnnotation(k, v string) *DeploymentWrapper {
	if d.Spec.Template.Annotations == nil {
		d.Spec.Template.Annotations = make(map[string]string, 1)
	}
	d.Spec.Template.Annotations[k] = v
	return d
}

// PodTemplateSpecQueue updates the queue name of the pod template spec of the Deployment
func (d *DeploymentWrapper) PodTemplateSpecQueue(q string) *DeploymentWrapper {
	return d.PodTemplateSpecLabel(controllerconstants.QueueLabel, q)
}

func (d *DeploymentWrapper) PodTemplateSpecManagedByKueue() *DeploymentWrapper {
	return d.PodTemplateSpecLabel(over_constants.ManagedByKueueLabelKey, over_constants.ManagedByKueueLabelValue)
}

func (d *DeploymentWrapper) TerminationGracePeriod(seconds int64) *DeploymentWrapper {
	d.Spec.Template.Spec.TerminationGracePeriodSeconds = &seconds
	return d
}

func (d *DeploymentWrapper) SetTypeMeta() *DeploymentWrapper {
	d.APIVersion = appsv1.SchemeGroupVersion.String()
	d.Kind = "Deployment"
	return d
}
