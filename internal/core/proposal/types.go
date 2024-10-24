package proposal

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/mcms/internal/core/config"
	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
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
	Signable(sim bool) (Signable, error)
	AddSignature(signature mcms.Signature)
	Validate() error
}

type Signable interface {
	SigningHash() (common.Hash, error)
	GetCurrentOpCounts() (map[mcms.ChainSelector]uint64, error)
	GetConfigs() (map[mcms.ChainSelector]*config.Config, error)
	CheckQuorum(chain mcms.ChainSelector) (bool, error)
	ValidateSignatures() (bool, error)
	ValidateConfigs() error
}
