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

type TimelockInspector struct{ invoker bindings.Invoker }

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
	c := i.client(address)
	n, err := c.GetRoleMemberCount(ctx, role)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, n)
	for j := range 32 {
		a, e := c.GetRoleMember(ctx, role, uint32(j))
		if e != nil {
			return nil, e
		}
		out = append(out, a)
	}

	return out, nil
}

func (i *TimelockInspector) GetProposers(ctx context.Context, a string) ([]string, error) {

	return i.members(ctx, a, "PROPOSER")
}

func (i *TimelockInspector) GetExecutors(ctx context.Context, a string) ([]string, error) {

	return i.members(ctx, a, "EXECUTOR")
}

func (i *TimelockInspector) GetBypassers(ctx context.Context, a string) ([]string, error) {

	return i.members(ctx, a, "BYPASSER")
}

func (i *TimelockInspector) GetCancellers(ctx context.Context, a string) ([]string, error) {

	return i.members(ctx, a, "CANCELLER")
}

func (i *TimelockInspector) IsOperation(ctx context.Context, a string, id [32]byte) (bool, error) {

	return i.client(a).IsOperation(ctx, id)
}

func (i *TimelockInspector) IsOperationPending(ctx context.Context, a string, id [32]byte) (bool, error) {

	return i.client(a).IsOperationPending(ctx, id)
}

func (i *TimelockInspector) IsOperationReady(ctx context.Context, a string, id [32]byte) (bool, error) {

	return i.client(a).IsOperationReady(ctx, id)
}

func (i *TimelockInspector) IsOperationDone(ctx context.Context, a string, id [32]byte) (bool, error) {

	return i.client(a).IsOperationDone(ctx, id)
}

func (i *TimelockInspector) GetMinDelay(ctx context.Context, a string) (uint64, error) {
	if i == nil || i.invoker == nil {
		return 0, fmt.Errorf("stellar timelock invoker is nil")
	}

	return i.client(a).GetMinDelay(ctx)
}
