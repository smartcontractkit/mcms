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

// InstanceExecutor is an optional extension of Executor for chain families that support
// multiple MCM instances per chain selector (v2 proposals). The library type-asserts
// executors to this interface when setting the root on a non-primary instance, so that
// the root metadata's postOpCount is derived from the instance's own operation count
// (not the executor's chain-wide transaction count), matching the per-instance metadata
// leaf in the Merkle proof.
type InstanceExecutor interface {
	// SetRootForInstance behaves like SetRoot, but derives postOpCount as
	// metadata.StartingOpCount + instanceOpCount.
	SetRootForInstance(
		ctx context.Context,
		metadata types.ChainMetadata,
		instanceOpCount uint64,
		proof []common.Hash,
		root [32]byte,
		validUntil uint32,
		sortedSignatures []types.Signature,
	) (types.TransactionResult, error)
}
