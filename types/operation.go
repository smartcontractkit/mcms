package types //nolint:revive,nolintlint // allow pkg name 'types'

import (
	"encoding/json"
)

// OperationMetadata contains metadata about an operation
type OperationMetadata struct {
	ContractType           string   `json:"contractType"`           // ContractType is the short type used in data store e.g. "Router". It represents the crosschain interface/expected behaviour of the destination contract.
	ContractTypeAndVersion string   `json:"contractTypeAndVersion"` // ContractTypeAndVersion is the .String() representation of the TypeAndVersion of our contract. It is used for decoding the message data (cell) according to the correct TL-B schema. For example, in TON we use a FullyQualifiedName as the contract type. The resulting string looks like "link.chain.ton.mcms.Timelock 1.0.0".
	Tags                   []string `json:"tags"`
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
	Transaction   Transaction   `json:"transaction" validate:"required"`
}

// BatchOperation represents an operation with a batch of transactions to be executed.
type BatchOperation struct {
	ChainSelector ChainSelector `json:"chainSelector" validate:"required"`
	Transactions  []Transaction `json:"transactions" validate:"required,min=1,dive"`
}
