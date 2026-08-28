package stellar

import (
	"context"
	"fmt"

	chainselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-stellar/bindings"
	mcmsbindings "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/mcms"

	"github.com/smartcontractkit/mcms/types"
)

// ContractDeployer is the minimal Stellar deployment surface required by the
// MCMS SDK.
//
// *chainlink-stellar/deployment.Deployer satisfies this interface, allowing
// chainlink/deployment to construct an SDK Deployer without coupling the MCMS
// SDK to the concrete chainlink-stellar deployment package.
type ContractDeployer interface {
	bindings.Invoker

	DeployContract(
		ctx context.Context,
		wasmPath string,
		salt [32]byte,
	) (string, error)
}

// Deployer provides the low-level Stellar deployment and initialization
// operations for MCMS and Timelock contracts.
//
// Deployment orchestration such as salt derivation, collision handling,
// adoption, datastore updates, qualifiers, and versioning belongs to the
// calling deployment framework.
type Deployer struct {
	deployer ContractDeployer
}

// NewDeployer creates a Stellar MCMS deployer backed by a contract deployer
// that can both deploy Soroban WASM and invoke deployed contracts.
func NewDeployer(deployer ContractDeployer) *Deployer {
	return &Deployer{
		deployer: deployer,
	}
}

// InitializeMCMSInput contains the arguments required to initialize a newly
// deployed Stellar MCMS contract.
type InitializeMCMSInput struct {
	ContractID    string
	Owner         string
	ChainSelector types.ChainSelector
	Config        *types.Config
	InstanceLabel string
}

// DeployMCMS deploys the Stellar MCMS WASM using the supplied salt.
//
// Salt derivation and existing-contract/adoption handling intentionally remain
// outside the SDK so callers can define their own deployment lifecycle.
func (d *Deployer) DeployMCMS(
	ctx context.Context,
	wasmPath string,
	salt [32]byte,
) (string, error) {
	if d == nil || d.deployer == nil {
		return "", fmt.Errorf("stellar MCMS deployer is nil")
	}
	if wasmPath == "" {
		return "", fmt.Errorf("stellar MCMS WASM path is empty")
	}

	contractID, err := d.deployer.DeployContract(
		ctx,
		wasmPath,
		salt,
	)
	if err != nil {
		return "", fmt.Errorf("deploy Stellar MCMS: %w", err)
	}
	if contractID == "" {
		return "", fmt.Errorf("deploy Stellar MCMS: empty contract ID")
	}

	return contractID, nil
}

// InitializeMCMS initializes a newly deployed Stellar MCMS contract.
//
// It owns the Stellar-specific conversion from the generic MCMS signer
// configuration into the generated Soroban binding types.
func (d *Deployer) InitializeMCMS(
	ctx context.Context,
	in InitializeMCMSInput,
) (types.TransactionResult, error) {
	if d == nil || d.deployer == nil {
		return types.TransactionResult{},
			fmt.Errorf("stellar MCMS deployer is nil")
	}
	if in.ContractID == "" {
		return types.TransactionResult{},
			fmt.Errorf("stellar MCMS contract ID is empty")
	}
	if in.Owner == "" {
		return types.TransactionResult{},
			fmt.Errorf("stellar MCMS owner is empty")
	}
	if in.Config == nil {
		return types.TransactionResult{},
			fmt.Errorf("stellar MCMS config is nil")
	}
	if in.InstanceLabel == "" {
		return types.TransactionResult{},
			fmt.Errorf("stellar MCMS instance label is empty")
	}

	if err := in.Config.Validate(); err != nil {
		return types.TransactionResult{},
			fmt.Errorf("validate Stellar MCMS config: %w", err)
	}

	networkID, err := chainNetworkID(in.ChainSelector)
	if err != nil {
		return types.TransactionResult{},
			fmt.Errorf("get Stellar chain network ID: %w", err)
	}

	signerAddresses,
		signerGroups,
		groupQuorums,
		groupParents,
		err := ConfigToSetConfigInputs(in.Config)
	if err != nil {
		return types.TransactionResult{},
			fmt.Errorf("convert Stellar MCMS config: %w", err)
	}

	client := mcmsbindings.NewMcmsClient(
		d.deployer,
		in.ContractID,
	)

	if err := client.Initialize(
		ctx,
		in.Owner,
		[32]byte(networkID),
		signerAddresses,
		signerGroups,
		groupQuorums,
		groupParents,
		in.InstanceLabel,
	); err != nil {
		return types.TransactionResult{},
			fmt.Errorf(
				"initialize Stellar MCMS %s: %w",
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
