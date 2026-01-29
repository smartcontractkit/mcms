package ton

import (
	"context"
	"fmt"
	"math/big"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/lib/access/rbac"
	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/timelock"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"

	"github.com/smartcontractkit/mcms/sdk"
)

var _ sdk.TimelockInspector = (*TimelockInspector)(nil)

// TimelockInspector is an Inspector implementation for TON, for accessing the MCMS-Timelock contract
type TimelockInspector struct {
	client ton.APIClientWrapped
}

func NewTimelockInspector(client ton.APIClientWrapped) sdk.TimelockInspector {
	return &TimelockInspector{client}
}

func (i TimelockInspector) GetMinDelay(ctx context.Context, _address string) (uint64, error) {
	addr, err := address.ParseAddr(_address)
	if err != nil {
		return 0, fmt.Errorf("invalid address: %w", err)
	}

	return tvm.CallGetterLatest(ctx, i.client, addr, timelock.GetMinDelay)
}

// GetAdmins returns the list of addresses with the admin role
func (i TimelockInspector) GetAdmins(ctx context.Context, addr string) ([]string, error) {
	return i.getRoleMembers(ctx, addr, [32]byte(timelock.RoleAdmin.Bytes()))
}

// GetProposers returns the list of addresses with the proposer role
func (i TimelockInspector) GetProposers(ctx context.Context, addr string) ([]string, error) {
	return i.getRoleMembers(ctx, addr, [32]byte(timelock.RoleProposer.Bytes()))
}

// GetExecutors returns the list of addresses with the executor role
func (i TimelockInspector) GetExecutors(ctx context.Context, addr string) ([]string, error) {
	return i.getRoleMembers(ctx, addr, [32]byte(timelock.RoleExecutor.Bytes()))
}

// GetBypassers returns the list of addresses with the bypasser role
func (i TimelockInspector) GetBypassers(ctx context.Context, addr string) ([]string, error) {
	return i.getRoleMembers(ctx, addr, [32]byte(timelock.RoleBypasser.Bytes()))
}

// GetCancellers returns the list of addresses with the canceller role
func (i TimelockInspector) GetCancellers(ctx context.Context, addr string) ([]string, error) {
	return i.getRoleMembers(ctx, addr, [32]byte(timelock.RoleCanceller.Bytes()))
}

// GetOracles returns the list of addresses with the oracle role
func (i TimelockInspector) GetOracles(ctx context.Context, addr string) ([]string, error) {
	return i.getRoleMembers(ctx, addr, [32]byte(timelock.RoleOracle.Bytes()))
}

// getRoleMembers returns the list of addresses with the given role
func (i TimelockInspector) getRoleMembers(ctx context.Context, _address string, role [32]byte) ([]string, error) {
	// Map to Ton Address type (timelock.address)
	addr, err := address.ParseAddr(_address)
	if err != nil {
		return nil, fmt.Errorf("invalid timelock address: %w", err)
	}

	_role := new(big.Int).SetBytes(role[:])
	_addresses, err := rbac.GetRoleMembersView(ctx, i.client, addr, _role)
	if err != nil {
		return nil, fmt.Errorf("error calling GetRoleMembersView: %w", err)
	}

	n := len(_addresses)
	addresses := make([]string, n)
	for j := range n {
		addresses[j] = _addresses[j].String()
	}

	return addresses, nil
}

func (i TimelockInspector) IsOperation(ctx context.Context, _address string, opID [32]byte) (bool, error) {
	return i.isOperationFor(ctx, _address, opID, timelock.IsOperation)
}

func (i TimelockInspector) IsOperationPending(ctx context.Context, _address string, opID [32]byte) (bool, error) {
	return i.isOperationFor(ctx, _address, opID, timelock.IsOperationPending)
}

func (i TimelockInspector) IsOperationReady(ctx context.Context, _address string, opID [32]byte) (bool, error) {
	return i.isOperationFor(ctx, _address, opID, timelock.IsOperationReady)
}

func (i TimelockInspector) IsOperationDone(ctx context.Context, _address string, opID [32]byte) (bool, error) {
	return i.isOperationFor(ctx, _address, opID, timelock.IsOperationDone)
}

func (i TimelockInspector) IsOperationError(ctx context.Context, _address string, opID [32]byte) (bool, error) {
	return i.isOperationFor(ctx, _address, opID, timelock.IsOperationError)
}

// isOperationFor is a helper function to check the status of an operation using the provided getter
func (i TimelockInspector) isOperationFor(
	ctx context.Context,
	_address string,
	opID [32]byte,
	getter tvm.Getter[*big.Int, bool],
) (bool, error) {
	addr, err := address.ParseAddr(_address)
	if err != nil {
		return false, fmt.Errorf("invalid address: %w", err)
	}

	_opID := new(big.Int).SetBytes(opID[:])

	return tvm.CallGetterLatest(ctx, i.client, addr, getter, _opID)
}
