package evm

import "github.com/smartcontractkit/mcms/internal/core/timelock"

type EVMTimelockProposal struct {
	proposal timelock.TimelockProposal
}

func NewEVMTimelockProposal(proposal timelock.TimelockProposal) *EVMTimelockProposal {
	return &EVMTimelockProposal{
		proposal: proposal,
	}
}

// Ensures EVMTimelockProposal follows TimelockEncoder interface
var _ timelock.TimelockEncoder = (*EVMTimelockProposal)(nil)

func (p *EVMTimelockProposal) Encode() ([]byte, error) {
	// TODO: Implement the EVM specific encoding logic
	return nil, nil
}
