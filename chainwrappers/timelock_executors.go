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

// BuildTimelockExecutors gets a map of timelock executors for the given chain metadata and chain clients
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

// BuildTimelockExecutor constructs a chain-family-specific TimelockExecutor from ChainAccessor plus metadata.
func BuildTimelockExecutor(
	chains ChainAccessor,
	chainSelector types.ChainSelector,
	action types.TimelockAction,
	metadata types.ChainMetadata,
) (sdk.TimelockExecutor, error) {
	family, err := types.GetChainSelectorFamily(chainSelector)
	if err != nil {
		return nil, fmt.Errorf("error getting chain family: %w", err)
	}
	rawSelector := uint64(chainSelector)

	switch family {
	case chainsel.FamilyEVM:
		client, ok := chains.EVMClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing evm chain client for selector %d", chainSelector)
		}
		auth, ok := chains.EVMSigner(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing evm signer for selector %d", rawSelector)
		}

		evmChainMetadata, err1 := evm.ParseChainMetadata(metadata)
		if err1 != nil {
			return nil, fmt.Errorf("failed to parse EVM chain metadata for selector %d: %w", rawSelector, err1)
		}
		auth.GasPrice = evmChainMetadata.GasPrice
		auth.GasLimit = evmChainMetadata.GasLimit

		return evm.NewTimelockExecutor(client, auth), nil

	case chainsel.FamilySolana:
		client, ok := chains.SolanaClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing solana chain client for selector %d", chainSelector)
		}
		signer, ok := chains.SolanaSigner(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing solana chain signer for selector %d", chainSelector)
		}

		return solana.NewTimelockExecutor(client, *signer), nil

	case chainsel.FamilyAptos:
		client, ok := chains.AptosClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing aptos chain client for selector %d", chainSelector)
		}
		signer, ok := chains.AptosSigner(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing aptos chain signer for selector %d", chainSelector)
		}

		mcmsType := aptos.MCMSTypeRegular
		if len(metadata.AdditionalFields) > 0 {
			var afm aptos.AdditionalFieldsMetadata
			if err = json.Unmarshal(metadata.AdditionalFields, &afm); err != nil {
				return nil, fmt.Errorf("failed to parse Aptos metadata for selector %d: %w", rawSelector, err)
			}
			mcmsType = afm.MCMSType
		}

		return aptos.NewTimelockExecutorWithMCMSType(client, signer, mcmsType), nil

	case chainsel.FamilySui:
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

		return sui.NewTimelockExecutor(client, signer, entrypointEncoder, suiMetadata.McmsPackageID,
			suiMetadata.RegistryObj, suiMetadata.AccountObj)

	case chainsel.FamilyTon:
		client, ok := chains.TonClient(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing ton chain client for selector %d", chainSelector)
		}
		signer, ok := chains.TonSigner(rawSelector)
		if !ok {
			return nil, fmt.Errorf("missing ton client signer for selector %d", chainSelector)
		}

		return ton.NewTimelockExecutor(ton.TimelockExecutorOpts{
			Client: client,
			Wallet: signer,
			Amount: ton.DefaultSendAmount,
		})

	default:
		return nil, fmt.Errorf("unsupported chain family %s", family)
	}
}
