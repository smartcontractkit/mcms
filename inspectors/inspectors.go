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
	chains sdk.BlockChains
}

func NewMCMInspectorFetcher(chains sdk.BlockChains) *MCMInspectorFetcher {
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
func GetInspectorFromChainSelector(chains sdk.BlockChains, selector uint64, proposal *mcms.TimelockProposal) (sdk.Inspector, error) {
	fam, err := types.GetChainSelectorFamily(types.ChainSelector(selector))
	if err != nil {
		return nil, fmt.Errorf("error getting chainClient family: %w", err)
	}

	action := proposal.Action
	var inspector sdk.Inspector
	switch fam {
	case chainsel.FamilyEVM:
		inspector = evm.NewInspector(chains.EVMChains()[selector].GetClient())
	case chainsel.FamilySolana:
		inspector = solana.NewInspector(chains.SolanaChains()[selector].GetClient())
	case chainsel.FamilyAptos:
		role, err := chainsmetadata.AptosRoleFromAction(action)
		if err != nil {
			return nil, fmt.Errorf("error getting aptos role from proposal: %w", err)
		}
		chainClient := chains.AptosChains()[selector]
		inspector = aptos.NewInspector(chainClient.GetClient(), role)
	case chainsel.FamilySui:
		metadata, err := chainsmetadata.SuiMetadata(proposal.ChainMetadata[types.ChainSelector(selector)])
		if err != nil {
			return nil, fmt.Errorf("error getting sui metadata from proposal: %w", err)
		}
		chain := chains.SuiChains()[selector]
		inspector, err = sui.NewInspector(chain.GetClient(), chain.GetSigner(), metadata.McmsPackageID, metadata.Role)
		if err != nil {
			return nil, fmt.Errorf("error creating sui inspector: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported chain family %s", fam)
	}

	return inspector, nil
}
