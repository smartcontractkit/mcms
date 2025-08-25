package sui

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/block-vision/sui-go-sdk/sui"
	"github.com/block-vision/sui-go-sdk/transaction"
	"github.com/ethereum/go-ethereum/common"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
	module_fee_quoter "github.com/smartcontractkit/chainlink-sui/bindings/generated/ccip/ccip/fee_quoter"
	module_mcms "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms"
	bindutils "github.com/smartcontractkit/chainlink-sui/bindings/utils"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

const (
	// EthereumSignatureLength represents the byte length for signature components
	EthereumSignatureLength = 32
	SignatureVOffset        = 27
	SignatureVThreshold     = 2
)

var _ sdk.Executor = &Executor{}

type Executor struct {
	*Encoder
	*Inspector
	client        sui.ISuiAPI
	signer        bindutils.SuiSigner
	mcmsPackageId string
	mcms          *module_mcms.McmsContract

	mcmsObj     string // MultisigState object ID
	accountObj  string
	registryObj string
	timelockObj string
}

func NewExecutor(client sui.ISuiAPI, signer bindutils.SuiSigner, encoder *Encoder, mcmsPackageId string, role TimelockRole, mcmsObj string, accountObj string, registryObj string, timelockObj string) (*Executor, error) {
	mcms, err := module_mcms.NewMcms(mcmsPackageId, client)
	if err != nil {
		return nil, err
	}

	inspector, err := NewInspector(client, signer, mcmsPackageId, role)
	if err != nil {
		return nil, err
	}

	return &Executor{
		Encoder:       encoder,
		Inspector:     inspector,
		client:        client,
		signer:        signer,
		mcmsPackageId: mcmsPackageId,
		mcms:          mcms,
		mcmsObj:       mcmsObj,
		accountObj:    accountObj,
		registryObj:   registryObj,
		timelockObj:   timelockObj,
	}, nil
}

// TODO: As the contracts are structured, we can't select the operation to execute, as the batch is inside the Op
func (e Executor) ExecuteOperation(
	ctx context.Context,
	metadata types.ChainMetadata,
	nonce uint32,
	proof []common.Hash,
	op types.Operation,
) (types.TransactionResult, error) {
	var additionalFields AdditionalFields
	if err := json.Unmarshal(op.Transaction.AdditionalFields, &additionalFields); err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to unmarshal additional fields: %w", err)
	}

	var additionalFieldsMetadata AdditionalFieldsMetadata
	if len(metadata.AdditionalFields) > 0 {
		if err := json.Unmarshal(metadata.AdditionalFields, &additionalFieldsMetadata); err != nil {
			return types.TransactionResult{}, fmt.Errorf("failed to unmarshal additional fields metadata: %w", err)
		}
	}

	chainID, err := chain_selectors.SuiChainIdFromSelector(uint64(e.ChainSelector))
	if err != nil {
		return types.TransactionResult{}, err
	}
	chainIDBig := new(big.Int).SetUint64(chainID)

	proofBytes := make([][]byte, len(proof))
	for i, hash := range proof {
		proofBytes[i] = hash.Bytes()
	}

	stateObj := bind.Object{Id: e.mcmsObj}
	clockObj := bind.Object{Id: "0x6"} // Clock object ID in Sui

	opts := &bind.CallOpts{
		Signer:           e.signer,
		WaitForExecution: true,
	}

	toAddress, err := AddressFromHex(op.Transaction.To)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to parse To address %q: %w", op.Transaction.To, err)
	}

	// If it's a bypass, op.Transaction.Data includes the state obj, but the contract doesn't need it
	data := op.Transaction.Data
	if additionalFields.Function == TimelockActionBypass {
		data, err = RemoveStateObjectsFromBypassData(op.Transaction.Data)
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("failed to remove state objects from bypass data: %w", err)
		}
	}

	executeCall, err := e.mcms.Encoder().Execute(
		stateObj,
		clockObj,
		additionalFieldsMetadata.Role.Byte(),
		chainIDBig,
		e.mcmsPackageId,
		uint64(nonce),
		toAddress.Hex(), // Needs to always be MCMS package id
		additionalFields.ModuleName,
		additionalFields.Function, // Can only be one of the dispatch
		data,                      // For timelock, data is the collection of every call we want to execute, including module, function and data
		proofBytes,
	)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("executing operation on Sui mcms contract: %w", err)
	}

	ptb := transaction.NewTransaction()
	// The execution needs to go in hand with the timelock operation in the same PTB transaction
	timelockCallback, err := e.mcms.AppendPTB(ctx, opts, ptb, executeCall)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("building PTB for execute call: %w", err)
	}
	// Now build the timelock call using the result from the execute call
	encoder := e.mcms.Encoder()
	if additionalFields.Function != TimelockActionBypass && additionalFields.Function != TimelockActionSchedule && additionalFields.Function != TimelockActionCancel {
		return types.TransactionResult{}, fmt.Errorf("unsupported timelock action: %s", additionalFields.Function)
	}

	if additionalFields.Function == TimelockActionSchedule {
		timelockCall, encodeErr := encoder.DispatchTimelockScheduleBatchWithArgs(e.timelockObj, "0x6", timelockCallback)
		if encodeErr != nil {
			return types.TransactionResult{}, fmt.Errorf("creating timelock call: %w", encodeErr)
		}
		_, err = e.mcms.AppendPTB(ctx, opts, ptb, timelockCall)
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("adding timelock call to PTB: %w", err)
		}
	}

	if additionalFields.Function == TimelockActionCancel {
		// TODO: Implement cancel functionality when needed
		return types.TransactionResult{}, fmt.Errorf("cancel functionality not yet implemented")
	}

	if additionalFields.Function == TimelockActionBypass {
		timelockCall, timelockErr := encoder.DispatchTimelockBypasserExecuteBatchWithArgs(timelockCallback)
		if timelockErr != nil {
			return types.TransactionResult{}, fmt.Errorf("creating timelock call: %w", timelockErr)
		}

		// Add the timelock call to the same PTB
		// If bypass, this a set of execute callbacks
		executeCallback, extendCallbackErr := e.mcms.AppendPTB(ctx, opts, ptb, timelockCall)
		if extendCallbackErr != nil {
			return types.TransactionResult{}, fmt.Errorf("building PTB for timelock call: %w", extendCallbackErr)
		}
		// Decode calls from transaction data
		calls, err := DeserializeTimelockBypasserExecuteBatch(op.Transaction.Data)
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("failed to deserialize timelock bypasser execute batch: %w", err)
		}

		if extendErr := AppendPTBFromExecutingCallbackParams(ctx, e.client, e.mcms, ptb, e.mcmsPackageId, executeCallback, calls, e.registryObj, e.accountObj); extendErr != nil {
			return types.TransactionResult{}, fmt.Errorf("extending PTB from executing callback params: %w", extendErr)
		}
	}
	// Execute the complete PTB with every call
	tx, err := bind.ExecutePTB(ctx, opts, e.client, ptb)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("op execution with PTB failed: %w", err)
	}

	return types.TransactionResult{
		Hash:        tx.Digest,
		ChainFamily: chain_selectors.FamilySui,
		RawData:     tx,
	}, nil
}

func (e Executor) SetRoot(
	ctx context.Context,
	metadata types.ChainMetadata,
	proof []common.Hash,
	root [32]byte,
	validUntil uint32,
	sortedSignatures []types.Signature,
) (types.TransactionResult, error) {
	var additionalFieldsMetadata AdditionalFieldsMetadata
	if len(metadata.AdditionalFields) > 0 {
		if err := json.Unmarshal(metadata.AdditionalFields, &additionalFieldsMetadata); err != nil {
			return types.TransactionResult{}, fmt.Errorf("failed to unmarshal additional fields metadata: %w", err)
		}
	}

	chainID, err := chain_selectors.SuiChainIdFromSelector(uint64(e.ChainSelector))
	if err != nil {
		return types.TransactionResult{}, err
	}
	chainIDBig := new(big.Int).SetUint64(chainID)

	proofBytes := make([][]byte, len(proof))
	for i, hash := range proof {
		proofBytes[i] = hash.Bytes()
	}
	signatures := encodeSignatures(sortedSignatures)

	opts := &bind.CallOpts{
		Signer:           e.signer,
		WaitForExecution: true,
	}

	stateObj := bind.Object{Id: e.mcmsObj} // Use stored MultisigState object ID
	clockObj := bind.Object{Id: "0x6"}     // Clock object ID in Sui

	tx, err := e.mcms.SetRoot(
		ctx,
		opts,
		stateObj,
		clockObj,
		additionalFieldsMetadata.Role.Byte(),
		root[:],
		uint64(validUntil), // the contract expects seconds
		chainIDBig,
		// Use the actual MCMS package address
		e.mcmsPackageId,
		metadata.StartingOpCount,
		metadata.StartingOpCount+e.TxCount,
		e.OverridePreviousRoot,
		proofBytes,
		signatures,
	)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("setting root on Sui mcms contract: %w", err)
	}

	return types.TransactionResult{
		Hash:        tx.Digest,
		ChainFamily: chain_selectors.FamilySui,
		RawData:     tx,
	}, nil
}

func encodeSignatures(signatures []types.Signature) [][]byte {
	sigs := make([][]byte, len(signatures))
	for i, signature := range signatures {
		// Ensure R and S are exactly 32 bytes each
		r := signature.R.Bytes()
		s := signature.S.Bytes()

		// Pad R to 32 bytes if needed
		if len(r) < EthereumSignatureLength {
			padded := make([]byte, EthereumSignatureLength)
			copy(padded[EthereumSignatureLength-len(r):], r)
			r = padded
		}

		// Pad S to 32 bytes if needed
		if len(s) < EthereumSignatureLength {
			padded := make([]byte, EthereumSignatureLength)
			copy(padded[EthereumSignatureLength-len(s):], s)
			s = padded
		}

		sigs[i] = append(r, s...)
		if signature.V <= SignatureVThreshold {
			sigs[i] = append(sigs[i], signature.V+SignatureVOffset)
		} else {
			sigs[i] = append(sigs[i], signature.V)
		}
	}

	return sigs
}

func extractExecutingCallbackParams(mcmsPackageId string, ptb *transaction.Transaction, vectorExecutingCallback *transaction.Argument) (*transaction.Argument, error) {
	// Convert the type string to TypeTag
	executingCallbackParamsType := fmt.Sprintf("%s::mcms_registry::ExecutingCallbackParams", mcmsPackageId)
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

func closeExecutingCallbackParams(mcmsPackageId string, ptb *transaction.Transaction, vectorExecutingCallback *transaction.Argument) error {
	// Get the type tag for ExecutingCallbackParams
	executingCallbackParamsType := fmt.Sprintf("%s::mcms_registry::ExecutingCallbackParams", mcmsPackageId)
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

func AppendPTBFromExecutingCallbackParams(
	ctx context.Context,
	client sui.ISuiAPI,
	mcms *module_mcms.McmsContract,
	ptb *transaction.Transaction,
	mcmsPackageId string,
	executeCallback *transaction.Argument,
	calls []Call,
	registryObj string,
	accountObj string,
) error {
	// Only used for object resolving caching
	opts := &bind.CallOpts{}
	// Process each ExecutingCallbackParams individually
	// We need to process them in reverse order since we're using pop_back to extract elements
	for i := len(calls) - 1; i >= 0; i-- {
		call := calls[i]

		// Ensure proper address formatting - convert bytes to hex with proper padding
		targetString := fmt.Sprintf("0x%s", strings.ToLower(hex.EncodeToString(call.Target)))
		isTargetMCMSPackage := targetString == mcmsPackageId

		// If the target is the MCMS package, we need to call ExecuteDispatchToAccount
		if isTargetMCMSPackage {
			// We need to extract individual ExecutingCallbackParams from the executeCallback vector
			executingCallbackParams, extractErr := extractExecutingCallbackParams(mcmsPackageId, ptb, executeCallback)
			if extractErr != nil {
				return fmt.Errorf("failed to extract executing callback params: %w", extractErr)
			}
			executeDispatchCall, err := mcms.Encoder().ExecuteDispatchToAccountWithArgs(
				registryObj,
				accountObj,
				executingCallbackParams,
			)
			if err != nil {
				return fmt.Errorf("creating ExecuteDispatchToAccount call %d: %w", i, err)
			}

			// Add the call to the PTB
			_, err = mcms.AppendPTB(ctx, opts, ptb, executeDispatchCall)
			if err != nil {
				return fmt.Errorf("adding ExecuteDispatchToAccount call %d to PTB: %w", i, err)
			}
			// If this is a destination contract operation, we need to call mcms_entrypoint
		} else {
			executingCallbackParams, err := extractExecutingCallbackParams(mcmsPackageId, ptb, executeCallback)
			if err != nil {
				return fmt.Errorf("extracting ExecutingCallbackParams %d: %w", i, err)
			}

			// We can use any contract that implements the mcms_entrypoint function
			entryPointContract, err := module_fee_quoter.NewFeeQuoter(targetString, client)
			if err != nil {
				return fmt.Errorf("failed to create MCMS EntryPoint contract: %w", err)
			}

			// Prepare the mcms_entrypoint call
			entryPointCall, err := entryPointContract.Encoder().McmsEntrypointWithArgs(
				call.StateObj,
				registryObj,
				executingCallbackParams,
			)
			if err != nil {
				return fmt.Errorf("failed to create mcms_entrypoint call: %w", err)
			}
			// Override the module info with the actual target
			entryPointCall.Module.ModuleName = call.ModuleName

			_, err = mcms.AppendPTB(ctx, opts, ptb, entryPointCall)
			if err != nil {
				return fmt.Errorf("failed to append mcms_entrypoint call to PTB: %w", err)
			}
		}
	}

	// After processing all elements, the vector should be empty, we need to close it
	if err := closeExecutingCallbackParams(mcmsPackageId, ptb, executeCallback); err != nil {
		return fmt.Errorf("closing ExecutingCallbackParams vector: %w", err)
	}

	return nil
}
