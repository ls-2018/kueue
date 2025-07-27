package v1beta1

const (
	ResourceInUseFinalizerName                 = "kueue.x-k8s.io/resource-in-use"
	DefaultPodSetName          PodSetReference = "main"
)

type StopPolicy string

const (
	None         StopPolicy = "None"
	HoldAndDrain StopPolicy = "HoldAndDrain"
	Hold         StopPolicy = "Hold"
)
