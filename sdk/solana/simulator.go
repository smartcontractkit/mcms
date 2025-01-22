package solana

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Simulator = &Simulator{}

type Simulator struct {
	*Executor
}

func NewSimulator(executor *Executor) *Simulator {
	return &Simulator{
		executor,
	}
}

func (s Simulator) SimulateSetRoot(
	ctx context.Context, _ string,
	metadata types.ChainMetadata, proof []common.Hash, root [32]byte,
	validUntil uint32, sortedSignatures []types.Signature,
) error {
	err := s.EnableSimulation(ctx, s.client, s.auth,
		rpc.SimulateTransactionOpts{
			Commitment: rpc.CommitmentConfirmed,
		},
		func() error {
			_, err := s.SetRoot(ctx, metadata, proof, root, validUntil, sortedSignatures)
			return err
		})

	return err
}

func (s Simulator) SimulateOperation(
	ctx context.Context, metadata types.ChainMetadata, operation types.Operation) error {
	panic("implement me")
}
