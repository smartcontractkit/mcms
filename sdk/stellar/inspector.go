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
	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"
)

var _ sdk.Inspector = (*Inspector)(nil)

type Inspector struct {
	ConfigTransformer
	invoker bindings.Invoker
}

func NewInspector(client *stellarrpc.Client, auth bindings.Signer, selector uint64) (*Inspector, error) {
	invoker, err := NewInvoker(client, auth, selector)
	if err != nil {
		return nil, err
	}

	return NewInspectorFromInvoker(invoker), nil
}

func NewInspectorWithNetworkPassphrase(client *stellarrpc.Client, auth bindings.Signer, passphrase string) (*Inspector, error) {
	invoker, err := NewInvokerWithNetworkPassphrase(client, auth, passphrase)
	if err != nil {
		return nil, err
	}

	return NewInspectorFromInvoker(invoker), nil
}

func NewInspectorFromInvoker(invoker bindings.Invoker) *Inspector {
	return &Inspector{
		ConfigTransformer: ConfigTransformer{},
		invoker:           invoker,
	}
}

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

func (i *Inspector) GetOpCount(ctx context.Context, mcmAddr string) (uint64, error) {
	client := mcmsbindings.NewMcmsClient(i.invoker, mcmAddr)

	count, err := client.GetOpCount(ctx)
	if err != nil {
		return 0, fmt.Errorf("stellar mcms get_op_count: %w", err)
	}

	return count, nil
}

func (i *Inspector) GetRoot(ctx context.Context, mcmAddr string) (common.Hash, uint32, error) {
	client := mcmsbindings.NewMcmsClient(i.invoker, mcmAddr)

	root, validUntil, err := client.GetRoot(ctx)
	if err != nil {
		return common.Hash{}, 0, fmt.Errorf("stellar mcms get_root: %w", err)
	}

	return root, validUntil, nil
}

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
			}{
				ConfigVersion:   metadata.ConfigVersion,
				EncodingVersion: metadata.EncodingVersion,
			})

			return b
		}(),
	}, nil
}

func (i *Inspector) GetOwner(ctx context.Context, mcmAddr string) (*string, error) {
	client := mcmsbindings.NewMcmsClient(i.invoker, mcmAddr)

	owner, err := client.Owner(ctx)
	if err != nil {
		return nil, fmt.Errorf("stellar mcms owner: %w", err)
	}

	return owner, nil
}

func (i *Inspector) GetPendingOwner(ctx context.Context, mcmAddr string) (*string, error) {
	client := mcmsbindings.NewMcmsClient(i.invoker, mcmAddr)

	pending, err := client.GetPendingOwner(ctx)
	if err != nil {
		return nil, fmt.Errorf("stellar mcms get_pending_owner: %w", err)
	}

	return pending, nil
}

func (i *Inspector) GetChainNetworkID(ctx context.Context, mcmAddr string) ([32]byte, error) {
	if i == nil || i.invoker == nil {
		return [32]byte{}, fmt.Errorf("stellar mcms inspector invoker is nil")
	}

	result, err := i.invoker.SimulateContract(ctx, mcmAddr, "chain_network_id", nil)
	if err != nil {
		return [32]byte{}, fmt.Errorf("stellar mcms chain_network_id: %w", err)
	}

	if result == nil {
		return [32]byte{}, fmt.Errorf("stellar mcms chain_network_id returned no value")
	}

	networkID, err := scval.Bytes32FromScVal(*result)
	if err != nil {
		return [32]byte{}, fmt.Errorf("stellar mcms decode chain_network_id: %w", err)
	}

	return networkID, nil
}
