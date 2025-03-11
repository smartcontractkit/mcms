package types

import "encoding/json"

// ChainMetadata defines the metadata for a chain.
type ChainMetadata struct {
	StartingOpCount  uint64          `json:"startingOpCount"`
	MCMAddress       string          `json:"mcmAddress"`
	AdditionalFields json.RawMessage `json:"additionalFields" validate:"omitempty"`
}
