package proposal

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/internal/core/merkle"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

type ProposalType string

const (
	// MCMSProposalType is a proposal type for the MCMS contract.
	MCMS ProposalType = "MCMS"
	// MCMSWithTimelock is a proposal type for the MCMS contract with timelock.
	// MCMSWithTimelock ProposalType = "MCMSWithTimelock"
)

var StringToProposalType = map[string]ProposalType{
	"MCMS": MCMS,
	// "MCMSWithTimelock": MCMSWithTimelock,
}

type Proposal interface {
	Signable(sim bool, inspectors map[types.ChainSelector]sdk.Inspector) (Signable, error)
	AddSignature(signature types.Signature)
	Validate() error
}

type Signable interface {
	SigningHash() (common.Hash, error)
	GetCurrentOpCounts() (map[types.ChainSelector]uint64, error)
	GetConfigs() (map[types.ChainSelector]*types.Config, error)
	CheckQuorum(chain types.ChainSelector) (bool, error)
	ValidateSignatures() (bool, error)
	ValidateConfigs() error
	GetTree() *merkle.Tree
}

type Executable interface {
	SetRoot(chainSelector types.ChainSelector) (string, error)
	Execute(index int) (string, error)
}
