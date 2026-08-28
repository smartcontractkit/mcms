package stellar

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/chainlink-stellar/bindings"
	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"
	stellarrpc "github.com/stellar/go-stellar-sdk/clients/rpcclient"
	"github.com/stellar/go-stellar-sdk/xdr"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.TimelockConfigurer = (*TimelockConfigurer)(nil)

type TimelockConfigurer struct {
	invoker bindings.Invoker
	caller  string
}

func NewTimelockConfigurer(client *stellarrpc.Client, auth bindings.Signer, selector uint64, caller string) (*TimelockConfigurer, error) {
	invoker, err := NewInvoker(client, auth, selector)
	if err != nil {
		return nil, err
	}

	return &TimelockConfigurer{invoker: invoker, caller: caller}, nil
}

// NewTimelockConfigurerWithNetworkPassphrase creates a Timelock configurer
// with an explicit network passphrase supplied by the deployment framework.
func NewTimelockConfigurerWithNetworkPassphrase(client *stellarrpc.Client, auth bindings.Signer, passphrase, caller string) (*TimelockConfigurer, error) {
	invoker, err := NewInvokerWithNetworkPassphrase(client, auth, passphrase)
	if err != nil {
		return nil, err
	}

	return NewTimelockConfigurerFromInvoker(invoker, caller), nil
}

// NewTimelockConfigurerFromInvoker creates a configurer for deployment code
// that already owns a bindings.Invoker.
func NewTimelockConfigurerFromInvoker(invoker bindings.Invoker, caller string) *TimelockConfigurer {
	return &TimelockConfigurer{invoker: invoker, caller: caller}
}

func (c *TimelockConfigurer) invoke(ctx context.Context, address, fn string, args []xdr.ScVal) (types.TransactionResult, error) {
	if c == nil || c.invoker == nil {
		return types.TransactionResult{}, fmt.Errorf("stellar timelock invoker is nil")
	}
	if _, err := c.invoker.InvokeContract(ctx, address, fn, args); err != nil {
		return types.TransactionResult{}, err
	}

	return types.NewTransactionResult("", nil, "stellar"), nil
}
func (c *TimelockConfigurer) UpdateDelay(ctx context.Context, a string, d uint64) (types.TransactionResult, error) {
	return c.invoke(ctx, a, "update_delay", []xdr.ScVal{scval.AddressToScVal(c.caller), scval.Uint64ToScVal(d)})
}

func (c *TimelockConfigurer) GrantRole(ctx context.Context, a string, r sdk.TimelockRole, target string) (types.TransactionResult, error) {
	return c.invoke(ctx, a, "grant_role", []xdr.ScVal{scval.AddressToScVal(c.caller), scval.SymbolToScVal(stellarRoleName(r)), scval.AddressToScVal(target)})
}

func stellarRoleName(r sdk.TimelockRole) string {
	switch r {
	case sdk.TimelockRoleAdmin:
		return "ADMIN"
	case sdk.TimelockRoleBypasser:
		return "BYPASSER"
	case sdk.TimelockRoleCanceller:
		return "CANCELLER"
	case sdk.TimelockRoleExecutor:
		return "EXECUTOR"
	case sdk.TimelockRoleProposer:
		return "PROPOSER"
	default:
		return ""
	}
}
