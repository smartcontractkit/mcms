package stellar

import (
	"context"
	"fmt"

	chainselectors "github.com/smartcontractkit/chain-selectors"

	timelockbindings "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/timelock"

	"github.com/smartcontractkit/mcms/types"
)

// TimelockDeployer provides the low-level Stellar deployment and
// initialization operations for the MCMS timelock contract.
type TimelockDeployer struct {
	deployer ContractDeployer
}

// NewTimelockDeployer creates a Stellar timelock deployer backed by a contract
// deployer that can both deploy Soroban WASM and invoke deployed contracts.
func NewTimelockDeployer(deployer ContractDeployer) *TimelockDeployer {
	return &TimelockDeployer{
		deployer: deployer,
	}
}

// InitializeTimelockInput contains the arguments required to initialize a
// newly deployed Stellar timelock contract.
type InitializeTimelockInput struct {
	ContractID string
	MinDelay   uint64
	Proposers  []string
	Cancellers []string
	Bypassers  []string
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
		return types.TransactionResult{}, fmt.Errorf("stellar timelock deployer is nil")
	}
	if in.ContractID == "" {
		return types.TransactionResult{}, fmt.Errorf("stellar timelock contract ID is empty")
	}
	if err := validateTimelockRoleAddresses("proposers", in.Proposers); err != nil {
		return types.TransactionResult{}, err
	}
	if err := validateTimelockRoleAddresses("cancellers", in.Cancellers); err != nil {
		return types.TransactionResult{}, err
	}
	if err := validateTimelockRoleAddresses("bypassers", in.Bypassers); err != nil {
		return types.TransactionResult{}, err
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
		return types.TransactionResult{}, fmt.Errorf(
			"initialize Stellar timelock %s: %w",
			in.ContractID,
			err,
		)
	}

	return types.NewTransactionResult(
		"",
		nil,
		chainselectors.FamilyStellar,
	), nil
}

func validateTimelockRoleAddresses(role string, addresses []string) error {
	if len(addresses) == 0 {
		return fmt.Errorf("initialize Stellar timelock: %s are empty", role)
	}

	seen := make(map[string]struct{}, len(addresses))
	for i, address := range addresses {
		if address == "" {
			return fmt.Errorf("initialize Stellar timelock: %s[%d] is empty", role, i)
		}
		if _, exists := seen[address]; exists {
			return fmt.Errorf(
				"initialize Stellar timelock: %s contains duplicate address %q",
				role,
				address,
			)
		}
		seen[address] = struct{}{}
	}

	return nil
}
