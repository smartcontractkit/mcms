package aptos

import (
	"context"
	"fmt"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/chainlink-internal-integrations/aptos/bindings/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Inspector = &Inspector{}

type Inspector struct {
	ConfigTransformer
	client aptos.AptosRpcClient
}

func NewInspector(client aptos.AptosRpcClient) *Inspector {
	return &Inspector{client: client}
}

func (i Inspector) GetConfig(ctx context.Context, mcmAddr string) (*types.Config, error) {
	mcmsAddress := aptos.AccountAddress{}
	_ = mcmsAddress.ParseStringRelaxed(mcmAddr)
	mcmsC := mcms.Bind(mcmsAddress, i.client)
	config, err := mcmsC.MCMS.GetConfig(nil)
	if err != nil {
		return nil, fmt.Errorf("get config: %w", err)
	}

	return i.ToConfig(config)
}

func (i Inspector) GetOpCount(ctx context.Context, mcmAddr string) (uint64, error) {
	mcmsAddress := aptos.AccountAddress{}
	_ = mcmsAddress.ParseStringRelaxed(mcmAddr)
	mcmsC := mcms.Bind(mcmsAddress, i.client)
	opCount, err := mcmsC.MCMS.GetOpCount(nil)
	if err != nil {
		return 0, fmt.Errorf("get op count: %w", err)
	}

	return opCount, nil
}

func (i Inspector) GetRoot(ctx context.Context, mcmAddr string) (common.Hash, uint32, error) {
	mcmsAddress := aptos.AccountAddress{}
	_ = mcmsAddress.ParseStringRelaxed(mcmAddr)
	mcmsC := mcms.Bind(mcmsAddress, i.client)
	root, validUntil, err := mcmsC.MCMS.GetRoot(nil)
	if err != nil {
		return common.Hash{}, 0, fmt.Errorf("get root: %w", err)
	}

	return root, uint32(validUntil), nil
}

func (i Inspector) GetRootMetadata(ctx context.Context, mcmAddr string) (types.ChainMetadata, error) {
	mcmsAddress := aptos.AccountAddress{}
	_ = mcmsAddress.ParseStringRelaxed(mcmAddr)
	mcmsC := mcms.Bind(mcmsAddress, i.client)

	rootMetadata, err := mcmsC.MCMS.GetRootMetadata(nil)
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("get root metadata: %w", err)
	}

	return types.ChainMetadata{
		StartingOpCount: rootMetadata.PreOpCount,
		MCMAddress:      rootMetadata.Multisig.StringLong(),
	}, nil
}
