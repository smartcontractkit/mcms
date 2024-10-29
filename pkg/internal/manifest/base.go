package manifest

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
)

type BaseProposal struct {
	// Version specifies the format version of the proposal manifest, ensuring backward compatibility
	// across different parsers.
	Version string `json:"version"`

	// Kind defines the type of proposal manifest this represents.
	Kind string `json:"kind"`

	// Description is a human-readable (and typically generated) description intended to give
	// signers context for the proposed change.
	Description string `json:"description"`

	// Signatures is a list of signatures that have signed the proposal.
	Signatures []Signature `json:"signatures"`
}

type SpecConfig struct {
	// ValidUntil is a Unix timestamp that specifies the proposal's expiration.
	ValidUntil int64 `json:"validUntil"`

	// OverridePreviousRoot
	OverridePreviousRoot bool `json:"overridePreviousRoot"`
}

// MCMOperation is a struct that represents a single operation to be executed on a chain.
type Operation struct {
	// ChainSelector is the chain identifier that the operation is intended for.
	ChainSelector uint64 `json:"chainSelector"`

	// Transaction is the transaction to be executed on the chain.
	Transaction Transaction `json:"transaction"`
}

// BatchOperation is a struct that represents a batch of operations to be executed on a chain.
type BatchOperation struct {
	// ChainSelector is the chain identifier that the operation is intended for.
	ChainSelector uint64 `json:"chainSelector"`

	// Transactions is the list of transactions to be executed on the chain in a single operation.
	Transactions []Transaction `json:"transactions"`
}

// Chain is a struct that represents a chain-specific configuration for a blockchain.
type Chain struct {
	// ChainSelector is the chain identifier that the configuration is intended for.
	Selector uint64 `json:"selector"`

	// StartingOpCount is the starting operation count for the chain.
	StartingOpCount uint64 `json:"startingOpCount"`

	// MCMAddress is the address of the MCM contract on the chain.
	MCMAddress string `json:"mcmAddress"`

	// TimelockAddress is the address of the timelock contract on the chain. This is only used for
	// timelock proposals.
	TimelockAddress string `json:"timelockAddress,omitEmpty"`
}

// Transaction is a struct that represents a single transaction to be executed on a chain.
type Transaction struct {
	To               string          `json:"to"`
	Data             []byte          `json:"data"`
	AdditionalFields json.RawMessage `json:"additionalFields"`
	ContractType     string          `json:"contractType"`
	Tags             []string        `json:"tags"`
}

type Signature struct {
	R common.Hash `json:"r"`
	S common.Hash `json:"s"`
	V uint8       `json:"v"`
}
