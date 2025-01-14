package solana

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/timelock"
	solanaCommon "github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/common"

	"github.com/smartcontractkit/mcms/sdk"
)

var _ sdk.TimelockInspector = (*TimelockInspector)(nil)

var TimelockOpDoneTimestamp = uint64(1)

// TimelockInspector is an Inspector implementation for Solana chains for accessing the RBACTimelock contract
type TimelockInspector struct {
	client *rpc.Client
}

// NewTimelockInspector creates a new TimelockInspector
func NewTimelockInspector(client *rpc.Client) *TimelockInspector {
	return &TimelockInspector{client: client}
}

func (t TimelockInspector) GetProposers(ctx context.Context, address string) ([]common.Address, error) {
	panic("implement me")
}

func (t TimelockInspector) GetExecutors(ctx context.Context, address string) ([]common.Address, error) {
	panic("implement me")
}

func (t TimelockInspector) GetBypassers(ctx context.Context, address string) ([]common.Address, error) {
	panic("implement me")
}

func (t TimelockInspector) GetCancellers(ctx context.Context, address string) ([]common.Address, error) {
	panic("implement me")
}

func (t TimelockInspector) IsOperation(ctx context.Context, address string, opID [32]byte) (bool, error) {
	_, err := t.getOpData(ctx, address, opID)
	if err != nil {
		if errors.Is(err, rpc.ErrNotFound) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (t TimelockInspector) IsOperationPending(ctx context.Context, address string, opID [32]byte) (bool, error) {
	op, err := t.getOpData(ctx, address, opID)
	if err != nil {
		return false, err
	}

	return op.Timestamp > TimelockOpDoneTimestamp, nil
}

func (t TimelockInspector) IsOperationReady(ctx context.Context, address string, opID [32]byte) (bool, error) {
	op, err := t.getOpData(ctx, address, opID)
	if err != nil {
		return false, err
	}

	blockTime, err := solanaCommon.GetBlockTime(ctx, t.client, rpc.CommitmentConfirmed)
	if err != nil {
		return false, err
	}

	ts, err := safeConvertUint64ToInt64(op.Timestamp)
	if err != nil {
		return false, err
	}

	// a safety catch to prevent nil blocktime - ideally should not happen
	if blockTime == nil {
		return false, errors.New("failed to get block time: nil value")
	}

	return op.Timestamp > TimelockOpDoneTimestamp && ts <= int64(*blockTime), nil
}

func (t TimelockInspector) IsOperationDone(ctx context.Context, address string, opID [32]byte) (bool, error) {
	op, err := t.getOpData(ctx, address, opID)
	if err != nil {
		return false, err
	}

	return op.Timestamp == TimelockOpDoneTimestamp, nil
}

func (t TimelockInspector) getOpData(ctx context.Context, address string, opID [32]byte) (timelock.Operation, error) {
	programID, seed, err := ParseContractAddress(address)
	if err != nil {
		return timelock.Operation{}, err
	}

	mcm.SetProgramID(programID)

	pda, err := FindTimelockOperationPDA(programID, seed, opID)
	if err != nil {
		return timelock.Operation{}, err
	}

	var opAccount timelock.Operation
	err = solanaCommon.GetAccountDataBorshInto(ctx, t.client, pda, rpc.CommitmentConfirmed, &opAccount)
	if err != nil {
		return timelock.Operation{}, err
	}

	return opAccount, nil
}

func safeConvertUint64ToInt64(value uint64) (int64, error) {
	const maxInt64 = int64(^uint64(0) >> 1)

	if value > uint64(maxInt64) {
		return 0, errors.New("cannot convert uint64 to int64: value out of int64 range")
	}

	return int64(value), nil
}
