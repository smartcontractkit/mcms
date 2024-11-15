package types

import "encoding/json"

// OperationMetadata contains metadata about an operation
type OperationMetadata struct {
	ContractType string   `json:"contractType"`
	Tags         []string `json:"tags"`
}

// Transaction contains the transaction data to be executed
type Transaction struct {
	OperationMetadata

	To               string          `json:"to"`
	Data             []byte          `json:"data"`
	AdditionalFields json.RawMessage `json:"additionalFields"`
}

// Operation represents an operation with a single transaction to be executed
type Operation struct {
	ChainSelector ChainSelector `json:"chainSelector"`
	Transaction   Transaction   `json:"transaction"`
}

// BatchOperation represents an operation with a batch of transactions to be executed.
type BatchOperation struct {
	ChainSelector ChainSelector `json:"chainSelector"`
	Transactions  []Transaction `json:"transactions" validate:"min=1"`
}
