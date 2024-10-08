package evm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/ccip-owner-contracts/gethwrappers"

	"github.com/smartcontractkit/mcms/pkg/contract"
)

// Example of implementing the Deployer interface

type BaseDeployment struct {
	Auth    *bind.TransactOpts
	Backend bind.ContractBackend
}

// MCMS

var _ contract.Deployer2[DeployResult[*gethwrappers.ManyChainMultiSig]] = MCMSContractDeployment2{}

type MCMSContractDeployment2 struct {
	BaseDeployment
}

func NewMCMSContractDeployment2(
	auth *bind.TransactOpts, backend bind.ContractBackend,
) MCMSContractDeployment2 {
	return MCMSContractDeployment2{
		BaseDeployment: BaseDeployment{
			Auth:    auth,
			Backend: backend,
		},
	}
}

func (d MCMSContractDeployment2) Deploy() (DeployResult[*gethwrappers.ManyChainMultiSig], error) {
	addr, tx, binding, err := gethwrappers.DeployManyChainMultiSig(d.Auth, d.Backend)

	return newDeployResult(addr, tx, binding), err
}

// Call Proxy

var _ contract.Deployer2[DeployResult[*gethwrappers.CallProxy]] = CallProxyContractDeployment2{}

type CallProxyContractDeployment2 struct {
	BaseDeployment

	target common.Address
}

func NewCallProxyContractDeployment2(
	auth *bind.TransactOpts,
	backend bind.ContractBackend,
	target common.Address,
) CallProxyContractDeployment2 {
	return CallProxyContractDeployment2{
		BaseDeployment: BaseDeployment{
			Auth:    auth,
			Backend: backend,
		},
		target: target,
	}
}

func (d CallProxyContractDeployment2) Deploy() (DeployResult[*gethwrappers.CallProxy], error) {
	addr, tx, binding, err := gethwrappers.DeployCallProxy(d.Auth, d.Backend, d.target)

	return newDeployResult(addr, tx, binding), err
}

// Timelock

var _ contract.Deployer2[DeployResult[*gethwrappers.RBACTimelock]] = TimelockContractDeployment2{}

type TimelockContractDeployment2 struct {
	BaseDeployment

	minDelay   *big.Int
	admin      common.Address
	proposers  []common.Address
	executors  []common.Address
	cancellers []common.Address
	bypassers  []common.Address
}

// TimelockContractDeployment returns a DeployFunc that deploys the RBACTimelock contract.
func NewTimelockContractDeployment2(
	auth *bind.TransactOpts,
	backend bind.ContractBackend,
	minDelay *big.Int,
	admin common.Address,
	proposers []common.Address,
	executors []common.Address,
	cancellers []common.Address,
	bypassers []common.Address,
) TimelockContractDeployment2 {
	return TimelockContractDeployment2{
		BaseDeployment: BaseDeployment{
			Auth:    auth,
			Backend: backend,
		},
		minDelay:   minDelay,
		admin:      admin,
		proposers:  proposers,
		executors:  executors,
		cancellers: cancellers,
		bypassers:  bypassers,
	}
}

func (d TimelockContractDeployment2) Deploy() (DeployResult[*gethwrappers.RBACTimelock], error) {
	addr, tx, binding, err := gethwrappers.DeployRBACTimelock(
		d.Auth, d.Backend, d.minDelay, d.admin, d.proposers, d.executors, d.cancellers, d.bypassers,
	)

	return newDeployResult(addr, tx, binding), err
}
