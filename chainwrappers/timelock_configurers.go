package chainwrappers

import (
	"encoding/json"
	"fmt"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/sdk/ton"
	"github.com/smartcontractkit/mcms/types"
)

// BuildTimelockConfigurers gets a map of timelock configurers for the given
// chain metadata and chain clients.
func BuildTimelockConfigurers(
	chains ChainAccessor,
	chainMetadata map[types.ChainSelector]types.ChainMetadata,
	action types.TimelockAction,
) (map[types.ChainSelector]sdk.TimelockConfigurer, error) {
	configurers := map[types.ChainSelector]sdk.TimelockConfigurer{}
	for selector, metadata := range chainMetadata {
		configurer, err := BuildTimelockConfigurer(chains, selector, action, metadata)
		if err != nil {
			return nil, err
		}
		configurers[selector] = configurer
	}

	return configurers, nil
}

// BuildTimelockConfigurer constructs a chain-family-specific
// [sdk.TimelockConfigurer] from a [ChainAccessor] plus chain metadata.
func BuildTimelockConfigurer(
	chains ChainAccessor,
	selector types.ChainSelector,
	action types.TimelockAction,
	metadata types.ChainMetadata,
) (sdk.TimelockConfigurer, error) {
	_ = action

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
		signer, ok := chains.EVMSigner(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing EVM chain signer for selector %d", rawSelector)
		}

		return evm.NewTimelockConfigurer(client, signer), nil

	case chainsel.FamilySolana:
		client, ok := chains.SolanaClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing Solana chain client for selector %d", rawSelector)
		}
		signer, ok := chains.SolanaSigner(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing Solana chain signer for selector %d", rawSelector)
		}

		return solana.NewTimelockConfigurer(client, *signer), nil

	case chainsel.FamilyAptos:
		client, ok := chains.AptosClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing Aptos chain client for selector %d", rawSelector)
		}
		var afm aptos.AdditionalFieldsMetadata
		if len(metadata.AdditionalFields) > 0 {
			if err = json.Unmarshal(metadata.AdditionalFields, &afm); err != nil {
				return nil, fmt.Errorf("error parsing aptos metadata: %w", err)
			}
		}

		return aptos.NewTimelockConfigurerWithMCMSType(client, afm.MCMSType), nil

	case chainsel.FamilySui:
		suiMetadata, err := sui.SuiMetadata(metadata)
		if err != nil {
			return nil, fmt.Errorf("error parsing sui metadata: %w", err)
		}

		return sui.NewTimelockConfigurer(suiMetadata.McmsPackageID), nil

	case chainsel.FamilyTon:
		w, ok := chains.TonSigner(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing TON chain wallet for selector %d", rawSelector)
		}

		return ton.NewTimelockConfigurer(w, ton.DefaultSendAmount), nil

	default:
		return nil, fmt.Errorf("unsupported chain family %s", family)
	}
}
