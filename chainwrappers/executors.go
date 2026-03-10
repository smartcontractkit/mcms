package chainwrappers

import (
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

// BuildExecutors gets a map of executors for the given chain metadata and chain clients
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

// BuildExecutor constructs a chain-family-specific Executor from ChainAccessor plus metadata.
func BuildExecutor(
	chains ChainAccessor,
	chainSelector types.ChainSelector,
	encoder sdk.Encoder,
	action types.TimelockAction,
	metadata types.ChainMetadata,
) (sdk.Executor, error) {
	family, err := types.GetChainSelectorFamily(chainSelector)
	if err != nil {
		return nil, fmt.Errorf("error getting chain family: %w", err)
	}
	rawSelector := uint64(chainSelector)

	switch family {
	case chainsel.FamilyEVM:
		evmEncoder, ok := encoder.(*evm.Encoder)
		if !ok {
			return nil, fmt.Errorf("invalid encoder type for selector %d: %T", chainSelector, encoder)
		}
		client, ok := chains.EVMClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing evm chain client for selector %d", chainSelector)
		}
		auth, ok := chains.EVMSigner(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing evm signer for selector %d", rawSelector)
		}

		evmChainMetadata, err := evm.ParseChainMetadata(metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to parse EVM chain metadata for selector %d: %w", rawSelector, err)
		}
		auth.GasPrice = evmChainMetadata.GasPrice
		auth.GasLimit = evmChainMetadata.GasLimit

		return evm.NewExecutor(evmEncoder, client, auth), nil

	case chainsel.FamilySolana:
		solanaEncoder, ok := encoder.(*solana.Encoder)
		if !ok {
			return nil, fmt.Errorf("invalid encoder type for selector %d: %T", chainSelector, encoder)
		}
		client, ok := chains.SolanaClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing solana chain client for selector %d", chainSelector)
		}
		signer, ok := chains.SolanaSigner(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing solana chain signer for selector %d", chainSelector)
		}

		return solana.NewExecutor(solanaEncoder, client, *signer), nil

	case chainsel.FamilyAptos:
		encoder, ok := encoder.(*aptos.Encoder)
		if !ok {
			return nil, fmt.Errorf("invalid encoder type for selector %d: %T", chainSelector, encoder)
		}
		role, err := aptos.AptosRoleFromAction(action)
		if err != nil {
			return nil, fmt.Errorf("error getting aptos role from proposal: %w", err)
		}
		client, ok := chains.AptosClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing aptos chain client for selector %d", chainSelector)
		}
		signer, ok := chains.AptosSigner(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing aptos chain signer for selector %d", chainSelector)
		}

		return aptos.NewExecutor(client, signer, encoder, role), nil

	case chainsel.FamilySui:
		encoder, ok := encoder.(*sui.Encoder)
		if !ok {
			return nil, fmt.Errorf("invalid encoder type for selector %d: %T", chainSelector, encoder)
		}
		client, ok := chains.SuiClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing sui chain client for selector %d", chainSelector)
		}
		signer, ok := chains.SuiSigner(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing sui chain signer for selector %d", chainSelector)
		}

		suiMetadata, err := sui.SuiMetadata(metadata)
		if err != nil {
			return nil, fmt.Errorf("error getting sui metadata from proposal: %w", err)
		}
		entrypointEncoder := sui.NewCCIPEntrypointArgEncoder(suiMetadata.RegistryObj, suiMetadata.DeployerStateObj)

		return sui.NewExecutor(client, signer, encoder, entrypointEncoder, suiMetadata.McmsPackageID, suiMetadata.Role,
			metadata.MCMAddress, suiMetadata.AccountObj, suiMetadata.RegistryObj, suiMetadata.TimelockObj)

	case chainsel.FamilyTon:
		tonEncoder, ok := encoder.(*ton.Encoder)
		if !ok {
			return nil, fmt.Errorf("invalid encoder type for selector %d: %T", chainSelector, encoder)
		}
		client, ok := chains.TonClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing ton chain client for selector %d", chainSelector)
		}
		signer, ok := chains.TonSigner(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing ton client signer for selector %d", chainSelector)
		}

		return ton.NewExecutor(ton.ExecutorOpts{
			Encoder: tonEncoder,
			Client:  client,
			Wallet:  signer,
			Amount:  ton.DefaultSendAmount,
		})

	default:
		return nil, fmt.Errorf("unsupported chain family %s", family)
	}
}
