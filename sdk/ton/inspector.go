package ton

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Inspector = (*Inspector)(nil)

// Inspector is an interface for inspecting on chain state of MCMS contracts.
type Inspector struct {
	client *ton.APIClient

	configTransformer ConfigTransformer
}

// NewInspector creates a new Inspector for EVM chains
func NewInspector(client *ton.APIClient, configTransformer ConfigTransformer) *Inspector {
	return &Inspector{
		client:            client,
		configTransformer: configTransformer,
	}
}

func (i *Inspector) GetConfig(ctx context.Context, address string) (*types.Config, error) {
	return nil, fmt.Errorf("not implemented")
}

func (i *Inspector) GetOpCount(ctx context.Context, _address string) (uint64, error) {
	// Map to Ton Address type (mcms.address)
	addr, err := address.ParseAddr(_address)
	if err != nil {
		return 0, fmt.Errorf("invalid mcms address: %w", err)
	}

	// TODO: mv and import from github.com/smartcontractkit/chainlink-ton/bindings/mcms/mcms
	block, err := i.client.CurrentMasterchainInfo(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get current masterchain info: %w", err)
	}

	result, err := i.client.RunGetMethod(ctx, block, addr, "getOpCount")
	if err != nil {
		return 0, fmt.Errorf("error getting getOpCount: %w", err)
	}

	rs, err := result.Slice(0)
	if err != nil {
		return 0, fmt.Errorf("error getting opCount slice: %w", err)
	}

	return rs.LoadUInt(64)
}

func (i *Inspector) GetRoot(ctx context.Context, address string) (common.Hash, uint32, error) {
	return common.Hash{}, 0, fmt.Errorf("not implemented")
}

func (i *Inspector) GetRootMetadata(ctx context.Context, _address string) (types.ChainMetadata, error) {
	// Map to Ton Address type (mcms.address)
	addr, err := address.ParseAddr(_address)
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("invalid mcms address: %w", err)
	}

	// TODO: mv and import from github.com/smartcontractkit/chainlink-ton/bindings/mcms/mcms
	block, err := i.client.CurrentMasterchainInfo(ctx)
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("failed to get current masterchain info: %w", err)
	}

	result, err := i.client.RunGetMethod(ctx, block, addr, "getRootMetadata")
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("error getting getRootMetadata: %w", err)
	}

	var preOpCount *big.Int
	{
		rs, err := result.Slice(2)
		if err != nil {
			return types.ChainMetadata{}, fmt.Errorf("error getting slice: %w", err)
		}

		preOpCount, err = rs.LoadBigUInt(64)
		if err != nil {
			return types.ChainMetadata{}, fmt.Errorf("error getting preOpCount: %w", err)
		}
	}

	return types.ChainMetadata{
		StartingOpCount: preOpCount.Uint64(),
		MCMAddress:      _address,
	}, nil
}
