package types //nolint:revive,nolintlint // allow pkg name 'types'

import (
	"context"
	"errors"
	"fmt"
	"slices"

	chainsel "github.com/smartcontractkit/chain-selectors"
	chainselremote "github.com/smartcontractkit/chain-selectors/remote"
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
	chainsel.FamilyEVM,
	chainsel.FamilySolana,
	chainsel.FamilyAptos,
	chainsel.FamilySui,
	chainsel.FamilyTon,
}

// GetChainSelectorFamily returns the family of the chain selector.
func GetChainSelectorFamily(sel ChainSelector) (string, error) {
	// TODO: pass this ctx as a parameter.
	// this function is used in a lot of places, and passing the context through all of them would be a big breaking change
	// so a bigger refactor may be needed to properly pass context through all the layers that use this function
	ctx := context.Background()
	details, err := chainselremote.GetChainDetailsBySelector(ctx, uint64(sel))
	if err != nil {
		return "", fmt.Errorf("%w for selector %d", ErrChainFamilyNotFound, sel)
	}

	if !slices.Contains(supportedFamilies, details.Family) {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedChainFamily, details.Family)
	}

	return details.Family, nil
}
