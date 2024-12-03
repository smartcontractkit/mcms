package sdk

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/types"
)

// Simulator simulates mcms actions on a chain
//
// This is only required if the chain supports simulation.
type Simulator interface {
	SimulateSetRoot(
		metadata types.ChainMetadata,
		proof []common.Hash,
		root [32]byte,
		validUntil uint32,
		sortedSignatures []types.Signature,
	) error

	SimulateOperation(
		metadata types.ChainMetadata,
		operation types.Operation,
	) error
}
