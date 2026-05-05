package stellar

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/chainlink-stellar/bindings"
	timelockbindings "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/timelock"

	"github.com/smartcontractkit/mcms/sdk"
)

// Soroban timelock role symbols match contracts/timelock/src/types.rs (symbol_short).
const (
	timelockRoleProposer  = "PROPOSER"
	timelockRoleExecutor  = "EXECUTOR"
	timelockRoleBypasser  = "BYPASSER"
	timelockRoleCanceller = "CANCELLER"
)

var _ sdk.TimelockInspector = (*TimelockInspector)(nil)

// TimelockInspector reads Soroban RBACTimelock state via SimulateContract on [bindings.Invoker].
type TimelockInspector struct {
	invoker bindings.Invoker
}

// NewTimelockInspector constructs a TimelockInspector for the given invoker (RPC / deployer).
func NewTimelockInspector(invoker bindings.Invoker) *TimelockInspector {
	return &TimelockInspector{invoker: invoker}
}

func (t *TimelockInspector) clientFor(_ context.Context, address string) (*timelockbindings.TimelockClient, error) {
	id, err := normalizeContractIDStrkey(address)
	if err != nil {
		return nil, err
	}

	return timelockbindings.NewTimelockClient(t.invoker, id), nil
}

func (t *TimelockInspector) roleMembers(ctx context.Context, address string, role string) ([]string, error) {
	c, err := t.clientFor(ctx, address)
	if err != nil {
		return nil, err
	}

	n, err := c.GetRoleMemberCount(ctx, role)
	if err != nil {
		return nil, fmt.Errorf("get_role_member_count %s: %w", role, err)
	}

	out := make([]string, 0, n)
	for i := range n {
		member, err := c.GetRoleMember(ctx, role, i)
		if err != nil {
			return nil, fmt.Errorf("get_role_member %s index %d: %w", role, i, err)
		}

		out = append(out, member)
	}

	return out, nil
}

// GetProposers returns addresses with the PROPOSER role.
func (t *TimelockInspector) GetProposers(ctx context.Context, address string) ([]string, error) {
	return t.roleMembers(ctx, address, timelockRoleProposer)
}

// GetExecutors returns addresses with the EXECUTOR role.
func (t *TimelockInspector) GetExecutors(ctx context.Context, address string) ([]string, error) {
	return t.roleMembers(ctx, address, timelockRoleExecutor)
}

// GetBypassers returns addresses with the BYPASSER role.
func (t *TimelockInspector) GetBypassers(ctx context.Context, address string) ([]string, error) {
	return t.roleMembers(ctx, address, timelockRoleBypasser)
}

// GetCancellers returns addresses with the CANCELLER role.
func (t *TimelockInspector) GetCancellers(ctx context.Context, address string) ([]string, error) {
	return t.roleMembers(ctx, address, timelockRoleCanceller)
}

// IsOperation returns true if the operation id exists (any non-zero timestamp entry).
func (t *TimelockInspector) IsOperation(ctx context.Context, address string, opID [32]byte) (bool, error) {
	c, err := t.clientFor(ctx, address)
	if err != nil {
		return false, err
	}

	return c.IsOperation(ctx, opID)
}

// IsOperationPending returns true if the operation is scheduled but not yet ready or done.
func (t *TimelockInspector) IsOperationPending(ctx context.Context, address string, opID [32]byte) (bool, error) {
	c, err := t.clientFor(ctx, address)
	if err != nil {
		return false, err
	}

	return c.IsOperationPending(ctx, opID)
}

// IsOperationReady returns true if the operation is scheduled and the delay has elapsed.
func (t *TimelockInspector) IsOperationReady(ctx context.Context, address string, opID [32]byte) (bool, error) {
	c, err := t.clientFor(ctx, address)
	if err != nil {
		return false, err
	}

	return c.IsOperationReady(ctx, opID)
}

// IsOperationDone returns true if the operation has been executed.
func (t *TimelockInspector) IsOperationDone(ctx context.Context, address string, opID [32]byte) (bool, error) {
	c, err := t.clientFor(ctx, address)
	if err != nil {
		return false, err
	}

	return c.IsOperationDone(ctx, opID)
}

// GetMinDelay returns the timelock minimum delay in seconds.
func (t *TimelockInspector) GetMinDelay(ctx context.Context, address string) (uint64, error) {
	c, err := t.clientFor(ctx, address)
	if err != nil {
		return 0, err
	}

	return c.GetMinDelay(ctx)
}
