package stellar

import (
	"context"
	"fmt"

	timelockbindings "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/timelock"

	"github.com/smartcontractkit/mcms/types"
)

type TimelockDeployer struct {
	deployer ContractDeployer
}

type InitializeTimelockInput struct {
	ContractID string
	MinDelay   uint64
	Proposers  []string
	Cancellers []string
	Bypassers  []string
}

func NewTimelockDeployer(deployer ContractDeployer) *TimelockDeployer {
	return &TimelockDeployer{
		deployer: deployer,
	}
}

// DeployTimelock deploys the Stellar timelock WASM using the supplied salt.
//
// Salt derivation, deterministic address selection, collision handling, and
// adoption of existing deployments are intentionally left to the caller.
func (d *TimelockDeployer) DeployTimelock(
	ctx context.Context,
	wasmPath string,
	salt [32]byte,
) (string, error) {
	if d == nil || d.deployer == nil {
		return "", fmt.Errorf("stellar timelock deployer is nil")
	}
	if wasmPath == "" {
		return "", fmt.Errorf("stellar timelock WASM path is empty")
	}

	contractID, err := d.deployer.DeployContract(
		ctx,
		wasmPath,
		salt,
	)
	if err != nil {
		return "", fmt.Errorf("deploy Stellar timelock: %w", err)
	}
	if contractID == "" {
		return "", fmt.Errorf("deploy Stellar timelock: empty contract ID")
	}

	return contractID, nil
}

// InitializeTimelock initializes a newly deployed Stellar timelock.
func (d *TimelockDeployer) InitializeTimelock(
	ctx context.Context,
	in InitializeTimelockInput,
) (types.TransactionResult, error) {
	if d == nil || d.deployer == nil {
		return types.TransactionResult{},
			fmt.Errorf("stellar timelock deployer is nil")
	}
	if in.ContractID == "" {
		return types.TransactionResult{},
			fmt.Errorf("stellar timelock contract ID is empty")
	}

	client := timelockbindings.NewTimelockClient(
		d.deployer,
		in.ContractID,
	)

	if err := client.Initialize(
		ctx,
		in.MinDelay,
		in.Proposers,
		in.Cancellers,
		in.Bypassers,
	); err != nil {
		return types.TransactionResult{},
			fmt.Errorf(
				"initialize Stellar timelock %s: %w",
				in.ContractID,
				err,
			)
	}

	return types.NewTransactionResult(
		"",
		nil,
		"stellar",
	), nil
}
