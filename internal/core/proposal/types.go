package proposal

import (
	"github.com/smartcontractkit/mcms/types"
)

type ProposalType string

const (
	// MCMSProposalType is a proposal type for the MCMS contract.
	MCMS ProposalType = "MCMS"
	// MCMSWithTimelock is a proposal type for the MCMS contract with timelock.
	MCMSWithTimelock ProposalType = "MCMSWithTimelock"
)

var StringToProposalType = map[string]ProposalType{
	"MCMS":             MCMS,
	"MCMSWithTimelock": MCMSWithTimelock,
}

type Executable interface {
	SetRoot(chainSelector types.ChainSelector) (string, error)
	Execute(index int) (string, error)
}
