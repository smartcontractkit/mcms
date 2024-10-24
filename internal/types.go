package internal

import (
	"errors"

	"github.com/smartcontractkit/mcms/internal/core/proposal"
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

func LoadProposal(proposalType ProposalType, filePath string) (proposal.Proposal, error) {
	switch proposalType {
	case MCMS:
		return NewProposalFromFile(filePath)
	// case MCMSWithTimelock:
	// 	return timelock.NewMCMSWithTimelockProposalFromFile(filePath)
	default:
		return nil, errors.New("unknown proposal type")
	}
}
