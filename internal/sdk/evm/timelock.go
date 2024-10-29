package evm

import "github.com/smartcontractkit/mcms/internal/core/timelock"

type EVMTimelockProposal struct {
	config timelock.TimelockConfig
}

func NewEVMTimelockProposal(config timelock.TimelockConfig) *EVMTimelockProposal {
	return &EVMTimelockProposal{
		config: config,
	}
}

var _ timelock.TimelockProposal = (*EVMTimelockProposal)(nil)

func (p *EVMTimelockProposal) Encode() ([]byte, error) {
	// TODO: Implement the EVM specofic encoding logic
	return nil, nil
}
