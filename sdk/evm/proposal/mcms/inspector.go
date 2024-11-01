package evm

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/sdk"

	"github.com/smartcontractkit/mcms/internal/core/config"

	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	evm_config "github.com/smartcontractkit/mcms/sdk/evm/config"
	"github.com/smartcontractkit/mcms/types"
)

// EVMInspector is an Inspector implementation for EVM chains, giving access to the state of the MCMS contract
type EVMInspector struct {
	evm_config.EVMConfigurator
	client evm.ContractDeployBackend
}

var _ sdk.Inspector = &EVMInspector{}

func NewEVMInspector(client evm.ContractDeployBackend) *EVMInspector {
	return &EVMInspector{
		EVMConfigurator: evm_config.EVMConfigurator{},
		client:          client,
	}
}

func (e *EVMInspector) GetConfig(mcmAddress string) (*config.Config, error) {
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

func (e *EVMInspector) GetOpCount(mcmAddress string) (uint64, error) {
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

func (e *EVMInspector) GetRoot(mcmAddress string) (common.Hash, uint32, error) {
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

func (e *EVMInspector) GetRootMetadata(mcmAddress string) (types.ChainMetadata, error) {
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
