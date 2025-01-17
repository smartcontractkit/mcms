package solana

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.TimelockConverter = (*TimelockConverter)(nil)

type TimelockConverter struct {
	client *rpc.Client
	auth   solana.PublicKey
}

func NewTimelockConverter(client *rpc.Client, auth solana.PublicKey) *TimelockConverter {
	return &TimelockConverter{client: client, auth: auth}
}

func (t *TimelockConverter) ConvertBatchToChainOperation(
	ctx context.Context,
	batchOp types.BatchOperation,
	timelockAddress string,
	delay types.Duration,
	action types.TimelockAction,
	predecessor common.Hash,
	salt common.Hash,
) ([]types.Operation, common.Hash, error) {
	return []types.Operation{}, common.Hash{}, fmt.Errorf("not implemented")
}
