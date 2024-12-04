package sdk

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/types"
)

// Simulator simulates mcms actions on a chain
//
// This is only required if the chain supports simulation.
type Simulator interface {
	SimulateSetRoot(
		ctx context.Context,
		originCaller string,
		metadata types.ChainMetadata,
		proof []common.Hash,
		root [32]byte,
		validUntil uint32,
		sortedSignatures []types.Signature,
	) error

	SimulateOperation(
		ctx context.Context,
		metadata types.ChainMetadata,
		operation types.Operation,
	) error
}
