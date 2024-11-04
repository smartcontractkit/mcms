package evm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/internal/utils/safecast"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	sdk_timelock "github.com/smartcontractkit/mcms/sdk/timelock"
)

// TimelockEVMInspector is an Inspector implementation for EVM chains for accessing the RBACTimelock contract
type TimelockEVMInspector struct {
	client evm.ContractDeployBackend
}

var _ sdk_timelock.Inspector = &TimelockEVMInspector{}

// NewTimelockEVMInspector creates a new TimelockEVMInspector
func NewTimelockEVMInspector(client evm.ContractDeployBackend) *TimelockEVMInspector {
	return &TimelockEVMInspector{
		client: client,
	}
}

// getAddressesWithRole returns the list of addresses with the given role
func (tm TimelockEVMInspector) getAddressesWithRole(timelock *bindings.RBACTimelock, role [32]byte) ([]common.Address, error) {
	numAddresses, err := timelock.GetRoleMemberCount(&bind.CallOpts{}, role)
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
		address, err := timelock.GetRoleMember(&bind.CallOpts{}, role, big.NewInt(idx))
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, address)
	}

	return addresses, nil
}

// GetProposers returns the list of addresses with the proposer role
func (tm TimelockEVMInspector) GetProposers(address string) ([]common.Address, error) {
	timelock, err := bindings.NewRBACTimelock(common.HexToAddress(address), tm.client)
	if err != nil {
		return nil, err
	}
	proposerRole, err := timelock.PROPOSERROLE(nil)
	if err != nil {
		return nil, err
	}

	return tm.getAddressesWithRole(timelock, proposerRole)
}

// GetExecutors returns the list of addresses with the executor role
func (tm TimelockEVMInspector) GetExecutors(address string) ([]common.Address, error) {
	timelock, err := bindings.NewRBACTimelock(common.HexToAddress(address), tm.client)
	if err != nil {
		return nil, err
	}
	proposerRole, err := timelock.EXECUTORROLE(nil)
	if err != nil {
		return nil, err
	}

	return tm.getAddressesWithRole(timelock, proposerRole)
}

// GetBypassers returns the list of addresses with the bypasser role
func (tm TimelockEVMInspector) GetBypassers(address string) ([]common.Address, error) {
	timelock, err := bindings.NewRBACTimelock(common.HexToAddress(address), tm.client)
	if err != nil {
		return nil, err
	}
	proposerRole, err := timelock.BYPASSERROLE(nil)
	if err != nil {
		return nil, err
	}

	return tm.getAddressesWithRole(timelock, proposerRole)
}

// GetCancellers returns the list of addresses with the canceller role
func (tm TimelockEVMInspector) GetCancellers(address string) ([]common.Address, error) {
	timelock, err := bindings.NewRBACTimelock(common.HexToAddress(address), tm.client)
	if err != nil {
		return nil, err
	}
	proposerRole, err := timelock.CANCELLERROLE(nil)
	if err != nil {
		return nil, err
	}

	return tm.getAddressesWithRole(timelock, proposerRole)
}

func (tm TimelockEVMInspector) IsOperation(address string, opId [32]byte) (bool, error) {
	timelock, err := bindings.NewRBACTimelock(common.HexToAddress(address), tm.client)
	if err != nil {
		return false, err
	}

	return timelock.IsOperation(&bind.CallOpts{}, opId)
}

func (tm TimelockEVMInspector) IsOperationPending(address string, opId [32]byte) (bool, error) {
	timelock, err := bindings.NewRBACTimelock(common.HexToAddress(address), tm.client)
	if err != nil {
		return false, err
	}

	return timelock.IsOperationPending(&bind.CallOpts{}, opId)
}

func (tm TimelockEVMInspector) IsOperationReady(address string, opId [32]byte) (bool, error) {
	timelock, err := bindings.NewRBACTimelock(common.HexToAddress(address), tm.client)
	if err != nil {
		return false, err
	}

	return timelock.IsOperationReady(&bind.CallOpts{}, opId)
}

func (tm TimelockEVMInspector) IsOperationDone(address string, opId [32]byte) (bool, error) {
	timelock, err := bindings.NewRBACTimelock(common.HexToAddress(address), tm.client)
	if err != nil {
		return false, err
	}

	return timelock.IsOperationDone(&bind.CallOpts{}, opId)
}
