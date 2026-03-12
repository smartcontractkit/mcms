package aptos

import (
	"context"
	"errors"
	"fmt"

	"github.com/smartcontractkit/mcms/sdk"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/api"

	"github.com/smartcontractkit/chainlink-aptos/bindings/bind"
	"github.com/smartcontractkit/chainlink-aptos/bindings/curse_mcms"
	"github.com/smartcontractkit/chainlink-aptos/bindings/mcms"
)

var _ sdk.TimelockInspector = (*TimelockInspector)(nil)

// timelockContract is the subset of contract methods needed for timelock inspection and execution.
// Both mcms.MCMSContract and curse_mcms.CurseMCMSContract satisfy this interface.
type timelockContract interface {
	TimelockMinDelay(opts *bind.CallOpts) (uint64, error)
	TimelockIsOperation(opts *bind.CallOpts, id []byte) (bool, error)
	TimelockIsOperationPending(opts *bind.CallOpts, id []byte) (bool, error)
	TimelockIsOperationReady(opts *bind.CallOpts, id []byte) (bool, error)
	TimelockIsOperationDone(opts *bind.CallOpts, id []byte) (bool, error)
	TimelockExecuteBatch(opts *bind.TransactOpts, targets []aptos.AccountAddress, moduleNames []string, functionNames []string, datas [][]byte, predecessor []byte, salt []byte) (*api.PendingTransaction, error)
}

// TimelockInspector is an Inspector implementation for Aptos, for accessing the MCMS-Timelock contract
type TimelockInspector struct {
	client aptos.AptosRpcClient

	contractFn func(address aptos.AccountAddress, client aptos.AptosRpcClient) timelockContract
}

// NewTimelockInspector creates a new TimelockInspector that talks to a standard MCMS contract.
func NewTimelockInspector(client aptos.AptosRpcClient) *TimelockInspector {
	return NewTimelockInspectorWithMCMSType(client, MCMSTypeRegular)
}

// NewTimelockInspectorWithMCMSType creates a TimelockInspector that talks to either a
// standard MCMS or CurseMCMS contract depending on mcmsType.
func NewTimelockInspectorWithMCMSType(client aptos.AptosRpcClient, mcmsType MCMSType) *TimelockInspector {
	var fn func(aptos.AccountAddress, aptos.AptosRpcClient) timelockContract
	switch mcmsType {
	case MCMSTypeCurse:
		fn = func(addr aptos.AccountAddress, c aptos.AptosRpcClient) timelockContract {
			return curse_mcms.Bind(addr, c).CurseMCMS()
		}
	case MCMSTypeRegular:
		fn = func(addr aptos.AccountAddress, c aptos.AptosRpcClient) timelockContract {
			return mcms.Bind(addr, c).MCMS()
		}
	default:
		fn = func(addr aptos.AccountAddress, c aptos.AptosRpcClient) timelockContract {
			return mcms.Bind(addr, c).MCMS()
		}
	}

	return &TimelockInspector{
		client:     client,
		contractFn: fn,
	}
}

func (tm TimelockInspector) GetMinDelay(ctx context.Context, address string) (uint64, error) {
	mcmsAddress, err := hexToAddress(address)
	if err != nil {
		return 0, fmt.Errorf("failed to parse MCMS address %q: %w", mcmsAddress, err)
	}
	contract := tm.contractFn(mcmsAddress, tm.client)

	return contract.TimelockMinDelay(nil)
}

// GetProposers returns the list of addresses with the proposer role
func (tm TimelockInspector) GetProposers(ctx context.Context, address string) ([]string, error) {
	return nil, errors.New("unsupported on Aptos")
}

// GetExecutors returns the list of addresses with the executor role
func (tm TimelockInspector) GetExecutors(ctx context.Context, address string) ([]string, error) {
	return nil, errors.New("unsupported on Aptos")
}

// GetBypassers returns the list of addresses with the bypasser role
func (tm TimelockInspector) GetBypassers(ctx context.Context, address string) ([]string, error) {
	return nil, errors.New("unsupported on Aptos")
}

// GetCancellers returns the list of addresses with the canceller role
func (tm TimelockInspector) GetCancellers(ctx context.Context, address string) ([]string, error) {
	return nil, errors.New("unsupported on Aptos")
}

func (tm TimelockInspector) IsOperation(ctx context.Context, address string, opID [32]byte) (bool, error) {
	mcmsAddress, err := hexToAddress(address)
	if err != nil {
		return false, fmt.Errorf("failed to parse MCMS address %q: %w", mcmsAddress, err)
	}
	contract := tm.contractFn(mcmsAddress, tm.client)

	return contract.TimelockIsOperation(nil, opID[:])
}

func (tm TimelockInspector) IsOperationPending(ctx context.Context, address string, opID [32]byte) (bool, error) {
	mcmsAddress, err := hexToAddress(address)
	if err != nil {
		return false, fmt.Errorf("failed to parse MCMS address %q: %w", mcmsAddress, err)
	}
	contract := tm.contractFn(mcmsAddress, tm.client)

	return contract.TimelockIsOperationPending(nil, opID[:])
}

func (tm TimelockInspector) IsOperationReady(ctx context.Context, address string, opID [32]byte) (bool, error) {
	mcmsAddress, err := hexToAddress(address)
	if err != nil {
		return false, fmt.Errorf("failed to parse MCMS address %q: %w", mcmsAddress, err)
	}
	contract := tm.contractFn(mcmsAddress, tm.client)

	return contract.TimelockIsOperationReady(nil, opID[:])
}

func (tm TimelockInspector) IsOperationDone(ctx context.Context, address string, opID [32]byte) (bool, error) {
	mcmsAddress, err := hexToAddress(address)
	if err != nil {
		return false, fmt.Errorf("failed to parse MCMS address %q: %w", mcmsAddress, err)
	}
	contract := tm.contractFn(mcmsAddress, tm.client)

	return contract.TimelockIsOperationDone(nil, opID[:])
}
