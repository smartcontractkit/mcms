package types

import (
	"errors"
	"fmt"
	"slices"

	cselectors "github.com/smartcontractkit/chain-selectors"
)

// ChainSelector is a unique identifier for a chain.
//
// These values are defined in the chain-selectors dependency.
// https://github.com/smartcontractkit/chain-selectors
type ChainSelector uint64

var (
	// ErrChainFamilyNotFound is returned when the chain family is not found for a selector
	ErrChainFamilyNotFound = errors.New("chain family not found")

	// ErrUnsupportedChainFamily is returned when the chain family is not supported by MCMS
	ErrUnsupportedChainFamily = errors.New("unsupported chain family")
)

// supportedFamilies is a list of supported chain families that MCMS supports
var supportedFamilies = []string{
	cselectors.FamilyEVM,
	cselectors.FamilySolana,
	cselectors.FamilyAptos,
	cselectors.FamilySui,
}

// GetChainSelectorFamily returns the family of the chain selector.
func GetChainSelectorFamily(sel ChainSelector) (string, error) {
	f, err := cselectors.GetSelectorFamily(uint64(sel))
	if err != nil {
		return "", fmt.Errorf("%w for selector %d", ErrChainFamilyNotFound, sel)
	}

	if !slices.Contains(supportedFamilies, f) {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedChainFamily, f)
	}

	return f, nil
}
