package ton

import (
	"context"
	"fmt"
	"math/big"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/timelock"
	"github.com/smartcontractkit/mcms/sdk"
)

var _ sdk.TimelockInspector = (*timelockInspector)(nil)

// TimelockInspector is an Inspector implementation for TON, for accessing the MCMS-Timelock contract
type timelockInspector struct {
	client ton.APIClientWrapped
}

func NewTimelockInspector(client ton.APIClientWrapped) sdk.TimelockInspector {
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

// GetAdmins returns the list of addresses with the admin role
func (i timelockInspector) GetAdmins(ctx context.Context, address string) ([]string, error) {
	return i.getRoleMembers(ctx, address, [32]byte(timelock.RoleAdmin.Bytes()))
}

// GetProposers returns the list of addresses with the proposer role
func (i timelockInspector) GetProposers(ctx context.Context, address string) ([]string, error) {
	return i.getRoleMembers(ctx, address, [32]byte(timelock.RoleProposer.Bytes()))
}

// GetExecutors returns the list of addresses with the executor role
func (i timelockInspector) GetExecutors(ctx context.Context, address string) ([]string, error) {
	return i.getRoleMembers(ctx, address, [32]byte(timelock.RoleExecutor.Bytes()))
}

// GetBypassers returns the list of addresses with the bypasser role
func (i timelockInspector) GetBypassers(ctx context.Context, address string) ([]string, error) {
	return i.getRoleMembers(ctx, address, [32]byte(timelock.RoleBaypasser.Bytes()))
}

// GetCancellers returns the list of addresses with the canceller role
func (i timelockInspector) GetCancellers(ctx context.Context, address string) ([]string, error) {
	return i.getRoleMembers(ctx, address, [32]byte(timelock.RoleCanceller.Bytes()))
}

// GetOracles returns the list of addresses with the oracle role
func (i timelockInspector) GetOracles(ctx context.Context, address string) ([]string, error) {
	return i.getRoleMembers(ctx, address, [32]byte(timelock.RoleOracle.Bytes()))
}

// getRoleMembers returns the list of addresses with the given role
func (i timelockInspector) getRoleMembers(ctx context.Context, _address string, role [32]byte) ([]string, error) {
	// Map to Ton Address type (timelock.address)
	addr, err := address.ParseAddr(_address)
	if err != nil {
		return nil, fmt.Errorf("invalid timelock address: %w", err)
	}

	// TODO: mv and import from github.com/smartcontractkit/chainlink-ton/bindings/mcms/timelock
	block, err := i.client.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current masterchain info: %w", err)
	}

	_role, err := mapRoleParam(role)
	if err != nil {
		return nil, fmt.Errorf("failed to map opID param: %w", err)
	}

	r, err := i.client.RunGetMethod(ctx, block, addr, "getRoleMemberCount", _role)
	if err != nil {
		return nil, fmt.Errorf("error getting getRoleMemberCount: %w", err)
	}

	count, err := r.Int(0)
	if err != nil {
		return nil, fmt.Errorf("error decoding getRoleMemberCount result: %w", err)
	}

	// For each address index in the roles count, get the address
	addresses := make([]string, 0, count.Uint64())
	for j := range count.Uint64() {
		rAddr, err := i.client.RunGetMethod(ctx, block, addr, "getRoleMember", _role, uint32(j))
		if err != nil {
			return nil, fmt.Errorf("error getting getRoleMember: %w", err)
		}

		sAddr, err := rAddr.Slice(0)
		if err != nil {
			return nil, fmt.Errorf("error decoding getRoleMember result: %w", err)
		}

		addr, err := sAddr.LoadAddr()
		if err != nil {
			return nil, fmt.Errorf("error decoding getRoleMember result slice: %w", err)
		}

		addresses = append(addresses, addr.String())
	}

	return addresses, nil
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

// Help function to map (encode) role param to cell.Slice
func mapRoleParam(role [32]byte) (*cell.Slice, error) {
	b := cell.BeginCell()
	if err := b.StoreBigUInt(new(big.Int).SetBytes(role[:]), 256); err != nil {
		return nil, fmt.Errorf("failed to store domain separator: %w", err)
	}

	return b.EndCell().BeginParse(), nil
}
