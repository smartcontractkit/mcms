package types

import "encoding/json"

// OperationMetadata contains metadata about an operation
type OperationMetadata struct {
	ContractType string   `json:"contractType"`
	Tags         []string `json:"tags"`
}

// Operation represents the data of an operation to be executed
type Operation struct {
	To               string          `json:"to"`
	Data             []byte          `json:"data"`
	AdditionalFields json.RawMessage `json:"additionalFields"`
	OperationMetadata
}

// ChainOperation represents an operation to be executed on a chain
type ChainOperation struct {
	ChainSelector ChainSelector `json:"chainSelector"`
	Operation
}
