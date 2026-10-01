package chainwrappers

import (
	"fmt"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

// BuildRoleMetadataDeriver constructs the chain-family-specific RoleMetadataDeriver for selector.
// chains may be nil; it is only consulted for runtime inputs (the Solana execute payer is the
// public key of chains.SolanaSigner(selector)).
func BuildRoleMetadataDeriver(_ ChainAccessor, selector types.ChainSelector) (sdk.RoleMetadataDeriver, error) {
	family, err := types.GetChainSelectorFamily(selector)
	if err != nil {
		return nil, fmt.Errorf("error getting chain family: %w", err)
	}

	switch family {
	case chainsel.FamilyEVM:
		return evm.NewRoleMetadataDeriver(), nil
	default:
		return nil, fmt.Errorf("unsupported chain family %s", family)
	}
}
