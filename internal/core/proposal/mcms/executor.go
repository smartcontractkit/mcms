package mcms

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/types"
)

type Executor interface {
	Inspector
	Encoder
	// Returns a string of the transaction hash
	ExecuteOperation(metadata types.ChainMetadata, nonce uint32, proof []common.Hash, operation types.ChainOperation) (string, error)
	// Returns a string of the transaction hash
	SetRoot(
		metadata types.ChainMetadata,
		proof []common.Hash,
		root [32]byte,
		validUntil uint32,
		sortedSignatures []types.Signature,
	) (string, error)
}
