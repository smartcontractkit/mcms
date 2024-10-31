package types

type TimelockAction string

const (
	TimelockActionSchedule TimelockAction = "schedule"
	TimelockActionCancel   TimelockAction = "cancel"
	TimelockActionBypass   TimelockAction = "bypass"
)
