package solana

import (
	"context"
	"errors"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/access_controller"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/timelock"
	solanaCommon "github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/common"

	"github.com/smartcontractkit/mcms/internal/utils/safecast"
	"github.com/smartcontractkit/mcms/sdk"
)

var _ sdk.TimelockInspector = (*TimelockInspector)(nil)

// TimelockInspector is an Inspector implementation for Solana chains for accessing the RBACTimelock contract
type TimelockInspector struct {
	client *rpc.Client
}

// NewTimelockInspector creates a new TimelockInspector
func NewTimelockInspector(client *rpc.Client) *TimelockInspector {
	return &TimelockInspector{client: client}
}

func (t TimelockInspector) GetProposers(ctx context.Context, address string) ([]string, error) {
	accessList, err := t.getRoleAccessList(ctx, address, timelock.Proposer_Role)
	if err != nil {
		return nil, err
	}

	return accessList, nil
}

func (t TimelockInspector) GetExecutors(ctx context.Context, address string) ([]string, error) {
	accessList, err := t.getRoleAccessList(ctx, address, timelock.Executor_Role)
	if err != nil {
		return nil, err
	}

	return accessList, nil
}

func (t TimelockInspector) GetBypassers(ctx context.Context, address string) ([]string, error) {
	accessList, err := t.getRoleAccessList(ctx, address, timelock.Bypasser_Role)
	if err != nil {
		return nil, err
	}

	return accessList, nil
}

func (t TimelockInspector) GetCancellers(ctx context.Context, address string) ([]string, error) {
	accessList, err := t.getRoleAccessList(ctx, address, timelock.Canceller_Role)
	if err != nil {
		return nil, err
	}

	return accessList, nil
}

// ---------------------
// Implementation of these IsOperations are based on the rust implementation at
// https://github.com/smartcontractkit/chainlink-ccip/blob/main/chains/solana/contracts/programs/timelock/src/state/operation.rs#L33
// ---------------------

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

	return op.State == timelock.Scheduled_OperationState, nil
}

func (t TimelockInspector) IsOperationReady(ctx context.Context, address string, opID [32]byte) (bool, error) {
	op, err := t.getOpData(ctx, address, opID)
	if err != nil {
		return false, err
	}

	slot, err := t.client.GetSlot(ctx, rpc.CommitmentConfirmed)
	if err != nil {
		return false, fmt.Errorf("failed to get slot: %w", err)
	}

	blockTime, err := t.client.GetBlockTime(ctx, slot)
	if err != nil {
		return false, fmt.Errorf("failed to get block time: %w", err)
	}

	ts, err := safecast.Uint64ToInt64(op.Timestamp)
	if err != nil {
		return false, err
	}

	// a safety catch to prevent nil blocktime - ideally should not happen
	if blockTime == nil {
		return false, errors.New("failed to get block time: nil value")
	}

	return op.State == timelock.Scheduled_OperationState && ts <= int64(*blockTime), nil
}

func (t TimelockInspector) IsOperationDone(ctx context.Context, address string, opID [32]byte) (bool, error) {
	op, err := t.getOpData(ctx, address, opID)
	if err != nil {
		return false, err
	}

	return op.State == timelock.Done_OperationState, nil
}

func (t TimelockInspector) getOpData(ctx context.Context, address string, opID [32]byte) (timelock.Operation, error) {
	programID, seed, err := ParseContractAddress(address)
	if err != nil {
		return timelock.Operation{}, err
	}

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

func (t TimelockInspector) getRoleAccessList(ctx context.Context, address string, role timelock.Role) ([]string, error) {
	programID, seed, err := ParseContractAddress(address)
	if err != nil {
		return nil, err
	}

	pda, err := FindTimelockConfigPDA(programID, seed)
	if err != nil {
		return nil, err
	}

	configAccount, err := GetTimelockConfig(ctx, t.client, pda)
	if err != nil {
		return nil, err
	}

	controller, err := getRoleAccessController(configAccount, role)
	if err != nil {
		return nil, err
	}

	var ac access_controller.AccessController
	err = solanaCommon.GetAccountDataBorshInto(
		ctx,
		t.client,
		controller,
		rpc.CommitmentConfirmed,
		&ac,
	)
	if err != nil {
		return nil, err
	}

	accessListStr := make([]string, 0, ac.AccessList.Len)
	for _, acc := range ac.AccessList.Xs[:ac.AccessList.Len] {
		accessListStr = append(accessListStr, acc.String())
	}

	return accessListStr, nil
}

func GetTimelockConfig(ctx context.Context, client *rpc.Client, configPDA solana.PublicKey) (timelock.Config, error) {
	var config timelock.Config
	err := solanaCommon.GetAccountDataBorshInto(ctx, client, configPDA, rpc.CommitmentConfirmed, &config)
	if err != nil {
		return timelock.Config{}, err
	}

	return config, nil
}

func getRoleAccessController(config timelock.Config, role timelock.Role) (solana.PublicKey, error) {
	switch role {
	case timelock.Proposer_Role:
		return config.ProposerRoleAccessController, nil
	case timelock.Canceller_Role:
		return config.CancellerRoleAccessController, nil
	case timelock.Executor_Role:
		return config.ExecutorRoleAccessController, nil
	case timelock.Bypasser_Role:
		return config.BypasserRoleAccessController, nil
	case timelock.Admin_Role:
		return solana.PublicKey{}, fmt.Errorf("not supported role: %v", role)
	default:
		return solana.PublicKey{}, fmt.Errorf("unknown role")
	}
}
