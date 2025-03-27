package aptos

import (
	"context"
	"fmt"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/chainlink-aptos/bindings/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Inspector = &Inspector{}

type Inspector struct {
	ConfigTransformer
	client aptos.AptosRpcClient

	bindingFn func(address aptos.AccountAddress, client aptos.AptosRpcClient) mcms.MCMS
}

func NewInspector(client aptos.AptosRpcClient) *Inspector {
	return &Inspector{
		client:    client,
		bindingFn: mcms.Bind,
	}
}

func (i Inspector) GetConfig(ctx context.Context, mcmsAddr string) (*types.Config, error) {
	mcmsAddress, err := hexToAddress(mcmsAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MCMS address: %w", err)
	}
	mcmsBinding := i.bindingFn(mcmsAddress, i.client)

	config, err := mcmsBinding.MCMS().GetConfig(nil)
	if err != nil {
		return nil, fmt.Errorf("get config: %w", err)
	}

	return i.ToConfig(config)
}

func (i Inspector) GetOpCount(ctx context.Context, mcmsAddr string) (uint64, error) {
	mcmsAddress, err := hexToAddress(mcmsAddr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse MCMS address: %w", err)
	}
	mcmsBinding := i.bindingFn(mcmsAddress, i.client)

	opCount, err := mcmsBinding.MCMS().GetOpCount(nil)
	if err != nil {
		return 0, fmt.Errorf("get op count: %w", err)
	}

	return opCount, nil
}

func (i Inspector) GetRoot(ctx context.Context, mcmsAddr string) (common.Hash, uint32, error) {
	mcmsAddress, err := hexToAddress(mcmsAddr)
	if err != nil {
		return common.Hash{}, 0, fmt.Errorf("failed to parse MCMS address: %w", err)
	}
	mcmsBinding := i.bindingFn(mcmsAddress, i.client)

	root, validUntil, err := mcmsBinding.MCMS().GetRoot(nil)
	if err != nil {
		return common.Hash{}, 0, fmt.Errorf("get root: %w", err)
	}

	return common.BytesToHash(root), uint32(validUntil), nil
}

func (i Inspector) GetRootMetadata(ctx context.Context, mcmsAddr string) (types.ChainMetadata, error) {
	mcmsAddress, err := hexToAddress(mcmsAddr)
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("failed to parse MCMS address: %w", err)
	}
	mcmsBinding := i.bindingFn(mcmsAddress, i.client)

	rootMetadata, err := mcmsBinding.MCMS().GetRootMetadata(nil)
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("get root metadata: %w", err)
	}

	return types.ChainMetadata{
		StartingOpCount: rootMetadata.PreOpCount,
		MCMAddress:      rootMetadata.Multisig.StringLong(),
	}, nil
}
