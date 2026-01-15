package inspectors

import (
	"fmt"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/chainsmetadata"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"
)

type InspectorFetcher interface {
	FetchInspectors(chainMetadata map[types.ChainSelector]types.ChainMetadata, proposal *mcms.TimelockProposal) (map[types.ChainSelector]sdk.Inspector, error)
}

var _ InspectorFetcher = (*MCMInspectorFetcher)(nil)

type MCMInspectorFetcher struct {
	chains sdk.ChainAccess
}

func NewMCMInspectorFetcher(chains sdk.ChainAccess) *MCMInspectorFetcher {
	return &MCMInspectorFetcher{chains: chains}
}

// FetchInspectors gets a map of inspectors for the given chain metadata and chain clients
func (b *MCMInspectorFetcher) FetchInspectors(
	chainMetadata map[types.ChainSelector]types.ChainMetadata,
	proposal *mcms.TimelockProposal) (map[types.ChainSelector]sdk.Inspector, error) {
	inspectors := map[types.ChainSelector]sdk.Inspector{}
	for chainSelector := range chainMetadata {
		inspector, err := GetInspectorFromChainSelector(b.chains, uint64(chainSelector), proposal)
		if err != nil {
			return nil, fmt.Errorf("error getting inspector for chain selector %d: %w", chainSelector, err)
		}
		inspectors[chainSelector] = inspector
	}

	return inspectors, nil
}

// GetInspectorFromChainSelector returns an inspector for the given chain selector and chain clients
func GetInspectorFromChainSelector(chains sdk.ChainAccess, selector uint64, proposal *mcms.TimelockProposal) (sdk.Inspector, error) {
	fam, err := types.GetChainSelectorFamily(types.ChainSelector(selector))
	if err != nil {
		return nil, fmt.Errorf("error getting chainClient family: %w", err)
	}

	action := proposal.Action
	var inspector sdk.Inspector
	switch fam {
	case chainsel.FamilyEVM:
		client, ok := chains.EVMClient(selector)
		if !ok {
			return nil, fmt.Errorf("missing EVM chain client for selector %d", selector)
		}
		inspector = evm.NewInspector(client)
	case chainsel.FamilySolana:
		client, ok := chains.SolanaClient(selector)
		if !ok {
			return nil, fmt.Errorf("missing Solana chain client for selector %d", selector)
		}
		inspector = solana.NewInspector(client)
	case chainsel.FamilyAptos:
		role, err := chainsmetadata.AptosRoleFromAction(action)
		if err != nil {
			return nil, fmt.Errorf("error getting aptos role from proposal: %w", err)
		}
		client, ok := chains.AptosClient(selector)
		if !ok {
			return nil, fmt.Errorf("missing Aptos chain client for selector %d", selector)
		}
		inspector = aptos.NewInspector(client, role)
	case chainsel.FamilySui:
		metadata, err := chainsmetadata.SuiMetadata(proposal.ChainMetadata[types.ChainSelector(selector)])
		if err != nil {
			return nil, fmt.Errorf("error getting sui metadata from proposal: %w", err)
		}
		client, signer, ok := chains.Sui(selector)
		if !ok {
			return nil, fmt.Errorf("missing Sui chain client for selector %d", selector)
		}
		inspector, err = sui.NewInspector(client, signer, metadata.McmsPackageID, metadata.Role)
		if err != nil {
			return nil, fmt.Errorf("error creating sui inspector: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported chain family %s", fam)
	}

	return inspector, nil
}
