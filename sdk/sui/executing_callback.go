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
	EncodeEntryPointArg(executingCallbackParams *transaction.Argument, target, module, function, stateObjID string, data []byte) (*bind.EncodedCall, error)
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
	// Only used for object resolving caching
	opts := &bind.CallOpts{}
	// Process each ExecutingCallbackParams individually
	// We need to process them in reverse order since we're using pop_back to extract elements
	for i := len(calls) - 1; i >= 0; i-- {
		call := calls[i]

		// Ensure proper address formatting - convert bytes to hex with proper padding
		targetString := fmt.Sprintf("0x%s", strings.ToLower(hex.EncodeToString(call.Target)))
		isTargetMCMSPackage := targetString == e.mcmsPackageID

		// If the target is the MCMS package, we need to call ExecuteDispatchToAccount
		if isTargetMCMSPackage {
			// We need to extract individual ExecutingCallbackParams from the executeCallback vector
			executingCallbackParams, extractErr := e.extractExecutingCallbackParams(e.mcmsPackageID, ptb, executeCallback)
			if extractErr != nil {
				return fmt.Errorf("failed to extract executing callback params: %w", extractErr)
			}

			// Route based on module name within MCMS package
			if call.ModuleName == "mcms_deployer" {
				// Validate that this is an authorize_upgrade call
				if call.FunctionName != "authorize_upgrade" {
					return fmt.Errorf("mcms_deployer calls must have FunctionName 'authorize_upgrade', got: %s", call.FunctionName)
				}

				executeDispatchCall, err := e.mcms.Encoder().ExecuteDispatchToDeployerWithArgs(
					e.registryObj, // Registry object
					call.StateObj, // DeployerState object
					executingCallbackParams,
				)
				if err != nil {
					return fmt.Errorf("creating ExecuteDispatchToDeployer call %d: %w", i, err)
				}

				// Add the call to the PTB and capture the UpgradeTicket result
				upgradeTicketArg, err := e.mcms.Bound().AppendPTB(ctx, opts, ptb, executeDispatchCall)
				if err != nil {
					return fmt.Errorf("adding ExecuteDispatchToDeployer call %d to PTB: %w", i, err)
				}

				// Increase gas budget for the upgrade steps
				ptb.SetGasBudget(UpgradeGasBudget)

				// Step 2: Use the UpgradeTicket in package upgrade → produces UpgradeReceipt
				upgradeReceiptArg := ptb.Upgrade(
					call.CompiledModules,                     // Raw bytes (from Call)
					call.Dependencies,                        // Dependencies as addresses (from Call)
					models.SuiAddress(call.PackageToUpgrade), // Package being upgraded (from Call)
					*upgradeTicketArg,                        // UpgradeTicket from authorize step
				)

				deployerContract, err := e.createDeployerFunc(e.mcmsPackageID, e.client)
				if err != nil {
					return fmt.Errorf("failed to create deployer contract: %w", err)
				}

				commitEncoded, err := deployerContract.Encoder().CommitUpgradeWithArgs(
					bind.Object{Id: call.StateObj}, // DeployerState
					upgradeReceiptArg,              // UpgradeReceipt
				)
				if err != nil {
					return fmt.Errorf("failed to encode commit upgrade: %w", err)
				}

				_, err = deployerContract.Bound().AppendPTB(ctx, opts, ptb, commitEncoded)
				if err != nil {
					return fmt.Errorf("failed to append commit upgrade to PTB: %w", err)
				}
			} else {
				executeDispatchCall, err := e.mcms.Encoder().ExecuteDispatchToAccountWithArgs(
					e.registryObj,
					e.accountObj,
					executingCallbackParams,
				)
				if err != nil {
					return fmt.Errorf("creating ExecuteDispatchToAccount call %d: %w", i, err)
				}

				// Add the call to the PTB
				_, err = e.mcms.Bound().AppendPTB(ctx, opts, ptb, executeDispatchCall)
				if err != nil {
					return fmt.Errorf("adding ExecuteDispatchToAccount call %d to PTB: %w", i, err)
				}
			}
			// If this is a destination contract operation, we need to call the mcms_entrypoint like function
		} else {
			executingCallbackParams, err := e.extractExecutingCallbackParams(e.mcmsPackageID, ptb, executeCallback)
			if err != nil {
				return fmt.Errorf("extracting ExecutingCallbackParams %d: %w", i, err)
			}

			// Encode the entrypoint call
			entryPointCall, err := e.entryPointEncoder.EncodeEntryPointArg(executingCallbackParams, targetString, call.ModuleName, call.FunctionName, call.StateObj, call.Data)
			if err != nil {
				return fmt.Errorf("failed to create mcms_entrypoint call: %w", err)
			}

			_, err = e.mcms.Bound().AppendPTB(ctx, opts, ptb, entryPointCall)
			if err != nil {
				return fmt.Errorf("failed to append mcms_entrypoint call to PTB: %w", err)
			}
		}
	}

	// After processing all elements, the vector should be empty, we need to close it
	if err := e.closeExecutingCallbackParams(e.mcmsPackageID, ptb, executeCallback); err != nil {
		return fmt.Errorf("closing ExecutingCallbackParams vector: %w", err)
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
		"pop_back",
		[]transaction.TypeTag{*typeTag}, // Type arguments
		[]transaction.Argument{*vectorExecutingCallback}, // Arguments: just the vector
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
