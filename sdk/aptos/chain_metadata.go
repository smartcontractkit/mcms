package aptos

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

// MCMSType distinguishes the on-chain MCMS contract variant.
// The zero value is MCMSTypeRegular, so existing proposals that omit this
// field are handled correctly.
type MCMSType string

const (
	MCMSTypeRegular MCMSType = ""
	MCMSTypeCurse   MCMSType = "curse"
)

func (m MCMSType) IsCurseMCMS() bool {
	return m == MCMSTypeCurse
}

type AdditionalFieldsMetadata struct {
	Role     TimelockRole `json:"role"`
	MCMSType MCMSType     `json:"mcmsType,omitempty"`
}
