package solana

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Inspector = (*Inspector)(nil)

// Inspector is an Inspector implementation for Solana chains, giving access to the state of the MCMS contract
type Inspector struct {
}

func (e *Inspector) GetConfig(mcmAddress string) (*types.Config, error) {
	panic("implement me")
}

func (e *Inspector) GetOpCount(mcmAddress string) (uint64, error) {
	// todo: placeholder for now, to import the package from chainlink-ccip/chains/solana/gobindings/mcm
	mcm.NewExecuteInstructionBuilder().GetExpiringRootAndOpCountAccount()
	panic("implement me")
}

func (e *Inspector) GetRoot(mcmAddress string) (common.Hash, uint32, error) {
	panic("implement me")
}

func (e *Inspector) GetRootMetadata(mcmAddress string) (types.ChainMetadata, error) {
	panic("implement me")
}
