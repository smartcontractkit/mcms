package evm

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/internal/utils/safecast"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
)

var _ sdk.TimelockInspector[common.Address] = (*TimelockInspector)(nil)

// TimelockInspector is an Inspector implementation for EVM chains for accessing the RBACTimelock contract
type TimelockInspector struct {
	client ContractDeployBackend
}

// NewTimelockInspector creates a new TimelockInspector
func NewTimelockInspector(client ContractDeployBackend) *TimelockInspector {
	return &TimelockInspector{
		client: client,
	}
}

// getAddressesWithRole returns the list of addresses with the given role
func (tm TimelockInspector) getAddressesWithRole(
	ctx context.Context, timelock *bindings.RBACTimelock, role [32]byte,
) ([]common.Address, error) {
	numAddresses, err := timelock.GetRoleMemberCount(&bind.CallOpts{Context: ctx}, role)
	if err != nil {
		return nil, err
	}
	// For each address index in the roles count, get the address
	addresses := make([]common.Address, 0, numAddresses.Uint64())
	for i := range numAddresses.Uint64() {
		idx, err := safecast.Uint64ToInt64(i)
		if err != nil {
			return nil, err
		}
		address, err := timelock.GetRoleMember(&bind.CallOpts{Context: ctx}, role, big.NewInt(idx))
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, address)
	}

	return addresses, nil
}

// GetProposers returns the list of addresses with the proposer role
func (tm TimelockInspector) GetProposers(ctx context.Context, address string) ([]common.Address, error) {
	timelock, err := bindings.NewRBACTimelock(common.HexToAddress(address), tm.client)
	if err != nil {
		return nil, err
	}
	proposerRole, err := timelock.PROPOSERROLE(nil)
	if err != nil {
		return nil, err
	}

	return tm.getAddressesWithRole(ctx, timelock, proposerRole)
}

// GetExecutors returns the list of addresses with the executor role
func (tm TimelockInspector) GetExecutors(ctx context.Context, address string) ([]common.Address, error) {
	timelock, err := bindings.NewRBACTimelock(common.HexToAddress(address), tm.client)
	if err != nil {
		return nil, err
	}
	executorRole, err := timelock.EXECUTORROLE(nil)
	if err != nil {
		return nil, err
	}

	return tm.getAddressesWithRole(ctx, timelock, executorRole)
}

// GetBypassers returns the list of addresses with the bypasser role
func (tm TimelockInspector) GetBypassers(ctx context.Context, address string) ([]common.Address, error) {
	timelock, err := bindings.NewRBACTimelock(common.HexToAddress(address), tm.client)
	if err != nil {
		return nil, err
	}
	bypasserRole, err := timelock.BYPASSERROLE(nil)
	if err != nil {
		return nil, err
	}

	return tm.getAddressesWithRole(ctx, timelock, bypasserRole)
}

// GetCancellers returns the list of addresses with the canceller role
func (tm TimelockInspector) GetCancellers(ctx context.Context, address string) ([]common.Address, error) {
	timelock, err := bindings.NewRBACTimelock(common.HexToAddress(address), tm.client)
	if err != nil {
		return nil, err
	}
	cancellerRole, err := timelock.CANCELLERROLE(nil)
	if err != nil {
		return nil, err
	}

	return tm.getAddressesWithRole(ctx, timelock, cancellerRole)
}

func (tm TimelockInspector) IsOperation(ctx context.Context, address string, opID [32]byte) (bool, error) {
	timelock, err := bindings.NewRBACTimelock(common.HexToAddress(address), tm.client)
	if err != nil {
		return false, err
	}

	return timelock.IsOperation(&bind.CallOpts{Context: ctx}, opID)
}

func (tm TimelockInspector) IsOperationPending(ctx context.Context, address string, opID [32]byte) (bool, error) {
	timelock, err := bindings.NewRBACTimelock(common.HexToAddress(address), tm.client)
	if err != nil {
		return false, err
	}

	return timelock.IsOperationPending(&bind.CallOpts{Context: ctx}, opID)
}

func (tm TimelockInspector) IsOperationReady(ctx context.Context, address string, opID [32]byte) (bool, error) {
	timelock, err := bindings.NewRBACTimelock(common.HexToAddress(address), tm.client)
	if err != nil {
		return false, err
	}

	return timelock.IsOperationReady(&bind.CallOpts{Context: ctx}, opID)
}

func (tm TimelockInspector) IsOperationDone(ctx context.Context, address string, opID [32]byte) (bool, error) {
	timelock, err := bindings.NewRBACTimelock(common.HexToAddress(address), tm.client)
	if err != nil {
		return false, err
	}

	return timelock.IsOperationDone(&bind.CallOpts{Context: ctx}, opID)
}
