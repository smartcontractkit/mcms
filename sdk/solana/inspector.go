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
	solanaClient *rpc.Client
}

func NewInspector(solanaClient *rpc.Client) *Inspector {
	return &Inspector{
		solanaClient: solanaClient,
	}
}

func (e *Inspector) GetConfig(ctx context.Context, mcmID types.ContractID) (*types.Config, error) {
	panic("implement me")
}

func (e *Inspector) GetOpCount(ctx context.Context, mcmID types.ContractID) (uint64, error) {
	mcmSolanaID, err := FromContractID(mcmID)
	if err != nil {
		return 0, err
	}

	pda, err := e.expiringRootAndOpCountAddress(mcmSolanaID)
	if err != nil {
		return 0, err
	}
	data, err := e.getExpiringRootAndOpCountData(ctx, pda)
	if err != nil {
		return 0, err
	}

	return data.OpCount, err
}

func (e *Inspector) GetRoot(ctx context.Context, mcmID types.ContractID) (common.Hash, uint32, error) {
	mcmSolanaID, err := FromContractID(mcmID)
	if err != nil {
		return common.Hash{}, 0, err
	}

	pda, err := e.expiringRootAndOpCountAddress(mcmSolanaID)
	if err != nil {
		return common.Hash{}, 0, err
	}
	data, err := e.getExpiringRootAndOpCountData(ctx, pda)
	if err != nil {
		return common.Hash{}, 0, err
	}

	return data.Root, data.ValidUntil, err
}

func (e *Inspector) GetRootMetadata(ctx context.Context, mcmID types.ContractID) (types.ChainMetadata, error) {
	panic("implement me")
}

func (e *Inspector) expiringRootAndOpCountAddress(mcmID *SolanaContractID) (solana.PublicKey, error) {
	pda, _, err := solana.FindProgramAddress([][]byte{
		[]byte("expiring_root_and_op_count"),
		mcmID.InstanceID[:],
	}, mcmID.ProgramID)

	return pda, err
}

func (e *Inspector) getExpiringRootAndOpCountData(ctx context.Context, expiringRootAndOpCountPDA solana.PublicKey) (mcm.ExpiringRootAndOpCount, error) {
	var newRootAndOpCount mcm.ExpiringRootAndOpCount
	err := solanaCommon.GetAccountDataBorshInto(ctx, e.solanaClient, expiringRootAndOpCountPDA, config.DefaultCommitment, &newRootAndOpCount)
	if err != nil {
		return mcm.ExpiringRootAndOpCount{}, err
	}

	return newRootAndOpCount, nil
}
