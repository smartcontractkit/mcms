package sui

type TimelockRole uint8

func (t TimelockRole) String() string {
	switch t {
	case TimelockRoleBypasser:
		return "bypasser"
	case TimelockRoleProposer:
		return "proposer"
	case TimelockRoleCanceller:
		return "canceller"
	}

	return "unknown"
}

func (t TimelockRole) Byte() byte {
	return byte(t)
}

const (
	TimelockRoleBypasser TimelockRole = iota
	TimelockRoleCanceller
	TimelockRoleProposer
)

type AdditionalFieldsMetadata struct {
	Role TimelockRole `json:"role"`
}
