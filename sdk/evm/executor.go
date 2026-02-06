package evm

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

// Executor is an Executor implementation for EVM chains, allowing for the execution of operations on the MCMS contract
type Executor struct {
	*Encoder
	*Inspector
	auth *bind.TransactOpts
}

// NewExecutor creates a new Executor for EVM chains
func NewExecutor(encoder *Encoder, client ContractDeployBackend, auth *bind.TransactOpts) *Executor {
	return &Executor{
		Encoder:   encoder,
		Inspector: NewInspector(client),
		auth:      auth,
	}
}

func (e *Executor) ExecuteOperation(
	ctx context.Context,
	metadata types.ChainMetadata,
	nonce uint32,
	proof []common.Hash,
	op types.Operation,
) (types.TransactionResult, error) {
	if e.Encoder == nil {
		return types.TransactionResult{}, errors.New("failed to create sdk.Executor - encoder (sdk.Encoder) is nil")
	}

	bindOp, err := e.ToGethOperation(nonce, metadata, op)
	if err != nil {
		return types.TransactionResult{}, err
	}

	opts := *e.auth
	opts.Context = ctx

	// Pre-pack calldata so we always have something to return on failure.
	// This is useful if the tx fails to give the user the calldata to retry or simulate.
	mcmsAddr := common.HexToAddress(metadata.MCMAddress)
	txPreview, err := buildExecuteTxData(&opts, mcmsAddr, bindOp, transformHashes(proof))
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to build execute call data: %w", err)
	}

	mcmsC, err := bindings.NewManyChainMultiSig(common.HexToAddress(metadata.MCMAddress), e.client)
	if err != nil {
		return types.TransactionResult{}, err
	}

	tx, err := mcmsC.Execute(&opts, bindOp, transformHashes(proof))
	if err != nil {
		// Extract timelock address and call data from the operation for bypass error handling
		timelockAddr := common.HexToAddress(op.Transaction.To)
		timelockCallData := op.Transaction.Data
		execErr := BuildExecutionError(ctx, err, txPreview, &opts, mcmsAddr, e.client, timelockAddr, timelockCallData)

		return types.TransactionResult{
			ChainFamily: chainsel.FamilyEVM,
		}, execErr
	}

	return types.TransactionResult{
		Hash:        tx.Hash().Hex(),
		ChainFamily: chainsel.FamilyEVM,
		RawData:     tx,
	}, err
}

func (e *Executor) SetRoot(
	ctx context.Context,
	metadata types.ChainMetadata,
	proof []common.Hash,
	root [32]byte,
	validUntil uint32,
	sortedSignatures []types.Signature,
) (types.TransactionResult, error) {
	if e.Encoder == nil {
		return types.TransactionResult{}, errors.New("failed to create sdk.Executor - encoder (sdk.Encoder) is nil")
	}

	bindMeta, err := e.ToGethRootMetadata(metadata)
	if err != nil {
		return types.TransactionResult{}, err
	}

	opts := *e.auth
	opts.Context = ctx

	// Pre-pack calldata so we always have something to return on failure.
	// This is useful if the tx fails to give the user the calldata to retry or simulate.
	mcmsAddr := common.HexToAddress(metadata.MCMAddress)
	txPreview, err := buildSetRootCallData(
		&opts,
		mcmsAddr,
		root,
		validUntil,
		bindMeta,
		transformHashes(proof),
		transformSignatures(sortedSignatures))
	if err != nil {
		return types.TransactionResult{}, err
	}

	mcmsC, err := bindings.NewManyChainMultiSig(common.HexToAddress(metadata.MCMAddress), e.client)
	if err != nil {
		return types.TransactionResult{}, err
	}

	tx, err := mcmsC.SetRoot(
		&opts,
		root,
		validUntil,
		bindMeta,
		transformHashes(proof),
		transformSignatures(sortedSignatures),
	)
	if err != nil {
		// SetRoot doesn't involve timelock, so pass empty values
		execErr := BuildExecutionError(ctx, err, txPreview, &opts, mcmsAddr, e.client, common.Address{}, nil)
		return types.TransactionResult{
			ChainFamily: chainsel.FamilyEVM,
		}, execErr
	}

	return types.TransactionResult{
		Hash:        tx.Hash().Hex(),
		ChainFamily: chainsel.FamilyEVM,
		RawData:     tx,
	}, err
}

// buildExecuteCallData packs calldata for ManyChainMultiSig.execute(...)
func buildExecuteTxData(
	opts *bind.TransactOpts,
	mcmsAddr common.Address,
	op bindings.ManyChainMultiSigOp,
	proof [][32]byte,
) (*gethtypes.Transaction, error) {
	abi, err := bindings.ManyChainMultiSigMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	data, err := abi.Pack("execute", op, proof)
	if err != nil {
		return nil, err
	}
	tx := buildUnsignedTxFromOpts(opts, mcmsAddr, data)

	return tx, nil
}

// buildSetRootCallData packs call data for ManyChainMultiSig.setRoot(...)
func buildSetRootCallData(
	opts *bind.TransactOpts,
	mcmsAddr common.Address,
	root [32]byte,
	validUntil uint32,
	meta bindings.ManyChainMultiSigRootMetadata,
	metaProof [][32]byte,
	sigs []bindings.ManyChainMultiSigSignature,
) (*gethtypes.Transaction, error) {
	abi, err := bindings.ManyChainMultiSigMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	data, err := abi.Pack("setRoot", root, validUntil, meta, metaProof, sigs)
	if err != nil {
		return nil, err
	}
	tx := buildUnsignedTxFromOpts(opts, mcmsAddr, data)

	return tx, nil
}

func buildUnsignedTxFromOpts(
	opts *bind.TransactOpts,
	to common.Address,
	data []byte,
) *gethtypes.Transaction {
	// Read gas/fee/nonce from opts when present; fall back to sane zeroes for a portable payload.
	nonce := uint64(0)
	if opts != nil && opts.Nonce != nil {
		nonce = opts.Nonce.Uint64()
	}

	var gas uint64
	var gasPrice, gasTipCap, gasFeeCap *big.Int

	if opts != nil {
		gas = opts.GasLimit
		gasPrice = opts.GasPrice   // legacy
		gasTipCap = opts.GasTipCap // 1559
		gasFeeCap = opts.GasFeeCap // 1559
	}

	// Prefer DynamicFee if FeeCap/TipCap are provided (EIP-1559). Otherwise fall back to Legacy.
	if gasFeeCap != nil || gasTipCap != nil {
		// Ensure non-nil fee pointers for DynamicFeeTx
		if gasFeeCap == nil {
			gasFeeCap = big.NewInt(0)
		}
		if gasTipCap == nil {
			gasTipCap = big.NewInt(0)
		}

		tx := gethtypes.NewTx(&gethtypes.DynamicFeeTx{
			Nonce:     nonce,
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
			Gas:       gas,
			To:        &to,
			Data:      data,
		})

		return tx
	}

	if gasPrice == nil {
		gasPrice = big.NewInt(0)
	}
	tx := gethtypes.NewTx(&gethtypes.LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Gas:      gas,
		GasPrice: gasPrice,
		Data:     data,
	})

	return tx
}
