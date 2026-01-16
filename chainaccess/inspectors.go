package chainaccess

import (
	"fmt"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/solana"
	sdkSui "github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"
)

// BuildInspectors gets a map of inspectors for the given chain metadata and chain clients
func BuildInspectors(
	chains ChainAccess,
	chainMetadata map[types.ChainSelector]types.ChainMetadata,
	action types.TimelockAction) (map[types.ChainSelector]sdk.Inspector, error) {
	inspectors := map[types.ChainSelector]sdk.Inspector{}
	for chainSelector, metadata := range chainMetadata {
		inspector, err := BuildInspector(chains, chainSelector, action, metadata)
		if err != nil {
			return nil, err
		}
		inspectors[chainSelector] = inspector
	}

	return inspectors, nil
}

// BuildInspector constructs a chain-family-specific Inspector from ChainAccess plus metadata.
func BuildInspector(
	chains ChainAccess,
	selector types.ChainSelector,
	action types.TimelockAction,
	metadata types.ChainMetadata,
) (sdk.Inspector, error) {
	if chains == nil {
		return nil, fmt.Errorf("chain access is required")
	}

	family, err := types.GetChainSelectorFamily(selector)
	if err != nil {
		return nil, fmt.Errorf("error getting chain family: %w", err)
	}

	rawSelector := uint64(selector)
	switch family {
	case chainsel.FamilyEVM:
		client, ok := chains.EVMClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing EVM chain client for selector %d", rawSelector)
		}

		return evm.NewInspector(client), nil
	case chainsel.FamilySolana:
		client, ok := chains.SolanaClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing Solana chain client for selector %d", rawSelector)
		}

		return solana.NewInspector(client), nil
	case chainsel.FamilyAptos:
		client, ok := chains.AptosClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing Aptos chain client for selector %d", rawSelector)
		}
		role, err := aptos.AptosRoleFromAction(action)
		if err != nil {
			return nil, fmt.Errorf("error determining aptos role: %w", err)
		}

		return aptos.NewInspector(client, role), nil
	case chainsel.FamilySui:
		client, signer, ok := chains.SuiClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing Sui chain client for selector %d", rawSelector)
		}
		suiMetadata, err := sdkSui.SuiMetadata(metadata)
		if err != nil {
			return nil, fmt.Errorf("error parsing sui metadata: %w", err)
		}

		return sdkSui.NewInspector(client, signer, suiMetadata.McmsPackageID, suiMetadata.Role)
	default:
		return nil, fmt.Errorf("unsupported chain family %s", family)
	}
}
