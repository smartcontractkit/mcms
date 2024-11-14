package evm

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Inspector = (*Inspector)(nil)

// Inspector is an Inspector implementation for EVM chains, giving access to the state of the MCMS contract
type Inspector struct {
	ConfigTransformer
	client ContractDeployBackend
}

// NewInspector creates a new Inspector for evm chains.
func NewInspector(client ContractDeployBackend) *Inspector {
	return &Inspector{
		ConfigTransformer: ConfigTransformer{},
		client:            client,
	}
}

// GetConfig gets the configurations for the contract.
func (e *Inspector) GetConfig(mcmAddress string) (*types.Config, error) {
	mcmsObj, err := bindings.NewManyChainMultiSig(common.HexToAddress(mcmAddress), e.client)
	if err != nil {
		return nil, err
	}

	onchainConfig, err := mcmsObj.GetConfig(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}

	return e.ToConfig(onchainConfig)
}

// GetOpCount get the operation count of the contract
func (e *Inspector) GetOpCount(mcmAddress string) (uint64, error) {
	mcmsObj, err := bindings.NewManyChainMultiSig(common.HexToAddress(mcmAddress), e.client)
	if err != nil {
		return 0, err
	}

	opCount, err := mcmsObj.GetOpCount(&bind.CallOpts{})
	if err != nil {
		return 0, err
	}

	return opCount.Uint64(), nil
}

// GetRoot gets the merkle root currently in the contract
func (e *Inspector) GetRoot(mcmAddress string) (common.Hash, uint32, error) {
	mcmsObj, err := bindings.NewManyChainMultiSig(common.HexToAddress(mcmAddress), e.client)
	if err != nil {
		return common.Hash{}, 0, err
	}

	root, err := mcmsObj.GetRoot(&bind.CallOpts{})
	if err != nil {
		return common.Hash{}, 0, err
	}

	return root.Root, root.ValidUntil, nil
}

// GetRootMetadata gets the root metadata, specifically the starting op count.
func (e *Inspector) GetRootMetadata(mcmAddress string) (types.ChainMetadata, error) {
	mcmsObj, err := bindings.NewManyChainMultiSig(common.HexToAddress(mcmAddress), e.client)
	if err != nil {
		return types.ChainMetadata{}, err
	}

	metadata, err := mcmsObj.GetRootMetadata(&bind.CallOpts{})
	if err != nil {
		return types.ChainMetadata{}, err
	}

	return types.ChainMetadata{
		StartingOpCount: metadata.PreOpCount.Uint64(),
		MCMAddress:      mcmAddress,
	}, nil
}
