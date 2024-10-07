package evm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/smartcontractkit/ccip-owner-contracts/gethwrappers"

	"github.com/smartcontractkit/mcms/pkg/contract"
)

// MCMSContractDeployment returns a DeployFunc that deploys the ManyChainMultiSig contract.
func MCMSContractDeployment(
	auth *bind.TransactOpts, backend bind.ContractBackend,
) contract.DeployFunc[common.Address, *types.Transaction] {
	return func() (common.Address, *types.Transaction, error) {
		addr, tx, _, err := gethwrappers.DeployManyChainMultiSig(auth, backend)

		return addr, tx, err
	}
}

// CallProxyContractDeployment returns a DeployFunc that deploys the CallProxy contract.
func CallProxyContractDeployment(
	auth *bind.TransactOpts, backend bind.ContractBackend, target common.Address,
) contract.DeployFunc[common.Address, *types.Transaction] {
	return func() (common.Address, *types.Transaction, error) {
		addr, tx, _, err := gethwrappers.DeployCallProxy(auth, backend, target)

		return addr, tx, err
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
) contract.DeployFunc[common.Address, *types.Transaction] {
	return func() (common.Address, *types.Transaction, error) {
		addr, tx, _, err := gethwrappers.DeployRBACTimelock(
			auth, backend, minDelay, admin, proposers, executors, cancellers, bypassers,
		)

		return addr, tx, err
	}
}
