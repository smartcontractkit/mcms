package chainwrappers

import (
	"encoding/json"
	"fmt"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/aptos"
	cantonsdk "github.com/smartcontractkit/mcms/sdk/canton"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/sdk/ton"
	"github.com/smartcontractkit/mcms/types"
)

// BuildInspectors gets a map of inspectors for the given chain metadata and chain clients
func BuildInspectors(
	chains ChainAccessor,
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

// BuildInspector constructs a chain-family-specific Inspector from ChainAccessor plus metadata.
func BuildInspector(
	chains ChainAccessor,
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
	case chainsel.FamilyCanton:
		ch, ok := chains.CantonChain(rawSelector)
		if !ok || len(ch.Participants) == 0 {
			return nil, fmt.Errorf("missing Canton chain participant for selector %d", rawSelector)
		}
		participant := ch.Participants[0]
		return cantonsdk.NewInspector(participant.LedgerServices.State, participant.PartyID, cantonRole(action)), nil
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
		var afm aptos.AdditionalFieldsMetadata
		if len(metadata.AdditionalFields) > 0 {
			if err = json.Unmarshal(metadata.AdditionalFields, &afm); err != nil {
				return nil, fmt.Errorf("error parsing aptos metadata: %w", err)
			}
		}

		return aptos.NewInspectorWithMCMSType(client, role, afm.MCMSType), nil
	case chainsel.FamilySui:
		client, ok := chains.SuiClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing Sui chain client for selector %d", rawSelector)
		}
		signer, ok := chains.SuiSigner(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing Sui chain client for selector %d", rawSelector)
		}
		suiMetadata, err := sui.SuiMetadata(metadata)
		if err != nil {
			return nil, fmt.Errorf("error parsing sui metadata: %w", err)
		}

		return sui.NewInspector(client, signer, suiMetadata.McmsPackageID, suiMetadata.Role)
	case chainsel.FamilyTon:
		client, ok := chains.TonClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing Ton chain client for selector %d", rawSelector)
		}

		return ton.NewInspector(client), nil
	default:
		return nil, fmt.Errorf("unsupported chain family %s", family)
	}
}

func cantonRole(action types.TimelockAction) cantonsdk.TimelockRole {
	switch action {
	case types.TimelockActionBypass:
		return cantonsdk.TimelockRoleBypasser
	case types.TimelockActionCancel:
		return cantonsdk.TimelockRoleCanceller
	default:
		return cantonsdk.TimelockRoleProposer
	}
}
