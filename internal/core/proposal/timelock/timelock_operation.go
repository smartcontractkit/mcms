package timelock

type TimelockOperationType string

const (
	Schedule TimelockOperationType = "schedule"
	Cancel   TimelockOperationType = "cancel"
	Bypass   TimelockOperationType = "bypass"
)
