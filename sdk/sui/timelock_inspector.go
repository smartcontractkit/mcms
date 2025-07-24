package sui

import (
	"context"
	"errors"
	"fmt"

	"github.com/block-vision/sui-go-sdk/sui"
	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
	module_mcms "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms"
	bindutils "github.com/smartcontractkit/chainlink-sui/bindings/utils"

	"github.com/smartcontractkit/mcms/sdk"
)

var _ sdk.TimelockInspector = (*TimelockInspector)(nil)

// TimelockInspector is an Inspector implementation for Sui, for accessing the MCMS-Timelock contract
type TimelockInspector struct {
	client        sui.ISuiAPI
	signer        bindutils.SuiSigner
	mcmsPackageId string
	mcms          *module_mcms.McmsContract
}

// NewTimelockInspector creates a new TimelockInspector
func NewTimelockInspector(client sui.ISuiAPI, signer bindutils.SuiSigner, mcmsPackageId string) (*TimelockInspector, error) {
	mcms, err := module_mcms.NewMcms(mcmsPackageId, client)
	if err != nil {
		return nil, err
	}

	return &TimelockInspector{
		client:        client,
		signer:        signer,
		mcmsPackageId: mcmsPackageId,
		mcms:          mcms,
	}, nil
}

// GetProposers returns the list of addresses with the proposer role
func (tm TimelockInspector) GetProposers(ctx context.Context, address string) ([]string, error) {
	return nil, errors.New("unsupported on Sui")
}

// GetExecutors returns the list of addresses with the executor role
func (tm TimelockInspector) GetExecutors(ctx context.Context, address string) ([]string, error) {
	return nil, errors.New("unsupported on Sui")
}

// GetBypassers returns the list of addresses with the bypasser role
func (tm TimelockInspector) GetBypassers(ctx context.Context, address string) ([]string, error) {
	return nil, errors.New("unsupported on Sui")
}

// GetCancellers returns the list of addresses with the canceller role
func (tm TimelockInspector) GetCancellers(ctx context.Context, address string) ([]string, error) {
	return nil, errors.New("unsupported on Sui")
}

func (tm TimelockInspector) IsOperation(ctx context.Context, address string, opID [32]byte) (bool, error) {
	timelockObj := bind.Object{Id: address}

	opts := &bind.CallOpts{
		Signer: tm.signer,
	}

	result, err := tm.mcms.DevInspect().TimelockIsOperation(ctx, opts, timelockObj, opID[:])
	if err != nil {
		return false, fmt.Errorf("failed to check if operation exists: %w", err)
	}

	return result, nil
}

func (tm TimelockInspector) IsOperationPending(ctx context.Context, address string, opID [32]byte) (bool, error) {
	timelockObj := bind.Object{Id: address}

	opts := &bind.CallOpts{
		Signer: tm.signer,
	}

	result, err := tm.mcms.DevInspect().TimelockIsOperationPending(ctx, opts, timelockObj, opID[:])
	if err != nil {
		return false, fmt.Errorf("failed to check if operation is pending: %w", err)
	}

	return result, nil
}

func (tm TimelockInspector) IsOperationReady(ctx context.Context, address string, opID [32]byte) (bool, error) {
	timelockObj := bind.Object{Id: address}
	clockObj := bind.Object{Id: "0x6"} // Clock object ID in Sui

	opts := &bind.CallOpts{
		Signer: tm.signer,
	}

	result, err := tm.mcms.DevInspect().TimelockIsOperationReady(ctx, opts, timelockObj, clockObj, opID[:])
	if err != nil {
		return false, fmt.Errorf("failed to check if operation is ready: %w", err)
	}

	return result, nil
}

func (tm TimelockInspector) IsOperationDone(ctx context.Context, address string, opID [32]byte) (bool, error) {
	timelockObj := bind.Object{Id: address}

	opts := &bind.CallOpts{
		Signer: tm.signer,
	}

	result, err := tm.mcms.DevInspect().TimelockIsOperationDone(ctx, opts, timelockObj, opID[:])
	if err != nil {
		return false, fmt.Errorf("failed to check if operation is done: %w", err)
	}

	return result, nil
}
