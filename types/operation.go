package types //nolint:revive,nolintlint // allow pkg name 'types'

import (
	"encoding/json"

	"github.com/Masterminds/semver/v3"
)

// OperationMetadata contains metadata about an operation
type OperationMetadata struct {
	ContractType    string          `json:"contractType"`              // ContractType is the short type used in data store e.g. "Router".
	ContractVersion *semver.Version `json:"contractVersion,omitempty"` // ContractVersion is the version of the deployed contract e.g. "1.0.0".
	Tags            []string        `json:"tags"`
}

// Transaction contains the transaction data to be executed
type Transaction struct {
	OperationMetadata

	To               string          `json:"to" validate:"required"`
	Data             []byte          `json:"data" validate:"required"`
	AdditionalFields json.RawMessage `json:"additionalFields" validate:"required"`
}

// Operation represents an operation with a single transaction to be executed
type Operation struct {
	ChainSelector ChainSelector `json:"chainSelector" validate:"required"`
	// McmAddress optionally identifies which MCM instance on the chain governs this
	// operation. Empty means the chain's primary MCM (the MCMAddress on the chain's
	// ChainMetadata entry). Non-empty values must match the primary MCM or one of its
	// AdditionalMCMs entries. Requires proposal version v2.
	McmAddress  string      `json:"mcmAddress,omitempty"`
	Transaction Transaction `json:"transaction" validate:"required"`
}

// BatchOperation represents an operation with a batch of transactions to be executed.
type BatchOperation struct {
	ChainSelector ChainSelector `json:"chainSelector" validate:"required"`
	// McmAddress optionally identifies which MCM instance on the chain governs this
	// batch. See Operation.McmAddress.
	McmAddress   string        `json:"mcmAddress,omitempty"`
	Transactions []Transaction `json:"transactions" validate:"required,min=1,dive"`
}
