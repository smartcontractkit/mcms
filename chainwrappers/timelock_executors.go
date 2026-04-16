package chainwrappers

import (
	"fmt"

	chainsel "github.com/smartcontractkit/chain-selectors"
	mcmsencoder "github.com/smartcontractkit/chainlink-sui/bindings"

	"github.com/smartcontractkit/mcms/sdk"
	aptossdk "github.com/smartcontractkit/mcms/sdk/aptos"
	cantonsdk "github.com/smartcontractkit/mcms/sdk/canton"
	evmsdk "github.com/smartcontractkit/mcms/sdk/evm"
	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	suisdk "github.com/smartcontractkit/mcms/sdk/sui"
	tonsdk "github.com/smartcontractkit/mcms/sdk/ton"
	"github.com/smartcontractkit/mcms/types"
)

func BuildTimelockExecutors(
	chains ChainAccessor,
	chainMetadata map[types.ChainSelector]types.ChainMetadata,
	action types.TimelockAction,
) (map[types.ChainSelector]sdk.TimelockExecutor, error) {
	executors := map[types.ChainSelector]sdk.TimelockExecutor{}
	for chainSelector, metadata := range chainMetadata {
		executor, err := BuildTimelockExecutor(chains, chainSelector, action, metadata)
		if err != nil {
			return nil, err
		}
		executors[chainSelector] = executor
	}
	return executors, nil
}

func BuildTimelockExecutor(
	chains ChainAccessor,
	selector types.ChainSelector,
	action types.TimelockAction,
	metadata types.ChainMetadata,
) (sdk.TimelockExecutor, error) {
	if chains == nil {
		return nil, fmt.Errorf("chain access is required")
	}
	_ = action
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
		return cantonsdk.NewTimelockExecutor(participant.LedgerServices.Command, participant.LedgerServices.State, participant.PartyID), nil
	case chainsel.FamilyEVM:
		client, ok := chains.EVMClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing EVM chain client for selector %d", rawSelector)
		}
		auth, ok := chains.EVMSigner(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing EVM signer for selector %d", rawSelector)
		}
		return evmsdk.NewTimelockExecutor(client, auth), nil
	case chainsel.FamilySolana:
		client, ok := chains.SolanaClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing Solana chain client for selector %d", rawSelector)
		}
		signer, ok := chains.SolanaSigner(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing Solana signer for selector %d", rawSelector)
		}
		return solanasdk.NewTimelockExecutor(client, *signer), nil
	case chainsel.FamilyAptos:
		client, ok := chains.AptosClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing Aptos chain client for selector %d", rawSelector)
		}
		signer, ok := chains.AptosSigner(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing Aptos signer for selector %d", rawSelector)
		}
		return aptossdk.NewTimelockExecutor(client, signer), nil
	case chainsel.FamilySui:
		client, ok := chains.SuiClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing Sui chain client for selector %d", rawSelector)
		}
		signer, ok := chains.SuiSigner(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing Sui signer for selector %d", rawSelector)
		}
		suiMetadata, err := suisdk.SuiMetadata(metadata)
		if err != nil {
			return nil, fmt.Errorf("error parsing sui metadata: %w", err)
		}
		entrypointEncoder := mcmsencoder.NewCCIPEntrypointArgEncoder(suiMetadata.RegistryObj, suiMetadata.DeployerStateObj)
		return suisdk.NewTimelockExecutor(client, signer, entrypointEncoder, suiMetadata.McmsPackageID, suiMetadata.RegistryObj, suiMetadata.AccountObj)
	case chainsel.FamilyTon:
		client, ok := chains.TonClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing TON chain client for selector %d", rawSelector)
		}
		wallet, ok := chains.TonSigner(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing TON signer for selector %d", rawSelector)
		}
		return tonsdk.NewTimelockExecutor(tonsdk.TimelockExecutorOpts{Client: client, Wallet: wallet, Amount: tonsdk.DefaultSendAmount})
	default:
		return nil, fmt.Errorf("unsupported chain family %s", family)
	}
}
