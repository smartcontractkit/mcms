package sui

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pattonkan/sui-go/suiclient"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Inspector = &Inspector{}

type Inspector struct {
	client        suiclient.ClientImpl
	mcmsPackageId string
}

func NewInspector(client suiclient.ClientImpl, mcmsPackageId string) *Inspector {
	return &Inspector{
		client:        client,
		mcmsPackageId: mcmsPackageId,
	}
}

func (i Inspector) GetConfig(ctx context.Context, mcmsAddr string) (*types.Config, error) {
	return nil, nil
}

func (i Inspector) GetOpCount(ctx context.Context, mcmsAddr string) (uint64, error) {
	return 0, nil
}

func (i Inspector) GetRoot(ctx context.Context, mcmsAddr string) (common.Hash, uint32, error) {
	return common.Hash{}, 0, nil
}

func (i Inspector) GetRootMetadata(ctx context.Context, mcmsAddr string) (types.ChainMetadata, error) {
	return types.ChainMetadata{}, nil
}
