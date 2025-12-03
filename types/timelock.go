package types

// TimelockAction is an enum for the different types of timelock actions that can be performed on
// a timelock proposal.
type TimelockAction string

const (
	// TimelockActionExecute sets up transactions to execute after a delay.
	TimelockActionSchedule TimelockAction = "schedule"

	// TimelockActionCancel cancels previously scheduled transactions.
	TimelockActionCancel TimelockAction = "cancel"

	// TimelockActionBypass directly executes transactions, skipping the timelock.
	TimelockActionBypass TimelockAction = "bypass"
)
