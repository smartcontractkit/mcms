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

func BuildExecutors(
	chains ChainAccessor,
	chainMetadata map[types.ChainSelector]types.ChainMetadata,
	encoders map[types.ChainSelector]sdk.Encoder,
	action types.TimelockAction,
) (map[types.ChainSelector]sdk.Executor, error) {
	executors := map[types.ChainSelector]sdk.Executor{}
	for chainSelector, metadata := range chainMetadata {
		encoder, ok := encoders[chainSelector]
		if !ok {
			return nil, fmt.Errorf("missing encoder for chain selector %d", chainSelector)
		}
		executor, err := BuildExecutor(chains, chainSelector, encoder, action, metadata)
		if err != nil {
			return nil, err
		}
		executors[chainSelector] = executor
	}
	return executors, nil
}

func BuildExecutor(
	chains ChainAccessor,
	selector types.ChainSelector,
	encoder sdk.Encoder,
	action types.TimelockAction,
	metadata types.ChainMetadata,
) (sdk.Executor, error) {
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
		cantonEncoder, ok := encoder.(*cantonsdk.Encoder)
		if !ok {
			return nil, fmt.Errorf("invalid canton encoder type for selector %d: %T", selector, encoder)
		}
		ch, ok := chains.CantonChain(rawSelector)
		if !ok || len(ch.Participants) == 0 {
			return nil, fmt.Errorf("missing Canton chain participant for selector %d", rawSelector)
		}
		participant := ch.Participants[0]
		inspector := cantonsdk.NewInspector(participant.LedgerServices.State, participant.PartyID, cantonRole(action))
		return cantonsdk.NewExecutor(cantonEncoder, inspector, participant.LedgerServices.Command, participant.UserID, participant.PartyID, cantonRole(action))
	case chainsel.FamilyEVM:
		evmEncoder, ok := encoder.(*evmsdk.Encoder)
		if !ok {
			return nil, fmt.Errorf("invalid EVM encoder type for selector %d: %T", selector, encoder)
		}
		client, ok := chains.EVMClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing EVM chain client for selector %d", rawSelector)
		}
		auth, ok := chains.EVMSigner(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing EVM signer for selector %d", rawSelector)
		}
		return evmsdk.NewExecutor(evmEncoder, client, auth), nil
	case chainsel.FamilySolana:
		solanaEncoder, ok := encoder.(*solanasdk.Encoder)
		if !ok {
			return nil, fmt.Errorf("invalid Solana encoder type for selector %d: %T", selector, encoder)
		}
		client, ok := chains.SolanaClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing Solana chain client for selector %d", rawSelector)
		}
		signer, ok := chains.SolanaSigner(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing Solana signer for selector %d", rawSelector)
		}
		return solanasdk.NewExecutor(solanaEncoder, client, *signer), nil
	case chainsel.FamilyAptos:
		aptosEncoder, ok := encoder.(*aptossdk.Encoder)
		if !ok {
			return nil, fmt.Errorf("invalid Aptos encoder type for selector %d: %T", selector, encoder)
		}
		client, ok := chains.AptosClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing Aptos chain client for selector %d", rawSelector)
		}
		signer, ok := chains.AptosSigner(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing Aptos signer for selector %d", rawSelector)
		}
		role, err := aptossdk.AptosRoleFromAction(action)
		if err != nil {
			return nil, fmt.Errorf("error determining aptos role: %w", err)
		}
		return aptossdk.NewExecutor(client, signer, aptosEncoder, role), nil
	case chainsel.FamilySui:
		suiEncoder, ok := encoder.(*suisdk.Encoder)
		if !ok {
			return nil, fmt.Errorf("invalid Sui encoder type for selector %d: %T", selector, encoder)
		}
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
		return suisdk.NewExecutor(client, signer, suiEncoder, entrypointEncoder, suiMetadata.McmsPackageID, suiMetadata.Role, metadata.MCMAddress, suiMetadata.AccountObj, suiMetadata.RegistryObj, suiMetadata.TimelockObj)
	case chainsel.FamilyTon:
		tonEncoder, ok := encoder.(*tonsdk.Encoder)
		if !ok {
			return nil, fmt.Errorf("invalid TON encoder type for selector %d: %T", selector, encoder)
		}
		client, ok := chains.TonClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing TON chain client for selector %d", rawSelector)
		}
		wallet, ok := chains.TonSigner(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing TON signer for selector %d", rawSelector)
		}
		return tonsdk.NewExecutor(tonsdk.ExecutorOpts{Encoder: tonEncoder, Client: client, Wallet: wallet, Amount: tonsdk.DefaultSendAmount})
	default:
		return nil, fmt.Errorf("unsupported chain family %s", family)
	}
}
