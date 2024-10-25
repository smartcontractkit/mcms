package mcms

import "github.com/ethereum/go-ethereum/common"

type Executor interface {
	Inspector
	Encoder
	// Returns a string of the transaction hash
	ExecuteOperation(nonce uint32, proof []common.Hash, operation ChainOperation) (string, error)
	// Returns a string of the transaction hash
	SetRoot(
		metadata ChainMetadata,
		proof []common.Hash,
		root [32]byte,
		validUntil uint32,
		sortedSignatures []Signature,
	) (string, error)
}
