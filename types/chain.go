package types //nolint:revive,nolintlint // allow pkg name 'types'

import (
	"encoding/json"
	"fmt"
)

// ChainMetadata defines the metadata for a chain.
type ChainMetadata struct {
	StartingOpCount  uint64          `json:"startingOpCount"`
	MCMAddress       string          `json:"mcmAddress"`
	AdditionalFields json.RawMessage `json:"additionalFields,omitempty" validate:"omitempty"`
}

func (m *ChainMetadata) Merge(other ChainMetadata) (ChainMetadata, error) {
	if m.MCMAddress != other.MCMAddress {
		return ChainMetadata{}, fmt.Errorf("cannot merge ChainMetadata with different MCMAddress: %s vs %s",
			m.MCMAddress, other.MCMAddress)
	}

	var thisAdditionalFields map[string]any
	if len(m.AdditionalFields) > 0 {
		err := json.Unmarshal(m.AdditionalFields, &thisAdditionalFields)
		if err != nil {
			return ChainMetadata{}, fmt.Errorf("failed to unmarshal AdditionalFields of ChainMetadata (%v): %w",
				string(m.AdditionalFields), err)
		}
	}

	var otherAdditionalFields map[string]any
	if len(other.AdditionalFields) > 0 {
		err := json.Unmarshal(other.AdditionalFields, &otherAdditionalFields)
		if err != nil {
			return ChainMetadata{}, fmt.Errorf("failed to unmarshal AdditionalFields of ChainMetadata (%v): %w",
				string(other.AdditionalFields), err)
		}
	}

	for key, otherValue := range otherAdditionalFields {
		thisValue, exists := thisAdditionalFields[key]
		if !exists {
			thisAdditionalFields[key] = otherValue
		} else if thisValue != otherValue {
			return ChainMetadata{}, fmt.Errorf("cannot merge ChainMetadata with different value for key %q in AdditionalFields: %v vs %v",
				key, thisValue, otherValue)
		}
	}

	var mergedAdditionalFields json.RawMessage
	if len(thisAdditionalFields) > 0 {
		var err error
		mergedAdditionalFields, err = json.Marshal(thisAdditionalFields)
		if err != nil {
			return ChainMetadata{}, fmt.Errorf("failed to marshal merged AdditionalFields of ChainMetadata: %w", err)
		}
	}

	return ChainMetadata{
		StartingOpCount:  min(m.StartingOpCount, other.StartingOpCount),
		MCMAddress:       m.MCMAddress,
		AdditionalFields: mergedAdditionalFields,
	}, nil
}
