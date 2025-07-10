package sui

import (
	"context"
	"fmt"

	"github.com/block-vision/sui-go-sdk/sui"
	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
	bindutils "github.com/smartcontractkit/chainlink-sui/bindings/utils"

	module_mcms "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Inspector = &Inspector{}

type Inspector struct {
	ConfigTransformer
	client        sui.ISuiAPI
	signer        bindutils.SuiSigner
	mcmsPackageId string
	mcms          module_mcms.IMcms
	role          TimelockRole
}

type ConfigTransformer struct {
	evmTransformer evm.ConfigTransformer
}

func NewConfigTransformer() *ConfigTransformer { return &ConfigTransformer{} }

func (c *ConfigTransformer) ToConfig(config module_mcms.Config) (*types.Config, error) {
	// Re-using the EVM implementation here, but need to convert input first
	evmConfig := bindings.ManyChainMultiSigConfig{
		Signers:      nil,
		GroupQuorums: [32]uint8{},
		GroupParents: [32]uint8{},
	}

	// Convert GroupQuorums slice to array
	for i, quorum := range config.GroupQuorums {
		if i < 32 {
			evmConfig.GroupQuorums[i] = quorum
		}
	}

	// Convert GroupParents slice to array
	for i, parent := range config.GroupParents {
		if i < 32 {
			evmConfig.GroupParents[i] = parent
		}
	}

	for _, signer := range config.Signers {
		evmConfig.Signers = append(evmConfig.Signers, bindings.ManyChainMultiSigSigner{
			Addr:  common.BytesToAddress(signer.Addr),
			Index: signer.Index,
			Group: signer.Group,
		})
	}

	return c.evmTransformer.ToConfig(evmConfig)
}

func NewInspector(client sui.ISuiAPI, signer bindutils.SuiSigner, mcmsPackageId string, role TimelockRole) (*Inspector, error) {
	mcms, err := module_mcms.NewMcms(mcmsPackageId, client)
	if err != nil {
		return nil, err
	}
	return &Inspector{
		client:        client,
		signer:        signer,
		mcmsPackageId: mcmsPackageId,
		mcms:          mcms,
		role:          role,
	}, nil
}

func (i Inspector) GetConfig(ctx context.Context, mcmsAddr string) (*types.Config, error) {
	stateObj := bind.Object{Id: mcmsAddr}

	opts := &bind.CallOpts{
		Signer: i.signer,
	}

	config, err := i.mcms.DevInspect().GetConfig(ctx, opts, stateObj, i.role.Byte())
	if err != nil {
		return nil, fmt.Errorf("get config: %w", err)
	}

	return i.ToConfig(config)
}

func (i Inspector) GetOpCount(ctx context.Context, mcmsAddr string) (uint64, error) {
	stateObj := bind.Object{Id: mcmsAddr}

	opts := &bind.CallOpts{
		Signer: i.signer,
	}

	opCount, err := i.mcms.DevInspect().GetOpCount(ctx, opts, stateObj, i.role.Byte())
	if err != nil {
		return 0, fmt.Errorf("get op count: %w", err)
	}

	return opCount, nil
}

func (i Inspector) GetRoot(ctx context.Context, mcmsAddr string) (common.Hash, uint32, error) {
	stateObj := bind.Object{Id: mcmsAddr}

	opts := &bind.CallOpts{
		Signer: i.signer,
	}

	result, err := i.mcms.DevInspect().GetRoot(ctx, opts, stateObj, i.role.Byte())
	if err != nil {
		return common.Hash{}, 0, fmt.Errorf("get root: %w", err)
	}

	// The result is []any containing [root []byte, validUntil uint64]
	if len(result) < 2 {
		return common.Hash{}, 0, fmt.Errorf("invalid root result: expected 2 elements, got %d", len(result))
	}

	root, ok := result[0].([]byte)
	if !ok {
		return common.Hash{}, 0, fmt.Errorf("invalid root type: expected []byte")
	}

	validUntil, ok := result[1].(uint64)
	if !ok {
		return common.Hash{}, 0, fmt.Errorf("invalid validUntil type: expected uint64")
	}

	//nolint:gosec
	return common.BytesToHash(root), uint32(validUntil), nil
}

func (i Inspector) GetRootMetadata(ctx context.Context, mcmsAddr string) (types.ChainMetadata, error) {
	stateObj := bind.Object{Id: mcmsAddr}

	opts := &bind.CallOpts{
		Signer: i.signer,
	}

	rootMetadata, err := i.mcms.DevInspect().GetRootMetadata(ctx, opts, stateObj, i.role.Byte())
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("get root metadata: %w", err)
	}

	return types.ChainMetadata{
		StartingOpCount: rootMetadata.PreOpCount,
		MCMAddress:      common.BytesToAddress(rootMetadata.Multisig).Hex(),
	}, nil
}
