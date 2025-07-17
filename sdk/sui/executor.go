package sui

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/block-vision/sui-go-sdk/models"
	"github.com/block-vision/sui-go-sdk/sui"
	"github.com/block-vision/sui-go-sdk/transaction"
	"github.com/ethereum/go-ethereum/common"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
	module_mcms "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms"
	bindutils "github.com/smartcontractkit/chainlink-sui/bindings/utils"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

const (
	SignatureVOffset    = 27
	SignatureVThreshold = 2

	ChunkSizeBytes = 50_000
)

var _ sdk.Executor = &Executor{}

type Executor struct {
	*Encoder
	*Inspector
	client        sui.ISuiAPI
	signer        bindutils.SuiSigner
	mcmsPackageId string
	mcms          *module_mcms.McmsContract

	accountObj  string
	registryObj string
}

func NewExecutor(client sui.ISuiAPI, signer bindutils.SuiSigner, encoder *Encoder, mcmsPackageId string, role TimelockRole, accountObj string, registryObj string) (*Executor, error) {
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
		accountObj:    accountObj,
		registryObj:   registryObj,
	}, nil
}

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
	chainIDBig := big.NewInt(int64(chainID))

	proofBytes := make([][]byte, len(proof))
	for i, hash := range proof {
		proofBytes[i] = hash.Bytes()
	}

	stateObj := bind.Object{Id: metadata.MCMAddress}
	clockObj := bind.Object{Id: "0x6"} // Clock object ID in Sui

	opts := &bind.CallOpts{
		Signer:           e.signer,
		WaitForExecution: true,
	}

	zeroAddress, err := AddressFromHex("0x0")
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to decode mcms package ID: %w", err)
	}
	toAddress, err := AddressFromHex(op.Transaction.To)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to parse To address %q: %w", op.Transaction.To, err)
	}

	// Decode calls from transaction data
	calls, err := DeserializeTimelockBypasserExecuteBatch(op.Transaction.Data)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to deserialize timelock bypasser execute batch: %w", err)
	}

	executeCall, err := e.mcms.Encoder().Execute(
		stateObj,
		clockObj,
		additionalFieldsMetadata.Role.Byte(),
		chainIDBig,
		// TODO: this is an issue in the contract. Hardcoded to zero address for now.
		zeroAddress.Bytes(), // Needs to always be MCMS package id
		uint64(nonce),
		toAddress.Bytes(), // Needs to always be MCMS pacakge id
		additionalFields.ModuleName,
		additionalFields.Function, // Can only be one of the dispatch
		op.Transaction.Data,       // For timelock, data is the collection of every call we want to execute, including module, function and data
		proofBytes,
	)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("executing operation on Sui mcms contract: %w", err)
	}

	// The execution needs to go in hand with the timelock operation in the same PTB transaction
	// ptb := transaction.NewTransaction()

	// // Build the execute call in the PTB
	ptb, timelockCallback, err := e.mcms.BuildPTBFromEncodedCall(ctx, opts, executeCall)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("building PTB for execute call: %w", err)
	}

	// Now build the timelock call using the result from the execute call
	var timelockCall *bind.EncodedCall
	encoder := e.mcms.Encoder()
	switch additionalFields.Function {
	case "timelock_schedule_batch":
		// timelockCall, err = encoder.DispatchTimelockScheduleBatchWithArgs(stateObj, "0x6", timelockCallback)
		return types.TransactionResult{}, fmt.Errorf("timelock action not available yet: %s", additionalFields.Function)
	case "timelock_cancel":
		// timelockCall, err = encoder.DispatchTimelockCancelWithArgs(timelockCallback)
		return types.TransactionResult{}, fmt.Errorf("timelock action not available yet: %s", additionalFields.Function)
	case "timelock_bypasser_execute_batch":
		// This returns []ExecutingCallbackParams. A set of inidividual calls that can be executed, either through `execute_dispatch_to_account` if it's an MCMS operation, or `mcms_entrypoint` of the destination contract
		timelockCall, err = encoder.DispatchTimelockBypasserExecuteBatchWithArgs(timelockCallback)
	default:
		return types.TransactionResult{}, fmt.Errorf("unsupported timelock action: %s", additionalFields.Function)
	}
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("creating timelock call: %w", err)
	}

	// Add the timelock call to the same PTB
	// If bypass, this a set of execute callbacks
	executeCallback, err := e.mcms.ExtendPTB(ctx, ptb, opts, timelockCall)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("building PTB for timelock call: %w", err)
	}

	objResolver := bind.NewObjectResolver(e.client)
	registryResolved, err := objResolver.ResolveObject(ctx, e.registryObj)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to resolve registry object: %w", err)
	}
	registryObjArg, err := objResolver.CreateObjectArgWithMutability(registryResolved, true)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to create object arg for registry: %w", err)
	}
	registryArg := ptb.Object(transaction.CallArg{
		Object: registryObjArg,
	})

	accountResolved, err := objResolver.ResolveObject(ctx, e.accountObj)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to resolve account object: %w", err)
	}
	accountObjArg, err := objResolver.CreateObjectArgWithMutability(accountResolved, true)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to create object arg for account: %w", err)
	}
	accountArg := ptb.Object(transaction.CallArg{
		Object: accountObjArg,
	})

	// Process each ExecutingCallbackParams individually
	// We iterate in forward order when using borrow with index
	for i, call := range calls {

		targetString := fmt.Sprintf("0x%x", call.Target)
		isTargetMCMSPackage := targetString == e.mcmsPackageId

		if isTargetMCMSPackage {
			// This is an MCMS operation, use ExecuteDispatchToAccount
			fmt.Printf("  - Processing as MCMS operation via ExecuteDispatchToAccount\n")

			// Create the index argument as a transaction.Argument
			indexArg := ptb.Pure(uint64(i))

			// we can just call the default program to access swap remove
			singleExecuteParam, err := e.mcms.Encoder().GetSingleExecuteCallbackParamsWithArgs(executeCallback, indexArg)
			if err != nil {
				return types.TransactionResult{}, fmt.Errorf("getting single execute callback params %d: %w", i, err)
			}

			callbackElement, err := e.mcms.ExtendPTB(ctx, ptb, opts, singleExecuteParam)
			if err != nil {
				return types.TransactionResult{}, fmt.Errorf("extending PTB with single execute callback params %d: %w", i, err)
			}

			// Create a new Argument that references the first element of the tuple result
			// The callbackElement.Result should contain the command index from the GetSingleExecuteCallbackParams call
			var commandIndex uint16
			if callbackElement.Result != nil {
				commandIndex = *callbackElement.Result
			} else {
				return types.TransactionResult{}, fmt.Errorf("callbackElement does not have a Result field set")
			}

			// Create a NestedResult argument to get the first element (index 0) of the tuple
			executingCallbackParams := transaction.Argument{
				NestedResult: &transaction.NestedResult{
					Index:       commandIndex,
					ResultIndex: 0, // First element of the tuple
				},
			}

			executeDispatchCall, err := e.mcms.Encoder().ExecuteDispatchToAccountWithArgs(
				registryArg,
				accountArg,
				executingCallbackParams,
			)
			if err != nil {
				return types.TransactionResult{}, fmt.Errorf("creating ExecuteDispatchToAccount call %d: %w", i, err)
			}

			// Add the call to the PTB
			_, err = e.mcms.ExtendPTB(ctx, ptb, opts, executeDispatchCall)
			if err != nil {
				return types.TransactionResult{}, fmt.Errorf("adding ExecuteDispatchToAccount call %d to PTB: %w", i, err)
			}

			fmt.Printf("  - Successfully added ExecuteDispatchToAccount call to PTB\n")

		} else {
			// This is a destination contract operation, use mcms_entrypoint
			fmt.Printf("  - Processing as destination contract operation via mcms_entrypoint\n")

			// Extract the element at index i from executeCallback vector using PTB vector borrow
			// We need to create the proper type tag for ExecutingCallbackParams
			executingCallbackParamsType := fmt.Sprintf("%s::mcms_registry::ExecutingCallbackParams", e.mcmsPackageId)

			// Convert the type string to TypeTag
			typeTag, err := bindutils.ConvertTypeStringToTypeTag(executingCallbackParamsType)
			if err != nil {
				return types.TransactionResult{}, fmt.Errorf("failed to convert type string to TypeTag: %w", err)
			}

			// Create the vector borrow call to get the element at index i by reference
			// This gives us a reference to the ExecutingCallbackParams
			indexArg := ptb.Pure(uint64(i))
			callbackElement := ptb.MoveCall(
				"0x1", // Standard library package for vector operations
				"vector",
				"borrow",
				[]transaction.TypeTag{*typeTag}, // Type arguments
				[]transaction.Argument{*executeCallback, indexArg}, // Arguments: vector and index
			)

			fmt.Printf("  - Extracted ExecutingCallbackParams[%d] from vector\n", i)

			// For destination contract operations, we need to call the mcms_entrypoint function
			// on the destination contract with the ExecutingCallbackParams
			fmt.Printf("  - Calling mcms_entrypoint on target contract: %s\n", targetString)

			// Call the mcms_entrypoint function on the destination contract
			ptb.MoveCall(
				models.SuiAddress(targetString), // The destination contract package
				call.ModuleName,
				"mcms_entrypoint",                       // Standard MCMS entrypoint function
				[]transaction.TypeTag{},                 // No type arguments
				[]transaction.Argument{callbackElement}, // Arguments: ExecutingCallbackParams
			)

			fmt.Printf("  - Successfully added mcms_entrypoint call to PTB\n")
		}

		fmt.Printf("  - Processed ExecutingCallbackParams[%d] for %s::%s\n", i, call.ModuleName, call.FunctionName)
	}

	// For now, log the completion of processing
	fmt.Printf("Completed processing %d ExecutingCallbackParams\n", len(calls))

	// Execute the complete PTB with both calls
	tx, err := bind.ExecutePTB(ctx, opts, e.client, ptb)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("Op execution with PTB failed: %w", err)
	}

	txJSON, err := json.MarshalIndent(tx, "", "  ")
	fmt.Printf("TX JSON:\n%s", string(txJSON))

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
	chainIDBig := big.NewInt(int64(chainID))

	proofBytes := make([][]byte, len(proof))
	for i, hash := range proof {
		proofBytes[i] = hash.Bytes()
	}
	signatures := encodeSignatures(sortedSignatures)

	opts := &bind.CallOpts{
		Signer:           e.signer,
		WaitForExecution: true,
	}

	stateObj := bind.Object{Id: metadata.MCMAddress}
	clockObj := bind.Object{Id: "0x6"} // Clock object ID in Sui

	zeroAddress, err := AddressFromHex("0x0")
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to decode mcms package ID: %w", err)
	}
	tx, err := e.mcms.SetRoot(
		ctx,
		opts,
		stateObj,
		clockObj,
		additionalFieldsMetadata.Role.Byte(),
		root[:],
		uint64(validUntil)*1000, // the contract expects milliseconds
		chainIDBig,
		// TODO: this is an issue in the contract. Hardcoded to zero address for now.
		zeroAddress.Bytes(),
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
		if len(r) < 32 {
			padded := make([]byte, 32)
			copy(padded[32-len(r):], r)
			r = padded
		}

		// Pad S to 32 bytes if needed
		if len(s) < 32 {
			padded := make([]byte, 32)
			copy(padded[32-len(s):], s)
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
