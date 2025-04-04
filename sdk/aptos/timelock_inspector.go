package aptos

import (
	"context"
	"fmt"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/smartcontractkit/chainlink-aptos/bindings/mcms"

	"github.com/smartcontractkit/mcms/sdk"
)

var _ sdk.TimelockInspector = (*TimelockInspector)(nil)

// TimelockInspector is an Inspector implementation for EVM chains for accessing the RBACTimelock contract
type TimelockInspector struct {
	client aptos.AptosRpcClient

	bindingFn func(address aptos.AccountAddress, client aptos.AptosRpcClient) mcms.MCMS
}

// NewTimelockInspector creates a new TimelockInspector
func NewTimelockInspector(client aptos.AptosRpcClient) *TimelockInspector {
	return &TimelockInspector{
		client:    client,
		bindingFn: mcms.Bind,
	}
}

// GetProposers returns the list of addresses with the proposer role
func (tm TimelockInspector) GetProposers(ctx context.Context, address string) ([]string, error) {
	return nil, nil
}

// GetExecutors returns the list of addresses with the executor role
func (tm TimelockInspector) GetExecutors(ctx context.Context, address string) ([]string, error) {
	return nil, nil
}

// GetBypassers returns the list of addresses with the bypasser role
func (tm TimelockInspector) GetBypassers(ctx context.Context, address string) ([]string, error) {
	return nil, nil
}

// GetCancellers returns the list of addresses with the canceller role
func (tm TimelockInspector) GetCancellers(ctx context.Context, address string) ([]string, error) {
	return nil, nil
}

func (tm TimelockInspector) IsOperation(ctx context.Context, address string, opID [32]byte) (bool, error) {
	mcmsAddress, err := hexToAddress(address)
	if err != nil {
		return false, fmt.Errorf("failed to parse MCMS address %q: %w", mcmsAddress, err)
	}
	mcmsBinding := tm.bindingFn(mcmsAddress, tm.client)

	return mcmsBinding.MCMS().TimelockIsOperation(nil, opID[:])
}

func (tm TimelockInspector) IsOperationPending(ctx context.Context, address string, opID [32]byte) (bool, error) {
	mcmsAddress, err := hexToAddress(address)
	if err != nil {
		return false, fmt.Errorf("failed to parse MCMS address %q: %w", mcmsAddress, err)
	}
	mcmsBinding := tm.bindingFn(mcmsAddress, tm.client)

	return mcmsBinding.MCMS().TimelockIsOperationPending(nil, opID[:])
}

func (tm TimelockInspector) IsOperationReady(ctx context.Context, address string, opID [32]byte) (bool, error) {
	mcmsAddress, err := hexToAddress(address)
	if err != nil {
		return false, fmt.Errorf("failed to parse MCMS address %q: %w", mcmsAddress, err)
	}
	mcmsBinding := tm.bindingFn(mcmsAddress, tm.client)

	return mcmsBinding.MCMS().TimelockIsOperationReady(nil, opID[:])
}

func (tm TimelockInspector) IsOperationDone(ctx context.Context, address string, opID [32]byte) (bool, error) {
	mcmsAddress, err := hexToAddress(address)
	if err != nil {
		return false, fmt.Errorf("failed to parse MCMS address %q: %w", mcmsAddress, err)
	}
	mcmsBinding := tm.bindingFn(mcmsAddress, tm.client)

	return mcmsBinding.MCMS().TimelockIsOperationDone(nil, opID[:])
}
