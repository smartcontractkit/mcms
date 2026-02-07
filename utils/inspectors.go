package utils

import (
	"context"
	"fmt"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"
)

// BuildInspectorsForTimelockProposal() gets a map of inspectors for the given proposal
func BuildInspectorsForTimelockProposal(
	proposal mcms.TimelockProposal,
	chainClientProvider BlockChainClientProvider,
) (map[types.ChainSelector]sdk.Inspector, error) {
	inspectors := map[types.ChainSelector]sdk.Inspector{}
	for chainSelector := range proposal.ChainMetadata {
		inspector, err := buildInspectorForChainSelector(proposal, chainSelector, chainClientProvider)
		if err != nil {
			return nil, fmt.Errorf("failed to build inspector for chain selector %d: %w", chainSelector, err)
		}
		inspectors[chainSelector] = inspector
	}

	return inspectors, nil
}

func buildInspectorForChainSelector(
	proposal mcms.TimelockProposal,
	chainSelector types.ChainSelector,
	chainClientProvider BlockChainClientProvider,
) (sdk.Inspector, error) {
	client, err := chainClientProvider.GetClient(uint64(chainSelector))
	if err != nil {
		return nil, fmt.Errorf("failed to get blockchain client for chain selector %d: %w", chainSelector, err)
	}
	inspector, err := getInspectorFromChainSelector(client, chainSelector, proposal.Action)
	if err != nil {
		return nil, fmt.Errorf("failed to get inspector for chain selector %d: %w", chainSelector, err)
	}

	return inspector, nil
}

// BuildInspectorsForChainSelectors allows us to implement convenience wrappers for
// the features we currently have to hide behind the "inspector" interface.
// For instance, GetOpCount.
// Ideally, we'd separate the mcms.TimelockProposal interface from the implementation,
// which could allow use to make GetOpCount a method of the TimelockProposal type itself (right now
// we can't because of import cycles)
func GetOpCount(
	ctx context.Context,
	proposal mcms.TimelockProposal,
	chainSelector types.ChainSelector,
	chainClientProvider BlockChainClientProvider,
) (uint64, error) {
	inspector, err := buildInspectorForChainSelector(proposal, chainSelector, chainClientProvider)
	if err != nil {
		return 0, fmt.Errorf("failed to build inspector for chain selector %d: %w", chainSelector, err)
	}

	chainMetadata, ok := proposal.ChainMetadata[chainSelector]
	if !ok {
		return 0, fmt.Errorf("failed to find chain metadata for chain selector %d: %w", chainSelector, err)
	}

	return inspector.GetOpCount(ctx, chainMetadata.MCMAddress)
}

// getInspectorFromChainSelector returns an inspector for the given chain selector and chain clients
func getInspectorFromChainSelector(
	chainClient any, selector types.ChainSelector, action types.TimelockAction,
) (sdk.Inspector, error) {
	fam, err := types.GetChainSelectorFamily(selector)
	if err != nil {
		return nil, fmt.Errorf("error getting chainClient family: %w", err)
	}

	var inspector sdk.Inspector
	switch fam {
	case chainsel.FamilyEVM:
		evmClient, ok := chainClient.(evm.ContractDeployBackend)
		if !ok {
			return nil, fmt.Errorf("invalid EVM client type for selector %d", selector)
		}
		inspector = evm.NewInspector(evmClient)

	case chainsel.FamilySolana:
		solanaClient, ok := chainClient.(solana.RPCClient)
		if !ok {
			return nil, fmt.Errorf("invalid Solana client type for selector %d", selector)
		}
		inspector = solana.NewInspector(solanaClient)

	case chainsel.FamilyAptos:
		role, rerr := AptosRoleFromAction(action)
		if rerr != nil {
			return nil, fmt.Errorf("error getting aptos role from proposal: %w", rerr)
		}
		aptosClient, ok := chainClient.(aptos.RPCClient)
		if !ok {
			return nil, fmt.Errorf("invalid Aptos client type for selector %d", selector)
		}
		inspector = aptos.NewInspector(aptosClient, role)

	case chainsel.FamilySui:
		suiClient, ok := chainClient.(sui.RPCClient)
		if !ok {
			return nil, fmt.Errorf("invalid Sui client type for selector %d", selector)
		}
		// FIXME: where do we get the bindSigner and mcmsPackageID set as nil and "" below?
		inspector, err = sui.NewInspector(suiClient, nil, "", sui.TimelockRoleFromAction(action))
		if err != nil {
			return nil, fmt.Errorf("failed to create Sui inspector for selector %d", selector)
		}

	default:
		return nil, fmt.Errorf("unsupported chain family %s", fam)
	}

	return inspector, nil
}

// This requires an addition to CLDf which we'd need to negotiate with the CLD team:
// diff --git i/chain/blockchain.go w/chain/blockchain.go
// index e80aac9..9a3b8b6 100644
// --- i/chain/blockchain.go
// +++ w/chain/blockchain.go
//
//	@@ -82,6 +82,18 @@ func (b BlockChains) GetBySelector(selector uint64) (BlockChain, error) {
//	 	return nil, ErrBlockChainNotFound
//	 }
//
// +// GetClient returns the rpc client for the given selector as an opaque type
// +func (b BlockChains) GetClient(selector uint64) (any, error) {
// +	chain, ok := b.chains[selector]
// +	if !ok {
// +		return nil, ErrBlockChainNotFound
// +	}
// +
// +	// FIXME: needs to be tested;
// +	// we could also avoid reflection by adding an "RPCClient()" method to the BlockChain interface
// +	// or we could do a "switch (chain.Type()"
// +	return reflect.ValueOf(chain).Elem().FieldByName("Client").Interface(), nil
// +}
// +
//
//	// Exists checks if a chain with the given selector exists.
//	func (b BlockChains) Exists(selector uint64) bool {
//		_, ok := b.chains[selector]
type BlockChainClientProvider interface {
	GetClient(selector uint64) (any, error)
}
