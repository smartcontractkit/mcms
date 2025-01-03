package solana

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/contracts/tests/config"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	solanaCommon "github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/common"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Inspector = (*Inspector)(nil)

// Inspector is an Inspector implementation for Solana chains, giving access to the state of the MCMS contract
type Inspector struct {
	ctx          context.Context
	solanaClient *rpc.Client
}

func NewInspector(ctx context.Context, solanaClient *rpc.Client) *Inspector {
	return &Inspector{
		ctx:          ctx,
		solanaClient: solanaClient,
	}
}

func (e *Inspector) GetConfig(multisigConfigPDA string) (*types.Config, error) {
	panic("implement me")
}

func (e *Inspector) GetOpCount(expiringRootAndOpCountPDA string) (uint64, error) {
	data, err := e.getExpiringRootAndOpCountData(expiringRootAndOpCountPDA)
	if err != nil {
		return 0, err
	}

	return data.OpCount, err
}

func (e *Inspector) GetRoot(expiringRootAndOpCountPDA string) (common.Hash, uint32, error) {
	data, err := e.getExpiringRootAndOpCountData(expiringRootAndOpCountPDA)
	if err != nil {
		return common.Hash{}, 0, err
	}

	return data.Root, data.ValidUntil, err
}

func (e *Inspector) GetRootMetadata(expiringRootAndOpCountPDA string) (types.ChainMetadata, error) {
	panic("implement me")
}

func (e *Inspector) getExpiringRootAndOpCountData(expiringRootAndOpCountPDA string) (mcm.ExpiringRootAndOpCount, error) {
	pdaAddr, err := solana.PublicKeyFromBase58(expiringRootAndOpCountPDA)
	if err != nil {
		return mcm.ExpiringRootAndOpCount{}, err
	}
	var newRootAndOpCount mcm.ExpiringRootAndOpCount
	err = solanaCommon.GetAccountDataBorshInto(e.ctx, e.solanaClient, pdaAddr, config.DefaultCommitment, &newRootAndOpCount)
	if err != nil {
		return mcm.ExpiringRootAndOpCount{}, err
	}

	return newRootAndOpCount, nil
}
