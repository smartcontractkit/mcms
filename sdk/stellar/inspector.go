package stellar

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stellar/go-stellar-sdk/strkey"

	"github.com/smartcontractkit/chainlink-stellar/bindings"
	stellarmcms "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/mcms"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Inspector = (*Inspector)(nil)

// Inspector reads MCMS contract state on Stellar via Soroban simulation (bindings.Invoker).
type Inspector struct {
	ConfigTransformer
	invoker bindings.Invoker
}

// NewInspector constructs an Inspector that uses invoker for read-only SimulateContract calls.
func NewInspector(invoker bindings.Invoker) *Inspector {
	return &Inspector{
		invoker: invoker,
	}
}

func (i *Inspector) contractClient(mcmAddr string) (*stellarmcms.McmsClient, error) {
	id, err := normalizeContractIDStrkey(mcmAddr)
	if err != nil {
		return nil, err
	}

	return stellarmcms.NewMcmsClient(i.invoker, id), nil
}

// GetConfig returns the live multisig configuration from the contract.
func (i *Inspector) GetConfig(ctx context.Context, mcmAddr string) (*types.Config, error) {
	client, err := i.contractClient(mcmAddr)
	if err != nil {
		return nil, err
	}

	cfg, err := client.GetConfig(ctx)
	if err != nil {
		return nil, err
	}

	return i.ToConfig(cfg)
}

// GetOpCount returns the executed operation counter from the contract.
func (i *Inspector) GetOpCount(ctx context.Context, mcmAddr string) (uint64, error) {
	client, err := i.contractClient(mcmAddr)
	if err != nil {
		return 0, err
	}

	return client.GetOpCount(ctx)
}

// GetRoot returns the current expiring Merkle root and its valid-until ledger/time bound.
func (i *Inspector) GetRoot(ctx context.Context, mcmAddr string) (common.Hash, uint32, error) {
	client, err := i.contractClient(mcmAddr)
	if err != nil {
		return common.Hash{}, 0, err
	}

	root, validUntil, err := client.GetRoot(ctx)
	if err != nil {
		return common.Hash{}, 0, err
	}

	return common.BytesToHash(root[:]), validUntil, nil
}

// GetRootMetadata returns proposal metadata aligned with MCMS (starting op count + MCM address).
func (i *Inspector) GetRootMetadata(ctx context.Context, mcmAddr string) (types.ChainMetadata, error) {
	client, err := i.contractClient(mcmAddr)
	if err != nil {
		return types.ChainMetadata{}, err
	}

	meta, err := client.GetRootMetadata(ctx)
	if err != nil {
		return types.ChainMetadata{}, err
	}

	if meta == nil {
		return types.ChainMetadata{}, fmt.Errorf("nil root metadata from contract")
	}

	return types.ChainMetadata{
		StartingOpCount: meta.PreOpCount,
		MCMAddress:      mcmAddr,
	}, nil
}

// normalizeContractIDStrkey accepts contract id hex or strkey and returns canonical contract strkey (C…).
func normalizeContractIDStrkey(s string) (string, error) {
	raw, err := ParseContractID(s)
	if err != nil {
		return "", err
	}

	encoded, err := strkey.Encode(strkey.VersionByteContract, raw[:])
	if err != nil {
		return "", fmt.Errorf("encode contract id: %w", err)
	}

	return encoded, nil
}
