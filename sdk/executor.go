package sdk

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/types"
)

// Executor is an interface for executing MCMS operations on a chain.
//
// This must be implemented by any chain.
type Executor interface {
	Inspector
	Encoder

	// ExecuteOperation Returns a string of the transaction hash
	ExecuteOperation(
		ctx context.Context,
		metadata types.ChainMetadata,
		nonce uint32,
		proof []common.Hash,
		op types.Operation,
	) (types.TransactionResult, error)

	// SetRoot Returns a string of the transaction hash
	SetRoot(
		ctx context.Context,
		metadata types.ChainMetadata,
		proof []common.Hash,
		root [32]byte,
		validUntil uint32,
		sortedSignatures []types.Signature,
	) (types.TransactionResult, error)
}
