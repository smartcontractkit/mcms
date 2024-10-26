package evm

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/internal/core/config"
	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
	"github.com/smartcontractkit/mcms/internal/evm/bindings"
)

type EVMInspector struct {
	EVMConfigurator
	client ContractDeployBackend
}

func NewEVMInspector(client ContractDeployBackend) *EVMInspector {
	return &EVMInspector{
		EVMConfigurator: EVMConfigurator{},
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

func (e *EVMInspector) GetRootMetadata(mcmAddress string) (mcms.ChainMetadata, error) {
	mcmsObj, err := bindings.NewManyChainMultiSig(common.HexToAddress(mcmAddress), e.client)
	if err != nil {
		return mcms.ChainMetadata{}, nil
	}

	metadata, err := mcmsObj.GetRootMetadata(&bind.CallOpts{})
	if err != nil {
		return mcms.ChainMetadata{}, err
	}

	return mcms.ChainMetadata{
		StartingOpCount: metadata.PreOpCount.Uint64(),
		MCMAddress:      mcmAddress,
	}, nil
}
