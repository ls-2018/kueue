package tas

import "time"

const (
	TASTopologyController       = "tas-topology-controller"
	TASResourceFlavorController = "tas-resource-flavor-controller"
	TASTopologyUngater          = "tas-topology-ungater"
	TASNodeFailureController    = "tas-node-failure-controller"
)

const (
	NodeFailureDelay = 30 * time.Second
)
