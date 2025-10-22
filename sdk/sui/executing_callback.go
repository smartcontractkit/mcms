package sui

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/block-vision/sui-go-sdk/models"
	"github.com/block-vision/sui-go-sdk/sui"
	"github.com/block-vision/sui-go-sdk/transaction"

	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
	modulemcms "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms"
	module_mcms_deployer "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms_deployer"
	bindutils "github.com/smartcontractkit/chainlink-sui/bindings/utils"
)

const (
	UpgradeGasBudget = 500_000_000
)

type EntrypointArgEncoder interface {
	EncodeEntryPointArg(executingCallbackParams *transaction.Argument, target, module, function, stateObjID string, data []byte, typeArgs []string) (*bind.EncodedCall, error)
}

type ExecutingCallbackAppender interface {
	AppendPTB(ctx context.Context, ptb *transaction.Transaction, executeCallback *transaction.Argument, calls []Call) error
}

type ExecutingCallbackParams struct {
	client                         sui.ISuiAPI
	mcms                           modulemcms.IMcms
	mcmsPackageID                  string
	entryPointEncoder              EntrypointArgEncoder // Encoder for the entrypoint function. Users can provide their own implementation
	registryObj                    string
	accountObj                     string
	extractExecutingCallbackParams func(mcmsPackageID string, ptb *transaction.Transaction, vectorExecutingCallback *transaction.Argument) (*transaction.Argument, error)
	closeExecutingCallbackParams   func(mcmsPackageID string, ptb *transaction.Transaction, vectorExecutingCallback *transaction.Argument) error
	createDeployerFunc             func(mcmsPackageID string, client sui.ISuiAPI) (module_mcms_deployer.IMcmsDeployer, error)
}

func NewExecutingCallbackParams(client sui.ISuiAPI, mcms modulemcms.IMcms, mcmsPackageID string, entryPointEncoder EntrypointArgEncoder, registryObj string, accountObj string) *ExecutingCallbackParams {
	return &ExecutingCallbackParams{
		client:                         client,
		mcms:                           mcms,
		mcmsPackageID:                  mcmsPackageID,
		entryPointEncoder:              entryPointEncoder,
		registryObj:                    registryObj,
		accountObj:                     accountObj,
		extractExecutingCallbackParams: extractExecutingCallbackParams,
		closeExecutingCallbackParams:   closeExecutingCallbackParams,
		createDeployerFunc:             module_mcms_deployer.NewMcmsDeployer,
	}
}

func (e *ExecutingCallbackParams) AppendPTB(ctx context.Context, ptb *transaction.Transaction, executeCallback *transaction.Argument, calls []Call) error {
	opts := &bind.CallOpts{}

	for i, call := range calls {
		if err := e.processCall(ctx, ptb, opts, executeCallback, call, i); err != nil {
			return err
		}
	}

	return e.closeExecutingCallbackParams(e.mcmsPackageID, ptb, executeCallback)
}

func (e *ExecutingCallbackParams) processCall(ctx context.Context, ptb *transaction.Transaction, opts *bind.CallOpts, executeCallback *transaction.Argument, call Call, index int) error {
	targetString := e.formatTargetAddress(call.Target)

	executingCallbackParams, err := e.extractExecutingCallbackParams(e.mcmsPackageID, ptb, executeCallback)
	if err != nil {
		return fmt.Errorf("extracting ExecutingCallbackParams %d: %w", index, err)
	}

	if e.isTargetMCMSPackage(targetString) {
		return e.processMCMSPackageCall(ctx, ptb, opts, executingCallbackParams, call, index)
	}

	return e.processDestinationContractCall(ctx, ptb, opts, executingCallbackParams, call, targetString)
}

func (e *ExecutingCallbackParams) formatTargetAddress(target []byte) string {
	return fmt.Sprintf("0x%s", strings.ToLower(hex.EncodeToString(target)))
}

func (e *ExecutingCallbackParams) isTargetMCMSPackage(targetString string) bool {
	return targetString == e.mcmsPackageID
}

func (e *ExecutingCallbackParams) processMCMSPackageCall(ctx context.Context, ptb *transaction.Transaction, opts *bind.CallOpts, executingCallbackParams *transaction.Argument, call Call, index int) error {
	switch call.ModuleName {
	case "mcms_deployer":
		return e.processMCMSDeployerCall(ctx, ptb, opts, executingCallbackParams, call, index)
	case "mcms_account":
		return e.processMCMSAccountCall(ctx, ptb, opts, executingCallbackParams, index)
	default:
		return e.processMCMSMainModuleCall(ctx, ptb, opts, executingCallbackParams, call, index)
	}
}

func (e *ExecutingCallbackParams) processMCMSDeployerCall(ctx context.Context, ptb *transaction.Transaction, opts *bind.CallOpts, executingCallbackParams *transaction.Argument, call Call, index int) error {
	if call.FunctionName != "authorize_upgrade" {
		return fmt.Errorf("mcms_deployer calls must have FunctionName 'authorize_upgrade', got: %s", call.FunctionName)
	}

	// Step 1: Create and execute dispatch call
	upgradeTicketArg, err := e.executeDispatchToDeployer(ctx, ptb, opts, executingCallbackParams, call, index)
	if err != nil {
		return err
	}

	// Step 2: Perform package upgrade
	upgradeReceiptArg := e.performPackageUpgrade(ptb, call, upgradeTicketArg)

	// Step 3: Commit upgrade
	return e.commitUpgrade(ctx, ptb, opts, call, upgradeReceiptArg)
}

func (e *ExecutingCallbackParams) executeDispatchToDeployer(ctx context.Context, ptb *transaction.Transaction, opts *bind.CallOpts, executingCallbackParams *transaction.Argument, call Call, index int) (*transaction.Argument, error) {
	executeDispatchCall, err := e.mcms.Encoder().McmsDispatchToDeployerWithArgs(
		e.registryObj,
		call.StateObj,
		executingCallbackParams,
	)
	if err != nil {
		return nil, fmt.Errorf("creating ExecuteDispatchToDeployer call %d: %w", index, err)
	}

	upgradeTicketArg, err := e.mcms.Bound().AppendPTB(ctx, opts, ptb, executeDispatchCall)
	if err != nil {
		return nil, fmt.Errorf("adding ExecuteDispatchToDeployer call %d to PTB: %w", index, err)
	}

	return upgradeTicketArg, nil
}

func (e *ExecutingCallbackParams) performPackageUpgrade(ptb *transaction.Transaction, call Call, upgradeTicketArg *transaction.Argument) *transaction.Argument {
	ptb.SetGasBudget(UpgradeGasBudget)

	upgradeReceiptArg := ptb.Upgrade(
		call.CompiledModules,
		call.Dependencies,
		models.SuiAddress(call.PackageToUpgrade),
		*upgradeTicketArg,
	)

	return &upgradeReceiptArg
}

func (e *ExecutingCallbackParams) commitUpgrade(ctx context.Context, ptb *transaction.Transaction, opts *bind.CallOpts, call Call, upgradeReceiptArg *transaction.Argument) error {
	deployerContract, err := e.createDeployerFunc(e.mcmsPackageID, e.client)
	if err != nil {
		return fmt.Errorf("failed to create deployer contract: %w", err)
	}

	commitEncoded, err := deployerContract.Encoder().CommitUpgradeWithArgs(
		bind.Object{Id: call.StateObj},
		*upgradeReceiptArg,
	)
	if err != nil {
		return fmt.Errorf("failed to encode commit upgrade: %w", err)
	}

	_, err = deployerContract.Bound().AppendPTB(ctx, opts, ptb, commitEncoded)
	if err != nil {
		return fmt.Errorf("failed to append commit upgrade to PTB: %w", err)
	}

	return nil
}

func (e *ExecutingCallbackParams) processMCMSAccountCall(ctx context.Context, ptb *transaction.Transaction, opts *bind.CallOpts, executingCallbackParams *transaction.Argument, index int) error {
	executeDispatchCall, err := e.mcms.Encoder().McmsDispatchToAccountWithArgs(
		e.registryObj,
		e.accountObj,
		executingCallbackParams,
	)
	if err != nil {
		return fmt.Errorf("creating ExecuteDispatchToAccount call %d: %w", index, err)
	}

	_, err = e.mcms.Bound().AppendPTB(ctx, opts, ptb, executeDispatchCall)
	if err != nil {
		return fmt.Errorf("adding ExecuteDispatchToAccount call %d to PTB: %w", index, err)
	}

	return nil
}

func (e *ExecutingCallbackParams) processMCMSMainModuleCall(ctx context.Context, ptb *transaction.Transaction, opts *bind.CallOpts, executingCallbackParams *transaction.Argument, call Call, index int) error {
	executeDispatchCall, err := e.mcms.Encoder().McmsSetConfigWithArgs(
		e.registryObj,
		call.StateObj,
		executingCallbackParams,
	)
	if err != nil {
		return fmt.Errorf("creating ExecuteDispatchToAccount call %d: %w", index, err)
	}

	// Adjust function name to match mcms_ prefix
	executeDispatchCall.Function = fmt.Sprintf("mcms_%s", strings.TrimPrefix(call.FunctionName, "mcms_"))

	_, err = e.mcms.Bound().AppendPTB(ctx, opts, ptb, executeDispatchCall)
	if err != nil {
		return fmt.Errorf("adding ExecuteDispatchToAccount call %d to PTB: %w", index, err)
	}

	return nil
}

func (e *ExecutingCallbackParams) processDestinationContractCall(ctx context.Context, ptb *transaction.Transaction, opts *bind.CallOpts, executingCallbackParams *transaction.Argument, call Call, targetString string) error {
	entryPointCall, err := e.entryPointEncoder.EncodeEntryPointArg(
		executingCallbackParams,
		targetString,
		call.ModuleName,
		call.FunctionName,
		call.StateObj,
		call.Data,
		call.TypeArgs,
	)
	if err != nil {
		return fmt.Errorf("failed to create mcms_entrypoint call: %w", err)
	}

	_, err = e.mcms.Bound().AppendPTB(ctx, opts, ptb, entryPointCall)
	if err != nil {
		return fmt.Errorf("failed to append mcms_entrypoint call to PTB: %w", err)
	}

	return nil
}

func extractExecutingCallbackParams(mcmsPackageID string, ptb *transaction.Transaction, vectorExecutingCallback *transaction.Argument) (*transaction.Argument, error) {
	// Convert the type string to TypeTag
	executingCallbackParamsType := fmt.Sprintf("%s::mcms_registry::ExecutingCallbackParams", mcmsPackageID)
	typeTag, err := bindutils.ConvertTypeStringToTypeTag(executingCallbackParamsType)
	if err != nil {
		return nil, fmt.Errorf("failed to convert type string to TypeTag: %w", err)
	}

	// Create the vector pop_back call to extract an element by value
	// This gives us the actual ExecutingCallbackParams by value, consuming it from the vector
	executingCallbackParams := ptb.MoveCall(
		"0x1", // Standard library package for vector operations
		"vector",
		"remove",
		[]transaction.TypeTag{*typeTag}, // Type arguments
		[]transaction.Argument{*vectorExecutingCallback, ptb.Pure(uint64(0))}, // Arguments: vector and position 0 to remove
	)

	return &executingCallbackParams, nil
}

func closeExecutingCallbackParams(mcmsPackageID string, ptb *transaction.Transaction, vectorExecutingCallback *transaction.Argument) error {
	// Get the type tag for ExecutingCallbackParams
	executingCallbackParamsType := fmt.Sprintf("%s::mcms_registry::ExecutingCallbackParams", mcmsPackageID)
	typeTag, err := bindutils.ConvertTypeStringToTypeTag(executingCallbackParamsType)
	if err != nil {
		return fmt.Errorf("failed to convert type string to TypeTag: %w", err)
	}

	ptb.MoveCall(
		"0x1", // Standard library package
		"vector",
		"destroy_empty",
		[]transaction.TypeTag{*typeTag},
		[]transaction.Argument{*vectorExecutingCallback},
	)

	return nil
}
