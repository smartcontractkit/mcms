package ton

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Inspector = (*Inspector)(nil)

// Inspector is an interface for inspecting on chain state of MCMS contracts.
type Inspector struct {
	client ton.APIClientWrapped

	configTransformer ConfigTransformer
}

// NewInspector creates a new Inspector for EVM chains
func NewInspector(client ton.APIClientWrapped, configTransformer ConfigTransformer) sdk.Inspector {
	return &Inspector{
		client:            client,
		configTransformer: configTransformer,
	}
}

func (i *Inspector) GetConfig(ctx context.Context, _address string) (*types.Config, error) {
	// Map to Ton Address type (mcms.address)
	addr, err := address.ParseAddr(_address)
	if err != nil {
		return nil, fmt.Errorf("invalid mcms address: %w", err)
	}

	// TODO: mv and import from github.com/smartcontractkit/chainlink-ton/bindings/mcms/mcms
	block, err := i.client.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current masterchain info: %w", err)
	}

	r, err := i.client.RunGetMethod(ctx, block, addr, "getConfig")
	if err != nil {
		return nil, fmt.Errorf("error getting getConfig: %w", err)
	}

	rc0, err := r.Cell(0)
	if err != nil {
		return nil, fmt.Errorf("error getting Config.Signers cell(0): %w", err)
	}

	keySz := uint(8)
	signers := cell.NewDict(keySz)
	if rc0 != nil {
		signers = rc0.AsDict(keySz)
	}

	rc1, err := r.Cell(1)
	if err != nil {
		return nil, fmt.Errorf("error getting Config.GroupQuorums cell(1): %w", err)
	}

	groupQuorums := cell.NewDict(keySz)
	if rc0 != nil {
		groupQuorums = rc1.AsDict(keySz)
	}

	rc2, err := r.Cell(2)
	if err != nil {
		return nil, fmt.Errorf("error getting Config.GroupParents cell(2): %w", err)
	}

	groupParents := cell.NewDict(keySz)
	if rc0 != nil {
		groupParents = rc2.AsDict(keySz)
	}

	return i.configTransformer.ToConfig(mcms.Config{
		Signers:      signers,
		GroupQuorums: groupQuorums,
		GroupParents: groupParents,
	})
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

	r, err := i.client.RunGetMethod(ctx, block, addr, "getOpCount")
	if err != nil {
		return 0, fmt.Errorf("error getting getOpCount: %w", err)
	}

	ri, err := r.Int(0)
	if err != nil {
		return 0, fmt.Errorf("error getting opCount slice: %w", err)
	}

	return ri.Uint64(), nil
}

func (i *Inspector) GetRoot(ctx context.Context, _address string) (common.Hash, uint32, error) {
	// Map to Ton Address type (mcms.address)
	addr, err := address.ParseAddr(_address)
	if err != nil {
		return [32]byte{}, 0, fmt.Errorf("invalid mcms address: %w", err)
	}

	// TODO: mv and import from github.com/smartcontractkit/chainlink-ton/bindings/mcms/mcms
	block, err := i.client.CurrentMasterchainInfo(ctx)
	if err != nil {
		return [32]byte{}, 0, fmt.Errorf("failed to get current masterchain info: %w", err)
	}

	r, err := i.client.RunGetMethod(ctx, block, addr, "getRoot")
	if err != nil {
		return [32]byte{}, 0, fmt.Errorf("error getting getRoot: %w", err)
	}

	root, err := r.Int(0)
	if err != nil {
		return [32]byte{}, 0, fmt.Errorf("error getting Int(0) - root: %w", err)
	}

	validUntil, err := r.Int(1)
	if err != nil {
		return [32]byte{}, 0, fmt.Errorf("error getting Int(1) - validUntil: %w", err)
	}

	return common.Hash(root.Bytes()), uint32(validUntil.Uint64()), nil
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
		rs, err := result.Slice(0)
		if err != nil {
			return types.ChainMetadata{}, fmt.Errorf("error getting slice: %w", err)
		}

		// skip two data points
		rs.LoadBigInt(256)
		rs.LoadAddr()

		preOpCount, err = rs.LoadBigUInt(40)
		if err != nil {
			return types.ChainMetadata{}, fmt.Errorf("error getting preOpCount: %w", err)
		}
	}

	return types.ChainMetadata{
		StartingOpCount: preOpCount.Uint64(),
		MCMAddress:      _address,
	}, nil
}
