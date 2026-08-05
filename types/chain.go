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
	// AdditionalMCMs holds metadata for extra MCM instances on the same chain, when the
	// chain is governed by more than one MCM contract (e.g. Canton's mcms-ccip and
	// mcms-ccv). Entries must have unique MCMAddress values distinct from the primary
	// MCMAddress, and must not themselves carry AdditionalMCMs. Requires proposal
	// version v2.
	AdditionalMCMs []ChainMetadata `json:"additionalMCMs,omitempty" validate:"omitempty,dive"`
}

// AllMCMs returns the primary MCM metadata followed by all AdditionalMCMs entries.
func (m ChainMetadata) AllMCMs() []ChainMetadata {
	all := make([]ChainMetadata, 0, len(m.AdditionalMCMs)+1)
	all = append(all, ChainMetadata{
		StartingOpCount:  m.StartingOpCount,
		MCMAddress:       m.MCMAddress,
		AdditionalFields: m.AdditionalFields,
	})
	return append(all, m.AdditionalMCMs...)
}

// GetMCM returns the metadata for the MCM instance with the given address. An empty
// address resolves to the primary MCM. The second return value reports whether the
// address matched the primary or an AdditionalMCMs entry.
func (m ChainMetadata) GetMCM(mcmAddress string) (ChainMetadata, bool) {
	if mcmAddress == "" || mcmAddress == m.MCMAddress {
		return ChainMetadata{
			StartingOpCount:  m.StartingOpCount,
			MCMAddress:       m.MCMAddress,
			AdditionalFields: m.AdditionalFields,
		}, true
	}
	for _, additional := range m.AdditionalMCMs {
		if additional.MCMAddress == mcmAddress {
			return additional, true
		}
	}
	return ChainMetadata{}, false
}

// ValidateMultiMCM checks the multi-MCM invariants: unique addresses across primary and
// AdditionalMCMs, and no nested AdditionalMCMs.
func (m ChainMetadata) ValidateMultiMCM() error {
	seen := map[string]struct{}{m.MCMAddress: {}}
	for _, additional := range m.AdditionalMCMs {
		if additional.MCMAddress == "" {
			return fmt.Errorf("additional MCM entries must have a non-empty MCMAddress")
		}
		if _, exists := seen[additional.MCMAddress]; exists {
			return fmt.Errorf("duplicate MCMAddress %q in chain metadata", additional.MCMAddress)
		}
		seen[additional.MCMAddress] = struct{}{}
		if len(additional.AdditionalMCMs) > 0 {
			return fmt.Errorf("additional MCM entries must not themselves have AdditionalMCMs")
		}
	}
	return nil
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

	mergedAdditionalMCMs, err := mergeAdditionalMCMs(m.AdditionalMCMs, other.AdditionalMCMs)
	if err != nil {
		return ChainMetadata{}, err
	}

	return ChainMetadata{
		StartingOpCount:  min(m.StartingOpCount, other.StartingOpCount),
		MCMAddress:       m.MCMAddress,
		AdditionalFields: mergedAdditionalFields,
		AdditionalMCMs:   mergedAdditionalMCMs,
	}, nil
}

// mergeAdditionalMCMs unions two AdditionalMCMs lists by MCMAddress. Entries with the
// same MCMAddress are merged with the usual per-instance rules.
func mergeAdditionalMCMs(a, b []ChainMetadata) ([]ChainMetadata, error) {
	if len(a) == 0 && len(b) == 0 {
		return nil, nil
	}

	merged := make([]ChainMetadata, 0, len(a)+len(b))
	indexByAddress := make(map[string]int, len(a)+len(b))
	for _, entry := range a {
		indexByAddress[entry.MCMAddress] = len(merged)
		merged = append(merged, entry)
	}
	for _, entry := range b {
		if idx, exists := indexByAddress[entry.MCMAddress]; exists {
			m, err := merged[idx].Merge(entry)
			if err != nil {
				return nil, err
			}
			m.AdditionalMCMs = nil // additional MCM entries never nest
			merged[idx] = m
			continue
		}
		indexByAddress[entry.MCMAddress] = len(merged)
		merged = append(merged, entry)
	}

	return merged, nil
}
