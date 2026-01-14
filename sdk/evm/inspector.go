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
	client sdk.ContractDeployBackend
}

// NewInspector creates a new Inspector for EVM chains
func NewInspector(client sdk.ContractDeployBackend) *Inspector {
	return &Inspector{
		ConfigTransformer: ConfigTransformer{},
		client:            client,
	}
}

func (e *Inspector) GetConfig(ctx context.Context, address string) (*types.Config, error) {
	mcmsObj, err := bindings.NewManyChainMultiSig(common.HexToAddress(address), e.client)
	if err != nil {
		return nil, err
	}

	onchainConfig, err := mcmsObj.GetConfig(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, err
	}

	return e.ToConfig(onchainConfig)
}

func (e *Inspector) GetOpCount(ctx context.Context, address string) (uint64, error) {
	mcmsObj, err := bindings.NewManyChainMultiSig(common.HexToAddress(address), e.client)
	if err != nil {
		return 0, err
	}

	opCount, err := mcmsObj.GetOpCount(&bind.CallOpts{Context: ctx})
	if err != nil {
		return 0, err
	}

	return opCount.Uint64(), nil
}

func (e *Inspector) GetRoot(ctx context.Context, address string) (common.Hash, uint32, error) {
	mcmsObj, err := bindings.NewManyChainMultiSig(common.HexToAddress(address), e.client)
	if err != nil {
		return common.Hash{}, 0, err
	}

	root, err := mcmsObj.GetRoot(&bind.CallOpts{Context: ctx})
	if err != nil {
		return common.Hash{}, 0, err
	}

	return root.Root, root.ValidUntil, nil
}

func (e *Inspector) GetRootMetadata(ctx context.Context, address string) (types.ChainMetadata, error) {
	mcmsObj, err := bindings.NewManyChainMultiSig(common.HexToAddress(address), e.client)
	if err != nil {
		return types.ChainMetadata{}, err
	}

	metadata, err := mcmsObj.GetRootMetadata(&bind.CallOpts{Context: ctx})
	if err != nil {
		return types.ChainMetadata{}, err
	}

	return types.ChainMetadata{
		StartingOpCount: metadata.PreOpCount.Uint64(),
		MCMAddress:      address,
	}, nil
}
