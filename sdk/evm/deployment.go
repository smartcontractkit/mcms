package evm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/smartcontractkit/ccip-owner-contracts/gethwrappers"

	"github.com/smartcontractkit/mcms/pkg/contract"
)

// DeployResult represents a deployed contract with its address, transaction data, and go-ethereum
// binding.
type DeployResult[B any] struct {
	Address  common.Address
	Tx       *types.Transaction
	Contract B
}

// newDeployResult creates a new DeployResult instance with the provided address, transaction, and
// contract binding.
func newDeployResult[B any](
	addr common.Address, tx *types.Transaction, binding B,
) DeployResult[B] {
	return DeployResult[B]{
		Address:  addr,
		Tx:       tx,
		Contract: binding,
	}
}

// MCMSContractDeployment returns a DeployFunc that deploys the ManyChainMultiSig contract.
func MCMSContractDeployment(
	auth *bind.TransactOpts, backend bind.ContractBackend,
) contract.DeployFunc[DeployResult[*gethwrappers.ManyChainMultiSig]] {
	return func() (DeployResult[*gethwrappers.ManyChainMultiSig], error) {
		addr, tx, binding, err := gethwrappers.DeployManyChainMultiSig(auth, backend)

		return newDeployResult(addr, tx, binding), err
	}
}

// CallProxyContractDeployment returns a DeployFunc that deploys the CallProxy contract.
func CallProxyContractDeployment(
	auth *bind.TransactOpts, backend bind.ContractBackend, target common.Address,
) contract.DeployFunc[DeployResult[*gethwrappers.CallProxy]] {
	return func() (DeployResult[*gethwrappers.CallProxy], error) {
		addr, tx, binding, err := gethwrappers.DeployCallProxy(auth, backend, target)

		return newDeployResult(addr, tx, binding), err
	}
}

// TimelockContractDeployment returns a DeployFunc that deploys the RBACTimelock contract.
func TimelockContractDeployment(
	auth *bind.TransactOpts,
	backend bind.ContractBackend,
	minDelay *big.Int,
	admin common.Address,
	proposers []common.Address,
	executors []common.Address,
	cancellers []common.Address,
	bypassers []common.Address,
) contract.DeployFunc[DeployResult[*gethwrappers.RBACTimelock]] {
	return func() (DeployResult[*gethwrappers.RBACTimelock], error) {
		addr, tx, binding, err := gethwrappers.DeployRBACTimelock(
			auth, backend, minDelay, admin, proposers, executors, cancellers, bypassers,
		)

		return newDeployResult(addr, tx, binding), err
	}
}
