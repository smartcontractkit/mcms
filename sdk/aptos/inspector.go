package aptos

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	"github.com/aptos-labs/aptos-go-sdk"

	"github.com/smartcontractkit/chainlink-aptos/bindings/bind"
	curse_mcms_pkg "github.com/smartcontractkit/chainlink-aptos/bindings/curse_mcms"
	module_curse_mcms "github.com/smartcontractkit/chainlink-aptos/bindings/curse_mcms/curse_mcms"
	mcms_pkg "github.com/smartcontractkit/chainlink-aptos/bindings/mcms"
	module_mcms "github.com/smartcontractkit/chainlink-aptos/bindings/mcms/mcms"
)

// mcmsViewer is the subset of view functions used by the Inspector, normalized
// to module_mcms types so the rest of the code doesn't need to know which
// on-chain module it is talking to.
type mcmsViewer interface {
	GetConfig(opts *bind.CallOpts, role byte) (module_mcms.Config, error)
	GetOpCount(opts *bind.CallOpts, role byte) (uint64, error)
	GetRoot(opts *bind.CallOpts, role byte) ([]byte, uint64, error)
	GetRootMetadata(opts *bind.CallOpts, role byte) (module_mcms.RootMetadata, error)
}

var _ mcmsViewer = &curseMcmsViewer{}

// curseMcmsViewer adapts a CurseMCMSInterface to mcmsViewer by converting
// curse_mcms types to module_mcms types (struct fields are identical).
type curseMcmsViewer struct {
	inner module_curse_mcms.CurseMCMSInterface
}

func (c *curseMcmsViewer) GetConfig(opts *bind.CallOpts, role byte) (module_mcms.Config, error) {
	cfg, err := c.inner.GetConfig(opts, role)
	if err != nil {
		return module_mcms.Config{}, err
	}
	signers := make([]module_mcms.Signer, 0, len(cfg.Signers))
	for _, s := range cfg.Signers {
		signers = append(signers, module_mcms.Signer{Addr: s.Addr, Index: s.Index, Group: s.Group})
	}

	return module_mcms.Config{
		Signers:      signers,
		GroupQuorums: cfg.GroupQuorums,
		GroupParents: cfg.GroupParents,
	}, nil
}

func (c *curseMcmsViewer) GetOpCount(opts *bind.CallOpts, role byte) (uint64, error) {
	return c.inner.GetOpCount(opts, role)
}

func (c *curseMcmsViewer) GetRoot(opts *bind.CallOpts, role byte) ([]byte, uint64, error) {
	return c.inner.GetRoot(opts, role)
}

func (c *curseMcmsViewer) GetRootMetadata(opts *bind.CallOpts, role byte) (module_mcms.RootMetadata, error) {
	rm, err := c.inner.GetRootMetadata(opts, role)
	if err != nil {
		return module_mcms.RootMetadata{}, err
	}

	return module_mcms.RootMetadata{
		Role:                 rm.Role,
		ChainId:              rm.ChainId,
		Multisig:             rm.Multisig,
		PreOpCount:           rm.PreOpCount,
		PostOpCount:          rm.PostOpCount,
		OverridePreviousRoot: rm.OverridePreviousRoot,
	}, nil
}

var _ sdk.Inspector = &Inspector{}

type Inspector struct {
	ConfigTransformer
	client   aptos.AptosRpcClient
	role     TimelockRole
	viewerFn func(address aptos.AccountAddress, client aptos.AptosRpcClient) mcmsViewer
}

// NewInspector creates an Inspector. When isCurseMCMS is true the Inspector
// talks to a CurseMCMS contract; when false it talks to a standard MCMS contract.
func NewInspector(client aptos.AptosRpcClient, role TimelockRole, isCurseMCMS bool) *Inspector {
	var vfn func(aptos.AccountAddress, aptos.AptosRpcClient) mcmsViewer
	if isCurseMCMS {
		vfn = func(addr aptos.AccountAddress, c aptos.AptosRpcClient) mcmsViewer {
			return &curseMcmsViewer{inner: curse_mcms_pkg.Bind(addr, c).CurseMCMS()}
		}
	} else {
		vfn = func(addr aptos.AccountAddress, c aptos.AptosRpcClient) mcmsViewer {
			return mcms_pkg.Bind(addr, c).MCMS()
		}
	}

	return &Inspector{
		client:   client,
		role:     role,
		viewerFn: vfn,
	}
}

func (i Inspector) GetConfig(ctx context.Context, mcmsAddr string) (*types.Config, error) {
	mcmsAddress, err := hexToAddress(mcmsAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MCMS address: %w", err)
	}
	viewer := i.viewerFn(mcmsAddress, i.client)

	config, err := viewer.GetConfig(nil, i.role.Byte())
	if err != nil {
		return nil, fmt.Errorf("get config: %w", err)
	}

	return i.ToConfig(config)
}

func (i Inspector) GetOpCount(ctx context.Context, mcmsAddr string) (uint64, error) {
	mcmsAddress, err := hexToAddress(mcmsAddr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse MCMS address: %w", err)
	}
	viewer := i.viewerFn(mcmsAddress, i.client)

	opCount, err := viewer.GetOpCount(nil, i.role.Byte())
	if err != nil {
		return 0, fmt.Errorf("get op count: %w", err)
	}

	return opCount, nil
}

func (i Inspector) GetRoot(ctx context.Context, mcmsAddr string) (common.Hash, uint32, error) {
	mcmsAddress, err := hexToAddress(mcmsAddr)
	if err != nil {
		return common.Hash{}, 0, fmt.Errorf("failed to parse MCMS address: %w", err)
	}
	viewer := i.viewerFn(mcmsAddress, i.client)

	root, validUntil, err := viewer.GetRoot(nil, i.role.Byte())
	if err != nil {
		return common.Hash{}, 0, fmt.Errorf("get root: %w", err)
	}

	//nolint:gosec
	return common.BytesToHash(root), uint32(validUntil), nil
}

func (i Inspector) GetRootMetadata(ctx context.Context, mcmsAddr string) (types.ChainMetadata, error) {
	mcmsAddress, err := hexToAddress(mcmsAddr)
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("failed to parse MCMS address: %w", err)
	}
	viewer := i.viewerFn(mcmsAddress, i.client)

	rootMetadata, err := viewer.GetRootMetadata(nil, i.role.Byte())
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("get root metadata: %w", err)
	}

	return types.ChainMetadata{
		StartingOpCount: rootMetadata.PreOpCount,
		MCMAddress:      rootMetadata.Multisig.StringLong(),
	}, nil
}
