package provisioning

const (
	ConfigKind                       = "ProvisioningRequestConfig"
	DeprecatedConsumesAnnotationKey  = "cluster-autoscaler.kubernetes.io/consume-provisioning-request"
	DeprecatedClassNameAnnotationKey = "cluster-autoscaler.kubernetes.io/provisioning-class-name"
	ConsumesAnnotationKey            = "autoscaling.x-k8s.io/consume-provisioning-request"
	ClassNameAnnotationKey           = "autoscaling.x-k8s.io/provisioning-class-name"

	CheckInactiveMessage = "the check is not active"
	NoRequestNeeded      = "the provisioning request is not needed"
)
