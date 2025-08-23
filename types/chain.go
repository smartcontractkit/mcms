package types

import (
	"encoding/json"
	"errors"
)

// ChainMetadata defines the metadata for a chain.
type ChainMetadata struct {
	StartingOpCount  uint64          `json:"startingOpCount"`
	MCMAddress       string          `json:"mcmAddress"`
	AdditionalFields json.RawMessage `json:"additionalFields" validate:"omitempty"`
}

func (m ChainMetadata) Merge(other ChainMetadata) (ChainMetadata, error) {
	return ChainMetadata{}, errors.New("not implemented")
}
