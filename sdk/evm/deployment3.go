package evm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/ccip-owner-contracts/gethwrappers"

	"github.com/smartcontractkit/mcms/pkg/contract"
)

// This gives an example of how to define the DeployFunc within a struct sharing
// the same geth contract backend and auth.

type ContractDeployer struct {
	auth    *bind.TransactOpts
	backend bind.ContractBackend
}

func (d ContractDeployer) MCMSDeployFunc() contract.DeployFunc[DeployResult[*gethwrappers.ManyChainMultiSig]] {
	return func() (DeployResult[*gethwrappers.ManyChainMultiSig], error) {
		addr, tx, binding, err := gethwrappers.DeployManyChainMultiSig(d.auth, d.backend)

		return newDeployResult(addr, tx, binding), err
	}
}

func (d ContractDeployer) CallProxyDeployFunc(target common.Address) contract.DeployFunc[DeployResult[*gethwrappers.CallProxy]] {
	return func() (DeployResult[*gethwrappers.CallProxy], error) {
		addr, tx, binding, err := gethwrappers.DeployCallProxy(d.auth, d.backend, target)

		return newDeployResult(addr, tx, binding), err
	}
}

func (d ContractDeployer) TimelockDeployFunc(
	minDelay *big.Int,
	admin common.Address,
	proposers []common.Address,
	executors []common.Address,
	cancellers []common.Address,
	bypassers []common.Address,
) contract.DeployFunc[DeployResult[*gethwrappers.RBACTimelock]] {
	return func() (DeployResult[*gethwrappers.RBACTimelock], error) {
		addr, tx, binding, err := gethwrappers.DeployRBACTimelock(
			d.auth, d.backend, minDelay, admin, proposers, executors, cancellers, bypassers,
		)

		return newDeployResult(addr, tx, binding), err
	}
}
