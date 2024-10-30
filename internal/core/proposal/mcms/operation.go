package mcms

import (
	"encoding/json"
)

type ChainSelector uint64

type OperationMetadata struct {
	ContractType string   `json:"contractType"`
	Tags         []string `json:"tags"`
}
type Operation struct {
	To               string          `json:"to"`
	Data             []byte          `json:"data"`
	AdditionalFields json.RawMessage `json:"additionalFields"`
	OperationMetadata
}
type ChainOperation struct {
	ChainSelector ChainSelector `json:"chainSelector"`
	Operation
}

// Validator interface used to validate the fields of a chain operation across different chains.
type Validator interface {
	Validate() error
}
