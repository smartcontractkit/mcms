package sdk

// TimelockRole identifies one of the supported RBACTimelock roles.
type TimelockRole uint8

const (
	TimelockRoleAdmin TimelockRole = iota
	TimelockRoleBypasser
	TimelockRoleCanceller
	TimelockRoleExecutor
	TimelockRoleProposer
)

var timelockRoleNames = map[TimelockRole]string{
	TimelockRoleAdmin:     "Admin",
	TimelockRoleBypasser:  "Bypasser",
	TimelockRoleCanceller: "Canceller",
	TimelockRoleExecutor:  "Executor",
	TimelockRoleProposer:  "Proposer",
}

// Valid reports whether r is one of the supported timelock roles.
func (r TimelockRole) Valid() bool {
	_, ok := timelockRoleNames[r]
	return ok
}

func (r TimelockRole) String() string {
	if name, ok := timelockRoleNames[r]; ok {
		return name
	}

	return "Unknown"
}
