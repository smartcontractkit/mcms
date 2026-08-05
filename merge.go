package mcms

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/smartcontractkit/mcms/types"
)

// Merge merges the given timelock proposal with the current one
func (m *TimelockProposal) Merge(ctx context.Context, other *TimelockProposal) (*TimelockProposal, error) {
	if m.Version != other.Version {
		return nil, errors.New("cannot merge proposals with different versions")
	}
	if m.Kind != other.Kind {
		return nil, errors.New("cannot merge proposals with different kinds")
	}
	if m.Action != other.Action {
		return nil, errors.New("cannot merge proposals with different actions")
	}

	if m.OverridePreviousRoot || other.OverridePreviousRoot {
		// FIXME: log warning when DX-1650 is done
		m.OverridePreviousRoot = true
	}

	if other.Description != "" {
		if m.Description != "" {
			m.Description += "\n"
		}
		m.Description += other.Description
	}

	m.Signatures = nil // reset signatures, as existing ones are no longer valid

	m.Metadata = mergeMetadata(m.Metadata, other.Metadata)

	for chainSelector, otherMetadata := range other.ChainMetadata {
		thisMetadata, exists := m.ChainMetadata[chainSelector]
		if !exists {
			m.ChainMetadata[chainSelector] = otherMetadata
			continue
		}

		mergedMetadata, err := mergeChainMetadata(thisMetadata, otherMetadata)
		if err != nil {
			return nil, fmt.Errorf("failed to merge metadata for chain %v: %w", chainSelector, err)
		}

		m.ChainMetadata[chainSelector] = mergedMetadata
	}

	m.ValidUntil = min(m.ValidUntil, other.ValidUntil)
	m.Delay = types.NewDuration(time.Duration(max(m.Delay.Nanoseconds(), other.Delay.Nanoseconds())))

	for chainSelector, otherTimelockAddress := range other.TimelockAddresses {
		currentAddress, exists := m.TimelockAddresses[chainSelector]
		if exists {
			if currentAddress != otherTimelockAddress {
				return nil, fmt.Errorf("cannot merge proposals with different timelock addresses (chain %v): %q vs %q",
					chainSelector, currentAddress, otherTimelockAddress)
			}
		} else {
			m.TimelockAddresses[chainSelector] = otherTimelockAddress
		}
	}

	if other.SaltOverride != nil {
		if m.SaltOverride == nil {
			m.SaltOverride = other.SaltOverride
		} else {
			for i := range m.SaltOverride {
				m.SaltOverride[i] ^= other.SaltOverride[i]
			}
		}
	}

	m.Operations = append(m.Operations, other.Operations...)

	return m, nil
}

// mergeChainMetadata merges two ChainMetadata entries for the same chain selector,
// supporting disjoint MCM instance sets: if the primary MCM addresses differ, each
// primary must be present in the other entry's instance set (primary + AdditionalMCMs),
// and the instances are unioned. The merged primary is chosen deterministically (the
// lexicographically smallest MCM address).
func mergeChainMetadata(this, other types.ChainMetadata) (types.ChainMetadata, error) {
	if this.MCMAddress == other.MCMAddress {
		return this.Merge(other)
	}

	// Build instance maps keyed by MCM address.
	thisInstances := make(map[string]types.ChainMetadata, len(this.AdditionalMCMs)+1)
	for _, instance := range this.AllMCMs() {
		thisInstances[instance.MCMAddress] = instance
	}
	otherInstances := make(map[string]types.ChainMetadata, len(other.AdditionalMCMs)+1)
	for _, instance := range other.AllMCMs() {
		otherInstances[instance.MCMAddress] = instance
	}

	// Each primary must be known to the other side; otherwise the instance sets are
	// truly disjoint and we cannot decide on a safe merge.
	if _, ok := otherInstances[this.MCMAddress]; !ok {
		return types.ChainMetadata{}, fmt.Errorf(
			"cannot merge ChainMetadata with different MCMAddress: %s vs %s",
			this.MCMAddress, other.MCMAddress)
	}
	if _, ok := thisInstances[other.MCMAddress]; !ok {
		return types.ChainMetadata{}, fmt.Errorf(
			"cannot merge ChainMetadata with different MCMAddress: %s vs %s",
			this.MCMAddress, other.MCMAddress)
	}

	// Union instances by address, merging duplicates with per-instance rules.
	addresses := make([]string, 0, len(thisInstances)+len(otherInstances))
	merged := make(map[string]types.ChainMetadata, len(thisInstances)+len(otherInstances))
	for addr, instance := range thisInstances {
		merged[addr] = instance
		addresses = append(addresses, addr)
	}
	for addr, instance := range otherInstances {
		if existing, ok := merged[addr]; ok {
			m, err := existing.Merge(instance)
			if err != nil {
				return types.ChainMetadata{}, err
			}
			m.AdditionalMCMs = nil // instances never nest
			merged[addr] = m
			continue
		}
		merged[addr] = instance
		addresses = append(addresses, addr)
	}

	slices.Sort(addresses)

	primary := merged[addresses[0]]
	result := types.ChainMetadata{
		StartingOpCount:  primary.StartingOpCount,
		MCMAddress:       primary.MCMAddress,
		AdditionalFields: primary.AdditionalFields,
	}
	for _, addr := range addresses[1:] {
		result.AdditionalMCMs = append(result.AdditionalMCMs, merged[addr])
	}

	return result, nil
}

func mergeMetadata(m1, m2 map[string]any) map[string]any {
	if len(m2) == 0 {
		return m1
	}

	return mergeMetadataMaps(m1, m2)
}

func mergeMetadataMaps(a, b map[string]any) map[string]any {
	out := make(map[string]any, len(a))
	maps.Copy(out, a)
	for k, bv := range b {
		av, ok := out[k]
		if !ok {
			out[k] = bv
			continue
		}

		// handle matching elements of type slices
		sliceAV, aIsSlice := av.([]any)
		sliceBV, bIsSlice := bv.([]any)
		if aIsSlice && bIsSlice {
			out[k] = append(sliceAV, sliceBV...)
			continue
		}

		// handle matching elements of type map
		mapAV, aIsMap := av.(map[string]any)
		mapBV, bIsMap := bv.(map[string]any)
		if aIsMap && bIsMap {
			out[k] = mergeMetadataMaps(mapAV, mapBV)
			continue
		}

		out[k] = bv // just (over)write with the value from b
	}

	return out
}
