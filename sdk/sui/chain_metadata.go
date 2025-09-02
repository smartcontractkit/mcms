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

func (t TimelockRole) Byte() uint8 {
	return uint8(t)
}

const (
	TimelockRoleBypasser TimelockRole = iota
	TimelockRoleCanceller
	TimelockRoleProposer
)

type AdditionalFieldsMetadata struct {
	Role          TimelockRole `json:"role"`
	McmsPackageID string       `json:"mcms_package_id"`
}
