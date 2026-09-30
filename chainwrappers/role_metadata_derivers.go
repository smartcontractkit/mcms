package chainwrappers

import (
	"fmt"

	solanago "github.com/gagliardetto/solana-go"
	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/sdk/evm"
	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

// BuildRoleMetadataDeriver constructs the chain-family-specific RoleMetadataDeriver for selector.
// chains may be nil; it is only consulted for runtime inputs (the Solana execute payer is the
// public key of chains.SolanaSigner(selector)).
func BuildRoleMetadataDeriver(chains ChainAccessor, selector types.ChainSelector) (sdk.RoleMetadataDeriver, error) {
	family, err := types.GetChainSelectorFamily(selector)
	if err != nil {
		return nil, fmt.Errorf("error getting chain family: %w", err)
	}

	switch family {
	case chainsel.FamilyEVM:
		return evm.NewRoleMetadataDeriver(), nil
	case chainsel.FamilySolana:
		return solanasdk.NewRoleMetadataDeriver(solanaExecutePayer(chains, selector)), nil
	case chainsel.FamilyAptos:
		return aptos.NewRoleMetadataDeriver(), nil
	default:
		return nil, fmt.Errorf("unsupported chain family %s", family)
	}
}

func solanaExecutePayer(chains ChainAccessor, selector types.ChainSelector) *solanago.PublicKey {
	if chains == nil {
		return nil
	}
	signer, ok := chains.SolanaSigner(uint64(selector))
	if !ok || signer == nil {
		return nil
	}
	payer := signer.PublicKey()

	return &payer
}
