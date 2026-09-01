package stellar

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/chainlink-stellar/bindings"
	tlb "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/timelock"
	stellarrpc "github.com/stellar/go-stellar-sdk/clients/rpcclient"

	"github.com/smartcontractkit/mcms/sdk"
)

var _ sdk.TimelockInspector = (*TimelockInspector)(nil)

type TimelockInspector struct {
	invoker bindings.Invoker
}

func NewTimelockInspector(client *stellarrpc.Client, auth bindings.Signer, selector uint64) (*TimelockInspector, error) {
	invoker, err := NewInvoker(client, auth, selector)
	if err != nil {
		return nil, err
	}

	return NewTimelockInspectorFromInvoker(invoker), nil
}

// NewTimelockInspectorWithNetworkPassphrase creates an inspector with an
// explicit network passphrase supplied by the deployment framework.
func NewTimelockInspectorWithNetworkPassphrase(client *stellarrpc.Client, auth bindings.Signer, passphrase string) (*TimelockInspector, error) {
	invoker, err := NewInvokerWithNetworkPassphrase(client, auth, passphrase)
	if err != nil {
		return nil, err
	}

	return NewTimelockInspectorFromInvoker(invoker), nil
}

// NewTimelockInspectorFromInvoker creates an inspector for callers that
// already own a bindings.Invoker.
func NewTimelockInspectorFromInvoker(invoker bindings.Invoker) *TimelockInspector {
	return &TimelockInspector{invoker: invoker}
}

func (i *TimelockInspector) client(address string) *tlb.TimelockClient {
	return tlb.NewTimelockClient(i.invoker, address)
}

func (i *TimelockInspector) members(ctx context.Context, address, role string) ([]string, error) {
	if i == nil || i.invoker == nil {
		return nil, fmt.Errorf("stellar timelock invoker is nil")
	}

	c := i.client(address)
	n, err := c.GetRoleMemberCount(ctx, role)
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, n)
	for j := range n {
		a, err := c.GetRoleMember(ctx, role, j)
		if err != nil {
			return nil, err
		}

		out = append(out, a)
	}

	return out, nil
}

func (i *TimelockInspector) GetProposers(ctx context.Context, address string) ([]string, error) {
	return i.members(ctx, address, "PROPOSER")
}

func (i *TimelockInspector) GetExecutors(ctx context.Context, address string) ([]string, error) {
	return i.members(ctx, address, "EXECUTOR")
}

func (i *TimelockInspector) GetBypassers(ctx context.Context, address string) ([]string, error) {
	return i.members(ctx, address, "BYPASSER")
}

func (i *TimelockInspector) GetCancellers(ctx context.Context, address string) ([]string, error) {
	return i.members(ctx, address, "CANCELLER")
}

func (i *TimelockInspector) IsOperation(ctx context.Context, address string, id [32]byte) (bool, error) {
	if i == nil || i.invoker == nil {
		return false, fmt.Errorf("stellar timelock invoker is nil")
	}

	return i.client(address).IsOperation(ctx, id)
}

func (i *TimelockInspector) IsOperationPending(ctx context.Context, address string, id [32]byte) (bool, error) {
	if i == nil || i.invoker == nil {
		return false, fmt.Errorf("stellar timelock invoker is nil")
	}

	return i.client(address).IsOperationPending(ctx, id)
}

func (i *TimelockInspector) IsOperationReady(ctx context.Context, address string, id [32]byte) (bool, error) {
	if i == nil || i.invoker == nil {
		return false, fmt.Errorf("stellar timelock invoker is nil")
	}

	return i.client(address).IsOperationReady(ctx, id)
}

func (i *TimelockInspector) IsOperationDone(ctx context.Context, address string, id [32]byte) (bool, error) {
	if i == nil || i.invoker == nil {
		return false, fmt.Errorf("stellar timelock invoker is nil")
	}

	return i.client(address).IsOperationDone(ctx, id)
}

func (i *TimelockInspector) GetMinDelay(ctx context.Context, address string) (uint64, error) {
	if i == nil || i.invoker == nil {
		return 0, fmt.Errorf("stellar timelock invoker is nil")
	}

	return i.client(address).GetMinDelay(ctx)
}

// IsInitialized reports whether the timelock has already been initialized.
func (i *TimelockInspector) IsInitialized(ctx context.Context, address string) (bool, error) {
	if i == nil || i.invoker == nil {
		return false, fmt.Errorf("stellar timelock invoker is nil")
	}

	if address == "" {
		return false, fmt.Errorf("stellar timelock contract ID is empty")
	}

	client := i.client(address)

	for _, role := range []string{"ADMIN", "PROPOSER", "CANCELLER", "BYPASSER"} {
		count, err := client.GetRoleMemberCount(ctx, role)
		if err != nil {
			return false, fmt.Errorf("get stellar timelock %s role member count: %w", role, err)
		}

		if count > 0 {
			return true, nil
		}
	}

	minDelay, err := client.GetMinDelay(ctx)
	if err != nil {
		return false, fmt.Errorf("get stellar timelock minimum delay: %w", err)
	}

	return minDelay > 0, nil
}
