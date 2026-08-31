package stellar

import (
	"context"
	"fmt"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stellar/go-stellar-sdk/network"

	"github.com/smartcontractkit/chainlink-stellar/bindings"
	mcmsbindings "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/mcms"

	"github.com/smartcontractkit/mcms/types"
)

// ContractDeployer is the minimal Stellar deployment surface required by the
// MCMS SDK.
type ContractDeployer interface {
	bindings.Invoker

	DeployContract(ctx context.Context, wasmPath string, salt [32]byte) (string, error)
}

// Deployer provides the low-level Stellar deployment and initialization
// operations for MCMS contracts.
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
	return &Deployer{deployer: deployer}
}

// InitializeMCMSInput contains the arguments required to initialize a newly
// deployed Stellar MCMS contract.
type InitializeMCMSInput struct {
	ContractID    string
	Owner         string
	ChainID       string
	Config        *types.Config
	InstanceLabel string
}

// DeployMCMS deploys the Stellar MCMS WASM using the supplied salt.
//
// Salt derivation and existing-contract/adoption handling intentionally remain
// outside the SDK so callers can define their own deployment lifecycle.
func (d *Deployer) DeployMCMS(ctx context.Context, wasmPath string, salt [32]byte) (string, error) {
	if d == nil || d.deployer == nil {
		return "", fmt.Errorf("stellar MCMS deployer is nil")
	}

	if wasmPath == "" {
		return "", fmt.Errorf("stellar MCMS WASM path is empty")
	}

	contractID, err := d.deployer.DeployContract(ctx, wasmPath, salt)
	if err != nil {
		return "", fmt.Errorf("deploy stellar MCMS: %w", err)
	}

	if contractID == "" {
		return "", fmt.Errorf("deploy stellar MCMS: empty contract ID")
	}

	return contractID, nil
}

// InitializeMCMS initializes a newly deployed Stellar MCMS contract.
//
// It owns validation of the MCMS initialization inputs and the Stellar-specific
// conversion from the generic MCMS signer configuration into the generated
// Soroban binding types.
func (d *Deployer) InitializeMCMS(ctx context.Context, in InitializeMCMSInput) (types.TransactionResult, error) {
	if d == nil || d.deployer == nil {
		return types.TransactionResult{}, fmt.Errorf("stellar MCMS deployer is nil")
	}

	if in.ContractID == "" {
		return types.TransactionResult{}, fmt.Errorf("stellar MCMS contract ID is empty")
	}

	if in.Owner == "" {
		return types.TransactionResult{}, fmt.Errorf("stellar MCMS owner is empty")
	}

	if in.ChainID == "" {
		return types.TransactionResult{}, fmt.Errorf("stellar MCMS chain ID is empty")
	}

	if in.Config == nil {
		return types.TransactionResult{}, fmt.Errorf("stellar MCMS config is nil")
	}

	if in.InstanceLabel == "" {
		return types.TransactionResult{}, fmt.Errorf("stellar MCMS instance label is empty")
	}
	const maxInstanceLabelLength = 32
	if len(in.InstanceLabel) > maxInstanceLabelLength {
		return types.TransactionResult{}, fmt.Errorf("stellar MCMS instance label %q exceeds 32 bytes", in.InstanceLabel)
	}

	if err := in.Config.Validate(); err != nil {
		return types.TransactionResult{}, fmt.Errorf("validate stellar MCMS config: %w", err)
	}

	networkPassphrase, err := chainsel.StellarPassphraseFromChainId(in.ChainID)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("get stellar network passphrase from chain ID %q: %w", in.ChainID, err)
	}

	networkID := network.ID(networkPassphrase)

	signerAddresses, signerGroups, groupQuorums, groupParents, err := ConfigToSetConfigInputs(in.Config)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("convert stellar MCMS config: %w", err)
	}

	client := mcmsbindings.NewMcmsClient(d.deployer, in.ContractID)

	if err := client.Initialize(ctx, in.Owner, networkID, signerAddresses, signerGroups, groupQuorums, groupParents, in.InstanceLabel); err != nil {
		return types.TransactionResult{}, fmt.Errorf("initialize stellar MCMS %s: %w", in.ContractID, err)
	}

	return types.NewTransactionResult("", nil, chainsel.FamilyStellar), nil
}
