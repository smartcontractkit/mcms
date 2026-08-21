package stellar

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	stellarrpc "github.com/stellar/go-stellar-sdk/clients/rpcclient"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-stellar/bindings"
	mcmsbindings "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/mcms"
)

var _ sdk.Inspector = (*Inspector)(nil)

// Inspector implements sdk.Inspector for the Stellar (Soroban) MCMS contract.
// It uses a bindings.Invoker to call the generated McmsClient, which wraps
// the Soroban RPC for simulation reads.
type Inspector struct {
	ConfigTransformer
	invoker bindings.Invoker
}

// NewInspector creates a new Inspector backed by the given Soroban invoker.
// The invoker can be a *deployment.Deployer or any other bindings.Invoker.
func NewInspector(client *stellarrpc.Client, auth Signer, selector uint64) (*Inspector, error) {
	invoker, err := NewInvoker(client, auth, selector)
	if err != nil {
		return nil, err
	}

	return NewInspectorFromInvoker(invoker), nil
}

// NewInspectorWithNetworkPassphrase creates an inspector with an explicit
// network passphrase supplied by the deployment framework.
func NewInspectorWithNetworkPassphrase(client *stellarrpc.Client, auth Signer, passphrase string) (*Inspector, error) {
	invoker, err := NewInvokerWithNetworkPassphrase(client, auth, passphrase)
	if err != nil {
		return nil, err
	}

	return NewInspectorFromInvoker(invoker), nil
}

// NewInspectorFromInvoker creates an inspector for deployment code that
// already owns a bindings.Invoker.
func NewInspectorFromInvoker(invoker bindings.Invoker) *Inspector {
	return &Inspector{
		ConfigTransformer: ConfigTransformer{},
		invoker:           invoker,
	}
}

// GetConfig reads the current MCMS signer/group config from the chain.
func (i *Inspector) GetConfig(ctx context.Context, mcmAddr string) (*types.Config, error) {
	client := mcmsbindings.NewMcmsClient(i.invoker, mcmAddr)
	cfg, err := client.GetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("stellar mcms get_config: %w", err)
	}

	converted, err := i.ToConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("stellar mcms convert config: %w", err)
	}

	return converted, nil
}

// GetOpCount reads the current op counter from the MCMS contract.
func (i *Inspector) GetOpCount(ctx context.Context, mcmAddr string) (uint64, error) {
	client := mcmsbindings.NewMcmsClient(i.invoker, mcmAddr)
	count, err := client.GetOpCount(ctx)
	if err != nil {
		return 0, fmt.Errorf("stellar mcms get_op_count: %w", err)
	}

	return count, nil
}

// GetRoot reads the current Merkle root and validUntil from the MCMS contract.
func (i *Inspector) GetRoot(ctx context.Context, mcmAddr string) (common.Hash, uint32, error) {
	client := mcmsbindings.NewMcmsClient(i.invoker, mcmAddr)
	root, validUntil, err := client.GetRoot(ctx)
	if err != nil {
		return common.Hash{}, 0, fmt.Errorf("stellar mcms get_root: %w", err)
	}

	return root, validUntil, nil
}

// GetRootMetadata reads the chain metadata from the MCMS contract's root_metadata.
func (i *Inspector) GetRootMetadata(ctx context.Context, mcmAddr string) (types.ChainMetadata, error) {
	client := mcmsbindings.NewMcmsClient(i.invoker, mcmAddr)
	metadata, err := client.GetRootMetadata(ctx)
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("stellar mcms get_root_metadata: %w", err)
	}

	return types.ChainMetadata{
		StartingOpCount: metadata.PreOpCount,
		MCMAddress:      metadata.Multisig,
		AdditionalFields: func() json.RawMessage {
			b, _ := json.Marshal(struct {
				ConfigVersion   uint64 `json:"configVersion"`
				EncodingVersion uint32 `json:"encodingVersion"`
			}{metadata.ConfigVersion, metadata.EncodingVersion})

			return b
		}(),
	}, nil
}

// GetOwner reads the current MCMS owner. A nil owner means the contract has
// no owner, as defined by the common Ownable implementation.
func (i *Inspector) GetOwner(ctx context.Context, mcmAddr string) (*string, error) {
	client := mcmsbindings.NewMcmsClient(i.invoker, mcmAddr)
	owner, err := client.Owner(ctx)
	if err != nil {
		return nil, fmt.Errorf("stellar mcms owner: %w", err)
	}

	return owner, nil
}

// GetPendingOwner reads the address that has been proposed as the next owner.
func (i *Inspector) GetPendingOwner(ctx context.Context, mcmAddr string) (*string, error) {
	client := mcmsbindings.NewMcmsClient(i.invoker, mcmAddr)
	pending, err := client.GetPendingOwner(ctx)
	if err != nil {
		return nil, fmt.Errorf("stellar mcms get_pending_owner: %w", err)
	}

	return pending, nil
}
