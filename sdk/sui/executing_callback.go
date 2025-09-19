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
	feequoter "github.com/smartcontractkit/chainlink-sui/bindings/generated/ccip/ccip/fee_quoter"
	modulemcms "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms"
	module_mcms_deployer "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms_deployer"
	bindutils "github.com/smartcontractkit/chainlink-sui/bindings/utils"
)

type ExecutingCallbackAppender interface {
	AppendPTB(ctx context.Context, ptb *transaction.Transaction, executeCallback *transaction.Argument, calls []Call, mainStateObj string) error
}

type ExecutingCallbackParams struct {
	client                         sui.ISuiAPI
	mcms                           modulemcms.IMcms
	mcmsPackageID                  string
	entryPointContractEncoder      feequoter.FeeQuoterEncoder // Any contract implementing the `mcms_` entrypoint fn
	registryObj                    string
	accountObj                     string
	extractExecutingCallbackParams func(mcmsPackageID string, ptb *transaction.Transaction, vectorExecutingCallback *transaction.Argument) (*transaction.Argument, error)
	closeExecutingCallbackParams   func(mcmsPackageID string, ptb *transaction.Transaction, vectorExecutingCallback *transaction.Argument) error
}

func NewExecutingCallbackParams(client sui.ISuiAPI, mcms modulemcms.IMcms, mcmsPackageID string, entryPointContractEncoder feequoter.FeeQuoterEncoder, registryObj string, accountObj string) *ExecutingCallbackParams {
	return &ExecutingCallbackParams{
		client:                         client,
		mcms:                           mcms,
		mcmsPackageID:                  mcmsPackageID,
		entryPointContractEncoder:      entryPointContractEncoder,
		registryObj:                    registryObj,
		accountObj:                     accountObj,
		extractExecutingCallbackParams: extractExecutingCallbackParams,
		closeExecutingCallbackParams:   closeExecutingCallbackParams,
	}
}

func (e *ExecutingCallbackParams) AppendPTB(ctx context.Context, ptb *transaction.Transaction, executeCallback *transaction.Argument, calls []Call, mainStateObj string) error {
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
				// For mcms_deployer calls, use ExecuteDispatchToDeployer
				// call.StateObj contains Registry (from InternalStateObjects[0])
				// mainStateObj contains DeployerState (from main StateObj)
				executeDispatchCall, err := e.mcms.Encoder().ExecuteDispatchToDeployerWithArgs(
					call.StateObj, // Registry object
					mainStateObj,  // DeployerState object
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

				// If this is an upgrade call, complete the atomic upgrade flow
				if call.FunctionName == "authorize_upgrade" && len(call.CompiledModules) > 0 {
					// Increase gas budget for the upgrade steps
					ptb.SetGasBudget(500_000_000)

					// Step 2: Use the UpgradeTicket in package upgrade → produces UpgradeReceipt
					upgradeReceiptArg := ptb.Upgrade(
						call.CompiledModules,                     // Raw bytes (from Call)
						call.Dependencies,                        // Dependencies as addresses (from Call)
						models.SuiAddress(call.PackageToUpgrade), // Package being upgraded (from Call)
						*upgradeTicketArg,                        // UpgradeTicket from authorize step
					)

					deployerContract, err := module_mcms_deployer.NewMcmsDeployer(e.mcmsPackageID, e.client)
					if err != nil {
						return fmt.Errorf("failed to create deployer contract: %w", err)
					}

					commitEncoded, err := deployerContract.Encoder().CommitUpgradeWithArgs(
						bind.Object{Id: mainStateObj}, // DeployerState
						upgradeReceiptArg,             // UpgradeReceipt
					)
					if err != nil {
						return fmt.Errorf("failed to encode commit upgrade: %w", err)
					}

					_, err = deployerContract.Bound().AppendPTB(ctx, opts, ptb, commitEncoded)
					if err != nil {
						return fmt.Errorf("failed to append commit upgrade to PTB: %w", err)
					}
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

				// Add the call to the PTB (no hot potato expected)
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

			// Validate that StateObj is not empty before proceeding
			if call.StateObj == "" {
				return fmt.Errorf("call.StateObj is empty for call %d, cannot create mcms_entrypoint call", i)
			}

			// Prepare the mcms_entrypoint like call. We can use any function to build the call
			// Special handling for onramp mcms_accept_ownership: needs CCIPObjectRef + OnRampState
			var secondArg string
			if call.ModuleName == "onramp" && call.FunctionName == "mcms_accept_ownership" {
				// For onramp: call.StateObj = CCIPObjectRef, need OnRampState as 2nd arg
				// OnRampState is passed as mainStateObj parameter
				secondArg = mainStateObj
			} else {
				secondArg = e.registryObj // Standard pattern
			}

			entryPointCall, err := e.entryPointContractEncoder.McmsApplyFeeTokenUpdatesWithArgs(
				call.StateObj,
				secondArg,
				executingCallbackParams,
			)
			if err != nil {
				return fmt.Errorf("failed to create mcms_entrypoint call: %w", err)
			}
			// Override the module info with the actual target
			entryPointCall.Module.ModuleName = call.ModuleName
			entryPointCall.Module.PackageID = targetString
			// Set the function name - if it already has mcms_ prefix, use as-is
			if strings.HasPrefix(call.FunctionName, "mcms_") {
				entryPointCall.Function = call.FunctionName
			} else {
				// mcms entrypoint like functions are the target function prefixed with `mcms_`
				entryPointCall.Function = fmt.Sprintf("mcms_%s", call.FunctionName)
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
