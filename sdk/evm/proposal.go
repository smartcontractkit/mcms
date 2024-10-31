package evm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/mcms/internal/core/timelock"
	"github.com/smartcontractkit/mcms/types"
)

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

// Q: When is predecessor retrieved?
func (p *EVMTimelockProposal) Encode() ([]types.Operation, error) {
	// TODO: Implement the EVM specific encoding logic

	ops := make([]types.Operation, 0, len(p.proposal.Batches))
	for _, batch := range p.proposal.Batches {
		ops = append(ops, types.Operation{
			To: common.HexToAddress(batch.TimelockAddress),
			// encode timlock data using gethwrappers, and depending on the operation type (shcedule, cancel, bypass)
			Data:         []byte{},
			Value:        big.NewInt(0),
			ContractType: "timelock",
		})
	}

	return ops, nil
}
