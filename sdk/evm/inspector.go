package evm

import (
	"context"

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

// NewInspector creates a new Inspector for EVM chains
func NewInspector(client ContractDeployBackend) *Inspector {
	return &Inspector{
		ConfigTransformer: ConfigTransformer{},
		client:            client,
	}
}

func (e *Inspector) GetConfig(_ context.Context, mcmID types.ContractID) (*types.Config, error) {
	mcmAddress, err := AddressFromContractID(mcmID)
	if err != nil {
		return nil, err
	}

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

func (e *Inspector) GetOpCount(_ context.Context, mcmID types.ContractID) (uint64, error) {
	mcmAddress, err := AddressFromContractID(mcmID)
	if err != nil {
		return 0, err
	}

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

func (e *Inspector) GetRoot(_ context.Context, mcmID types.ContractID) (common.Hash, uint32, error) {
	mcmAddress, err := AddressFromContractID(mcmID)
	if err != nil {
		return common.Hash{}, 0, err
	}

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

func (e *Inspector) GetRootMetadata(_ context.Context, mcmID types.ContractID) (types.ChainMetadata, error) {
	mcmAddress, err := AddressFromContractID(mcmID)
	if err != nil {
		return types.ChainMetadata{}, err
	}

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
