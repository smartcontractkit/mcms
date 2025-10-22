package ton

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/mcms/sdk"
)

var _ sdk.TimelockInspector = (*timelockInspector)(nil)

// TimelockInspector is an Inspector implementation for TON, for accessing the MCMS-Timelock contract
type timelockInspector struct {
	client *ton.APIClient
}

func NewTimelockInspector(client *ton.APIClient) sdk.TimelockInspector {
	return &timelockInspector{client}
}

func (i timelockInspector) GetMinDelay(ctx context.Context, _address string) (uint64, error) {
	// Map to Ton Address type (timelock.address)
	addr, err := address.ParseAddr(_address)
	if err != nil {
		return 0, fmt.Errorf("invalid timelock address: %w", err)
	}

	// TODO: mv and import from github.com/smartcontractkit/chainlink-ton/bindings/mcms/timelock
	block, err := i.client.CurrentMasterchainInfo(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get current masterchain info: %w", err)
	}

	result, err := i.client.RunGetMethod(ctx, block, addr, "getMinDelay")
	if err != nil {
		return 0, fmt.Errorf("error getting getMinDelay: %w", err)
	}

	rs, err := result.Slice(0)
	if err != nil {
		return 0, fmt.Errorf("error getting minDelay slice: %w", err)
	}

	return rs.LoadUInt(64)
}

// GetProposers returns the list of addresses with the proposer role
func (i timelockInspector) GetProposers(ctx context.Context, address string) ([]string, error) {
	return nil, errors.New("unimplemented")
}

// GetExecutors returns the list of addresses with the executor role
func (i timelockInspector) GetExecutors(ctx context.Context, address string) ([]string, error) {
	return nil, errors.New("unimplemented")
}

// GetBypassers returns the list of addresses with the bypasser role
func (i timelockInspector) GetBypassers(ctx context.Context, address string) ([]string, error) {
	return nil, errors.New("unimplemented")
}

// GetCancellers returns the list of addresses with the canceller role
func (i timelockInspector) GetCancellers(ctx context.Context, address string) ([]string, error) {
	return nil, errors.New("unimplemented")
}

func (i timelockInspector) IsOperation(ctx context.Context, _address string, opID [32]byte) (bool, error) {
	// Map to Ton Address type (timelock.address)
	addr, err := address.ParseAddr(_address)
	if err != nil {
		return false, fmt.Errorf("invalid timelock address: %w", err)
	}

	// TODO: mv and import from github.com/smartcontractkit/chainlink-ton/bindings/mcms/timelock
	block, err := i.client.CurrentMasterchainInfo(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get current masterchain info: %w", err)
	}

	_opID, err := mapOpIDParam(opID)
	if err != nil {
		return false, fmt.Errorf("failed to map opID param: %w", err)
	}

	result, err := i.client.RunGetMethod(ctx, block, addr, "isOperation", _opID)
	if err != nil {
		return false, fmt.Errorf("error getting isOperation: %w", err)
	}

	rs, err := result.Slice(0)
	if err != nil {
		return false, fmt.Errorf("error getting isOperation slice: %w", err)
	}

	return rs.LoadBoolBit()
}

func (i timelockInspector) IsOperationPending(ctx context.Context, _address string, opID [32]byte) (bool, error) {
	// Map to Ton Address type (timelock.address)
	addr, err := address.ParseAddr(_address)
	if err != nil {
		return false, fmt.Errorf("invalid timelock address: %w", err)
	}

	// TODO: mv and import from github.com/smartcontractkit/chainlink-ton/bindings/mcms/timelock
	block, err := i.client.CurrentMasterchainInfo(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get current masterchain info: %w", err)
	}

	_opID, err := mapOpIDParam(opID)
	if err != nil {
		return false, fmt.Errorf("failed to map opID param: %w", err)
	}

	result, err := i.client.RunGetMethod(ctx, block, addr, "isOperationPending", _opID)
	if err != nil {
		return false, fmt.Errorf("error getting isOperationPending: %w", err)
	}

	rs, err := result.Slice(0)
	if err != nil {
		return false, fmt.Errorf("error getting isOperationPending slice: %w", err)
	}

	return rs.LoadBoolBit()
}

func (i timelockInspector) IsOperationReady(ctx context.Context, _address string, opID [32]byte) (bool, error) {
	// Map to Ton Address type (timelock.address)
	addr, err := address.ParseAddr(_address)
	if err != nil {
		return false, fmt.Errorf("invalid timelock address: %w", err)
	}

	// TODO: mv and import from github.com/smartcontractkit/chainlink-ton/bindings/mcms/timelock
	block, err := i.client.CurrentMasterchainInfo(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get current masterchain info: %w", err)
	}

	_opID, err := mapOpIDParam(opID)
	if err != nil {
		return false, fmt.Errorf("failed to map opID param: %w", err)
	}

	result, err := i.client.RunGetMethod(ctx, block, addr, "isOperationReady", _opID)
	if err != nil {
		return false, fmt.Errorf("error getting isOperationReady: %w", err)
	}

	rs, err := result.Slice(0)
	if err != nil {
		return false, fmt.Errorf("error getting isOperationReady slice: %w", err)
	}

	return rs.LoadBoolBit()
}

func (i timelockInspector) IsOperationDone(ctx context.Context, _address string, opID [32]byte) (bool, error) {
	// Map to Ton Address type (timelock.address)
	addr, err := address.ParseAddr(_address)
	if err != nil {
		return false, fmt.Errorf("invalid timelock address: %w", err)
	}

	// TODO: mv and import from github.com/smartcontractkit/chainlink-ton/bindings/mcms/timelock
	block, err := i.client.CurrentMasterchainInfo(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get current masterchain info: %w", err)
	}

	_opID, err := mapOpIDParam(opID)
	if err != nil {
		return false, fmt.Errorf("failed to map opID param: %w", err)
	}

	result, err := i.client.RunGetMethod(ctx, block, addr, "isOperationDone", _opID)
	if err != nil {
		return false, fmt.Errorf("error getting isOperationDone: %w", err)
	}

	rs, err := result.Slice(0)
	if err != nil {
		return false, fmt.Errorf("error getting isOperationDone slice: %w", err)
	}

	return rs.LoadBoolBit()
}

// Help function to map (encode) opID param to cell.Slice
func mapOpIDParam(opID [32]byte) (*cell.Slice, error) {
	b := cell.BeginCell()
	if err := b.StoreBigUInt(new(big.Int).SetBytes(opID[:]), 256); err != nil {
		return nil, fmt.Errorf("failed to store domain separator: %w", err)
	}

	return b.EndCell().BeginParse(), nil
}
