package solana

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Inspector = (*Inspector)(nil)

// Inspector is an Inspector implementation for Solana chains, giving access to the state of the MCMS contract
type Inspector struct {
}

func (e *Inspector) GetConfig(ctx context.Context, mcmAddress string) (*types.Config, error) {
	panic("implement me")
}

func (e *Inspector) GetOpCount(ctx context.Context, mcmAddress string) (uint64, error) {
	// todo: placeholder for now, to import the package from chainlink-ccip/chains/solana/gobindings/mcm
	mcm.NewExecuteInstructionBuilder().GetExpiringRootAndOpCountAccount()
	panic("implement me")
}

func (e *Inspector) GetRoot(ctx context.Context, mcmAddress string) (common.Hash, uint32, error) {
	panic("implement me")
}

func (e *Inspector) GetRootMetadata(ctx context.Context, mcmAddress string) (types.ChainMetadata, error) {
	panic("implement me")
}
