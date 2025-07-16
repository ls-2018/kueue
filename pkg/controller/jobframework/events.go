package jobframework

// JobReconciler event reason list
const (
	ReasonStarted               = "Started"
	ReasonSuspended             = "Suspended"
	ReasonStopped               = "Stopped"
	ReasonDeleted               = "Deleted"
	ReasonCreatedWorkload       = "CreatedWorkload"
	ReasonDeletedWorkload       = "DeletedWorkload"
	ReasonUpdatedWorkload       = "UpdatedWorkload"
	ReasonFinishedWorkload      = "FinishedWorkload"
	ReasonErrWorkloadCompose    = "ErrWorkloadCompose"
	ReasonUpdatedAdmissionCheck = "UpdatedAdmissionCheck"
	ReasonJobNestingTooDeep     = "JobNestingTooDeep"
)
