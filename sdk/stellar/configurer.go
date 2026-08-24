package stellar

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-stellar/bindings"
	mcmsbindings "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/mcms"
)

var _ sdk.Configurer = (*Configurer)(nil)

// Configurer implements sdk.Configurer for the Stellar (Soroban) MCMS contract.
type Configurer struct {
	invoker bindings.Invoker
}

// NewConfigurer creates a new Configurer backed by the given Soroban invoker.
func NewConfigurer(invoker bindings.Invoker) *Configurer {
	return &Configurer{invoker: invoker}
}

// Initialize initializes a newly deployed MCMS contract using the current
// Stellar contract ABI. Deployment of the WASM remains the responsibility of
// the caller; this method only submits the contract initialization call.
func (c *Configurer) Initialize(
	ctx context.Context,
	mcmAddr string,
	owner string,
	networkID [32]byte,
	cfg *types.Config,
	instanceLabel string,
) (types.TransactionResult, error) {
	if c == nil || c.invoker == nil {
		return types.TransactionResult{}, fmt.Errorf("stellar mcms initialize: invoker is nil")
	}
	if owner == "" {
		return types.TransactionResult{}, fmt.Errorf("stellar mcms initialize: owner is empty")
	}
	if instanceLabel == "" {
		return types.TransactionResult{}, fmt.Errorf("stellar mcms initialize: instance label is empty")
	}

	signerAddresses, signerGroups, groupQuorums, groupParents, err := stellarSetConfigInputs(cfg)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("stellar mcms initialize: %w", err)
	}

	client := mcmsbindings.NewMcmsClient(c.invoker, mcmAddr)
	if err := client.Initialize(ctx, owner, networkID, signerAddresses, signerGroups, groupQuorums, groupParents, instanceLabel); err != nil {
		return types.TransactionResult{}, fmt.Errorf("stellar mcms initialize: %w", err)
	}

	return types.NewTransactionResult("", nil, "stellar"), nil
}

// InitializeInputs submits initialize using already-converted binding values.
// It supports deployment operations that store contract-shaped inputs.
func (c *Configurer) InitializeInputs(
	ctx context.Context,
	mcmAddr, owner string,
	networkID [32]byte,
	signerAddresses mcmsbindings.SignerAddresses,
	signerGroups mcmsbindings.SignerGroups,
	groupQuorums [32]byte,
	groupParents [32]byte,
	instanceLabel string,
) (types.TransactionResult, error) {
	if c == nil || c.invoker == nil {
		return types.TransactionResult{}, fmt.Errorf("stellar mcms initialize: invoker is nil")
	}
	if owner == "" || instanceLabel == "" {
		return types.TransactionResult{}, fmt.Errorf("stellar mcms initialize: owner and instance label are required")
	}
	client := mcmsbindings.NewMcmsClient(c.invoker, mcmAddr)
	if err := client.Initialize(ctx, owner, networkID, signerAddresses, signerGroups, groupQuorums, groupParents, instanceLabel); err != nil {
		return types.TransactionResult{}, fmt.Errorf("stellar mcms initialize: %w", err)
	}

	return types.NewTransactionResult("", nil, "stellar"), nil
}

// SetConfig applies a new signer configuration to the MCMS contract on Stellar.
func (c *Configurer) SetConfig(
	ctx context.Context,
	mcmAddr string,
	cfg *types.Config,
	clearRoot bool,
) (types.TransactionResult, error) {
	if c == nil || c.invoker == nil {
		return types.TransactionResult{}, fmt.Errorf("stellar mcms set_config: invoker is nil")
	}
	signerAddresses, signerGroups, groupQuorums, groupParents, err := stellarSetConfigInputs(cfg)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("extract set_config inputs: %w", err)
	}

	client := mcmsbindings.NewMcmsClient(c.invoker, mcmAddr)
	if err = client.SetConfig(ctx, signerAddresses, signerGroups, groupQuorums, groupParents, clearRoot); err != nil {
		return types.TransactionResult{}, fmt.Errorf("stellar mcms set_config: %w", err)
	}

	return types.NewTransactionResult("", nil, "stellar"), nil
}

// SetConfigInputs submits set_config using already-converted current-binding
// values. It is useful to deployment adapters that receive generated binding
// values from an existing configuration pipeline.
func (c *Configurer) SetConfigInputs(
	ctx context.Context,
	mcmAddr string,
	signerAddresses mcmsbindings.SignerAddresses,
	signerGroups mcmsbindings.SignerGroups,
	groupQuorums [32]byte,
	groupParents [32]byte,
	clearRoot bool,
) (types.TransactionResult, error) {
	if c == nil || c.invoker == nil {
		return types.TransactionResult{}, fmt.Errorf("stellar mcms set_config: invoker is nil")
	}
	client := mcmsbindings.NewMcmsClient(c.invoker, mcmAddr)
	if err := client.SetConfig(ctx, signerAddresses, signerGroups, groupQuorums, groupParents, clearRoot); err != nil {
		return types.TransactionResult{}, fmt.Errorf("stellar mcms set_config: %w", err)
	}

	return types.NewTransactionResult("", nil, "stellar"), nil
}

func stellarSetConfigInputs(cfg *types.Config) (
	mcmsbindings.SignerAddresses,
	mcmsbindings.SignerGroups,
	[32]byte,
	[32]byte,
	error,
) {
	if cfg == nil {
		return mcmsbindings.SignerAddresses{}, mcmsbindings.SignerGroups{}, [32]byte{}, [32]byte{}, nil
	}
	groupQuorums, groupParents, signerAddresses, signerGroups, err := sdk.ExtractSetConfigInputs(cfg)
	if err != nil {
		return mcmsbindings.SignerAddresses{}, mcmsbindings.SignerGroups{}, [32]byte{}, [32]byte{}, err
	}

	signerAddrsSoroban := mcmsbindings.SignerAddresses{
		Inner: make([][32]byte, len(signerAddresses)),
	}
	for i, addr := range signerAddresses {
		var a [32]byte
		copy(a[12:], addr.Bytes()) // 20-byte EVM address padded to 32 bytes
		signerAddrsSoroban.Inner[i] = a
	}

	signerGroupsSoroban := mcmsbindings.SignerGroups{
		Inner: make([]uint32, len(signerGroups)),
	}
	for i, g := range signerGroups {
		signerGroupsSoroban.Inner[i] = uint32(g)
	}

	var gq [32]byte
	copy(gq[:], groupQuorums[:])
	var gp [32]byte
	copy(gp[:], groupParents[:])

	return signerAddrsSoroban, signerGroupsSoroban, gq, gp, nil
}

// ConfigToSetConfigInputs converts the generic MCMS configuration to the
// current Stellar contract values. It is exported for deployment adapters
// that must pass binding values through a framework operation.
func ConfigToSetConfigInputs(cfg *types.Config) (
	mcmsbindings.SignerAddresses,
	mcmsbindings.SignerGroups,
	[32]byte,
	[32]byte,
	error,
) {
	return stellarSetConfigInputs(cfg)
}

// TransferOwnership starts the MCMS contract's two-step ownership transfer.
// The transaction must be authorized by the current owner. The new owner must
// subsequently call AcceptOwnership from its own account.
func (c *Configurer) TransferOwnership(
	ctx context.Context, mcmAddr string, newOwner string,
) (types.TransactionResult, error) {
	if c == nil || c.invoker == nil {
		return types.TransactionResult{}, fmt.Errorf("stellar mcms transfer_ownership: invoker is nil")
	}
	if newOwner == "" {
		return types.TransactionResult{}, fmt.Errorf("stellar mcms transfer_ownership: new owner is empty")
	}

	client := mcmsbindings.NewMcmsClient(c.invoker, mcmAddr)
	if err := client.TransferOwnership(ctx, newOwner); err != nil {
		return types.TransactionResult{}, fmt.Errorf("stellar mcms transfer_ownership: %w", err)
	}

	return types.NewTransactionResult("", nil, "stellar"), nil
}

// AcceptOwnership completes a pending MCMS ownership transfer. The invoker
// must submit this transaction as the pending owner.
func (c *Configurer) AcceptOwnership(
	ctx context.Context, mcmAddr string,
) (types.TransactionResult, error) {
	if c == nil || c.invoker == nil {
		return types.TransactionResult{}, fmt.Errorf("stellar mcms accept_ownership: invoker is nil")
	}

	client := mcmsbindings.NewMcmsClient(c.invoker, mcmAddr)
	if err := client.AcceptOwnership(ctx); err != nil {
		return types.TransactionResult{}, fmt.Errorf("stellar mcms accept_ownership: %w", err)
	}

	return types.NewTransactionResult("", nil, "stellar"), nil
}

// CancelOwnershipTransfer cancels the currently pending ownership transfer.
func (c *Configurer) CancelOwnershipTransfer(
	ctx context.Context, mcmAddr string,
) (types.TransactionResult, error) {
	if c == nil || c.invoker == nil {
		return types.TransactionResult{}, fmt.Errorf("stellar mcms cancel_ownership_transfer: invoker is nil")
	}

	client := mcmsbindings.NewMcmsClient(c.invoker, mcmAddr)
	if err := client.CancelOwnershipTransfer(ctx); err != nil {
		return types.TransactionResult{}, fmt.Errorf("stellar mcms cancel_ownership_transfer: %w", err)
	}

	return types.NewTransactionResult("", nil, "stellar"), nil
}
