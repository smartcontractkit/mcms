package sui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/block-vision/sui-go-sdk/models"
	"github.com/block-vision/sui-go-sdk/sui"
	"github.com/block-vision/sui-go-sdk/transaction"
	"github.com/ethereum/go-ethereum/common"

	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
	modulemcms "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms"
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
	mcmsPackageID string
	mcms          modulemcms.IMcms

	mcmsObj     string // MultisigState object ID
	accountObj  string
	registryObj string
	timelockObj string

	// ExecutePTB function for dependency injection and testing
	ExecutePTB func(ctx context.Context, opts *bind.CallOpts, client sui.ISuiAPI, ptb *transaction.Transaction) (*models.SuiTransactionBlockResponse, error)

	executingCallbackParams ExecutingCallbackAppender
}

func NewExecutor(client sui.ISuiAPI, signer bindutils.SuiSigner, encoder *Encoder, entrypointEncoder EntrypointArgEncoder, mcmsPackageID string, role TimelockRole, mcmsObj string, accountObj string, registryObj string, timelockObj string) (*Executor, error) {
	mcms, err := modulemcms.NewMcms(mcmsPackageID, client)
	if err != nil {
		return nil, err
	}

	inspector, err := NewInspector(client, signer, mcmsPackageID, role)
	if err != nil {
		return nil, err
	}

	executingCallbackParams := NewExecutingCallbackParams(client, mcms, mcmsPackageID, entrypointEncoder, registryObj, accountObj)

	return &Executor{
		Encoder:                 encoder,
		Inspector:               inspector,
		client:                  client,
		signer:                  signer,
		mcmsPackageID:           mcmsPackageID,
		mcms:                    mcms,
		mcmsObj:                 mcmsObj,
		accountObj:              accountObj,
		registryObj:             registryObj,
		timelockObj:             timelockObj,
		ExecutePTB:              bind.ExecutePTB, // Default implementation
		executingCallbackParams: executingCallbackParams,
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

	chainID, err := cselectors.SuiChainIdFromSelector(uint64(e.ChainSelector))
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

	executeCall, err := e.mcms.Encoder().Execute(
		stateObj,
		clockObj,
		additionalFieldsMetadata.Role.Byte(),
		chainIDBig,
		e.mcmsPackageID,
		uint64(nonce),
		toAddress.Hex(), // Needs to always be MCMS package id
		additionalFields.ModuleName,
		additionalFields.Function, // Can only be one of the dispatch
		op.Transaction.Data,       // For timelock, data is the collection of every call we want to execute, including module, function and data
		proofBytes,
	)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("executing operation on Sui mcms contract: %w", err)
	}

	ptb := transaction.NewTransaction()
	// The execution needs to go in hand with the timelock operation in the same PTB transaction
	timelockCallback, err := e.mcms.Bound().AppendPTB(ctx, opts, ptb, executeCall)
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
		_, err = e.mcms.Bound().AppendPTB(ctx, opts, ptb, timelockCall)
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("adding timelock call to PTB: %w", err)
		}
	}

	if additionalFields.Function == TimelockActionCancel {
		timelockCall, encodeErr := encoder.DispatchTimelockCancelWithArgs(e.timelockObj, timelockCallback)
		if encodeErr != nil {
			return types.TransactionResult{}, fmt.Errorf("creating timelock call: %w", encodeErr)
		}
		_, err = e.mcms.Bound().AppendPTB(ctx, opts, ptb, timelockCall)
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("adding timelock call to PTB: %w", err)
		}
	}

	if additionalFields.Function == TimelockActionBypass {
		timelockCall, timelockErr := encoder.DispatchTimelockBypasserExecuteBatchWithArgs(timelockCallback)
		if timelockErr != nil {
			return types.TransactionResult{}, fmt.Errorf("creating timelock call: %w", timelockErr)
		}

		// Add the timelock call to the same PTB
		// If bypass, this a set of execute callbacks
		executeCallback, extendCallbackErr := e.mcms.Bound().AppendPTB(ctx, opts, ptb, timelockCall)
		if extendCallbackErr != nil {
			return types.TransactionResult{}, fmt.Errorf("building PTB for timelock call: %w", extendCallbackErr)
		}
		// Decode calls from transaction data
		calls, desErr := deserializeTimelockBypasserExecuteBatch(op.Transaction.Data)
		if desErr != nil {
			return types.TransactionResult{}, fmt.Errorf("failed to deserialize timelock bypasser execute batch: %w", err)
		}
		if len(calls) != len(additionalFields.InternalStateObjects) {
			return types.TransactionResult{}, errors.New("mismatched call and state object count")
		}
		for i, call := range calls {
			calls[i] = Call{
				ModuleName:   call.ModuleName,
				FunctionName: call.FunctionName,
				StateObj:     additionalFields.InternalStateObjects[i],
				Data:         call.Data,
				Target:       call.Target,
				TypeArgs:     additionalFields.InternalTypeArgs[i],
			}
		}

		if extendErr := e.executingCallbackParams.AppendPTB(ctx, ptb, executeCallback, calls); extendErr != nil {
			return types.TransactionResult{}, fmt.Errorf("extending PTB from executing callback params: %w", extendErr)
		}
	}
	// Execute the complete PTB with every call
	tx, err := e.ExecutePTB(ctx, opts, e.client, ptb)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("op execution with PTB failed: %w", err)
	}

	return types.TransactionResult{
		Hash:        tx.Digest,
		ChainFamily: cselectors.FamilySui,
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

	chainID, err := cselectors.SuiChainIdFromSelector(uint64(e.ChainSelector))
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
		e.mcmsPackageID,
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
		ChainFamily: cselectors.FamilySui,
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
