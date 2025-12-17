package ton

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"
)

var _ sdk.Inspector = (*Inspector)(nil)

// Inspector is an interface for inspecting on chain state of MCMS contracts.
type Inspector struct {
	client ton.APIClientWrapped

	configTransformer ConfigTransformer
}

// NewInspector creates a new Inspector for TON chains
func NewInspector(client ton.APIClientWrapped) sdk.Inspector {
	return &Inspector{
		client:            client,
		configTransformer: NewConfigTransformer(),
	}
}

func (i Inspector) GetConfig(ctx context.Context, _address string) (*types.Config, error) {
	// Map to Ton Address type (mcms.address)
	addr, err := address.ParseAddr(_address)
	if err != nil {
		return nil, fmt.Errorf("invalid mcms address: %w", err)
	}

	block, err := i.client.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current masterchain info: %w", err)
	}

	_config, err := tvm.CallGetter(ctx, i.client, block, addr, mcms.GetConfig)
	if err != nil {
		return nil, err
	}

	return i.configTransformer.ToConfig(_config)
}

func (i Inspector) GetOpCount(ctx context.Context, _address string) (uint64, error) {
	// Map to Ton Address type (mcms.address)
	addr, err := address.ParseAddr(_address)
	if err != nil {
		return 0, fmt.Errorf("invalid mcms address: %w", err)
	}

	block, err := i.client.CurrentMasterchainInfo(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get current masterchain info: %w", err)
	}

	return tvm.CallGetter(ctx, i.client, block, addr, mcms.GetOpCount)
}

func (i Inspector) GetRoot(ctx context.Context, _address string) (common.Hash, uint32, error) {
	// Map to Ton Address type (mcms.address)
	addr, err := address.ParseAddr(_address)
	if err != nil {
		return [32]byte{}, 0, fmt.Errorf("invalid mcms address: %w", err)
	}

	block, err := i.client.CurrentMasterchainInfo(ctx)
	if err != nil {
		return [32]byte{}, 0, fmt.Errorf("failed to get current masterchain info: %w", err)
	}

	r, err := tvm.CallGetter(ctx, i.client, block, addr, mcms.GetRoot)
	if err != nil {
		return [32]byte{}, 0, err
	}

	return common.BigToHash(r.Root), r.ValidUntil, nil
}

func (i Inspector) GetRootMetadata(ctx context.Context, _address string) (types.ChainMetadata, error) {
	// Map to Ton Address type (mcms.address)
	addr, err := address.ParseAddr(_address)
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("invalid mcms address: %w", err)
	}

	block, err := i.client.CurrentMasterchainInfo(ctx)
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("failed to get current masterchain info: %w", err)
	}

	rm, err := tvm.CallGetter(ctx, i.client, block, addr, mcms.GetRootMetadata)
	if err != nil {
		return types.ChainMetadata{}, err
	}

	return types.ChainMetadata{
		StartingOpCount: rm.PreOpCount,
		MCMAddress:      _address,
	}, nil
}
