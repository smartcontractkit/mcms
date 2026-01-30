package ton

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tlbe"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"
)

const DefaultWaitBuffer = 250 * time.Millisecond

// sdk.Executor implementation for TON chains, allowing for the execution of operations on the MCMS contract
type executor struct {
	sdk.Encoder
	sdk.Inspector

	client ton.APIClientWrapped
	wallet *wallet.Wallet

	// Transaction options
	amount tlb.Coins

	// Executor options
	wait bool
}

type ExecutorOpts struct {
	Encoder sdk.Encoder
	Client  ton.APIClientWrapped
	Wallet  *wallet.Wallet

	// Value to send (to MCMS) with message
	Amount tlb.Coins
	// Whether to wait until the pending operation is finalized before trying to execute
	WaitPending *bool // default: true
}

// NewExecutor creates a new Executor for TON chains
func NewExecutor(opts ExecutorOpts) (sdk.Executor, error) {
	if lo.IsNil(opts.Encoder) {
		return nil, errors.New("failed to create sdk.Executor - encoder (sdk.Encoder) is nil")
	}

	if lo.IsNil(opts.Client) {
		return nil, errors.New("failed to create sdk.Executor - client (ton.APIClientWrapped) is nil")
	}

	if opts.Wallet == nil {
		return nil, errors.New("failed to create sdk.Executor - wallet (*wallet.Wallet) is nil")
	}

	wait := true // default
	if opts.WaitPending != nil {
		wait = *opts.WaitPending
	}

	return &executor{
		Encoder:   opts.Encoder,
		Inspector: NewInspector(opts.Client),
		client:    opts.Client,
		wallet:    opts.Wallet,
		amount:    opts.Amount,
		wait:      wait,
	}, nil
}

func (e *executor) ExecuteOperation(
	ctx context.Context,
	metadata types.ChainMetadata,
	nonce uint32,
	proof []common.Hash,
	op types.Operation,
) (types.TransactionResult, error) {
	var z types.TransactionResult // zero value

	// Encode operation
	oe, ok := e.Encoder.(OperationEncoder[mcms.Op])
	if !ok {
		return z, errors.New("failed to assert OperationEncoder")
	}

	bindOp, err := oe.ToOperation(nonce, metadata, op)
	if err != nil {
		return z, fmt.Errorf("failed to convert to operation: %w", err)
	}

	// Encode proofs
	pe, ok := e.Encoder.(ProofEncoder[mcms.Proof])
	if !ok {
		return z, errors.New("failed to assert ProofEncoder")
	}

	bindProof, err := pe.ToProof(proof)
	if err != nil {
		return z, fmt.Errorf("failed to encode proof: %w", err)
	}

	qID, err := tvm.RandomQueryID()
	if err != nil {
		return z, fmt.Errorf("failed to generate random query ID: %w", err)
	}

	body, err := tlb.ToCell(mcms.Execute{
		QueryID: qID,

		Op:    bindOp,
		Proof: bindProof,
	})
	if err != nil {
		return z, fmt.Errorf("failed to encode ExecuteBatch body: %w", err)
	}

	// Map to Ton Address type
	dstAddr, err := address.ParseAddr(metadata.MCMAddress)
	if err != nil {
		return z, fmt.Errorf("invalid mcms address: %w", err)
	}

	// Check the status of potential pending operation
	// Get current block
	blockID, err := e.client.CurrentMasterchainInfo(ctx)
	if err != nil {
		return z, fmt.Errorf("failed to get current block: %w", err)
	}

	// Load the full block to get timestamp and hash
	block, err := e.client.GetBlockData(ctx, blockID)
	if err != nil {
		return z, fmt.Errorf("failed to get block data: %w", err)
	}

	// Load the current on-chain time
	now := block.BlockInfo.GenUtime

	info, err := tvm.CallGetter(ctx, e.client, blockID, dstAddr, mcms.GetOpPendingInfo)
	if err != nil {
		return z, fmt.Errorf("failed to call mcms.GetOpPendingInfo getter: %w", err)
	}

	tx := TxOpts{
		Wallet:  e.wallet,
		DstAddr: dstAddr,
		Amount:  e.amount,
		Body:    body,
	}

	return SendTxAfter(ctx, tx, uint64(now), info.ValidAfter, DefaultWaitBuffer, e.wait)
}

func (e *executor) SetRoot(
	ctx context.Context,
	metadata types.ChainMetadata,
	proof []common.Hash,
	root [32]byte,
	validUntil uint32,
	sortedSignatures []types.Signature,
) (types.TransactionResult, error) {
	var z types.TransactionResult // zero value

	// Encode root metadata
	rme, ok := e.Encoder.(RootMetadataEncoder[mcms.RootMetadata])
	if !ok {
		return z, errors.New("failed to assert RootMetadataEncoder")
	}

	rm, err := rme.ToRootMetadata(metadata)
	if err != nil {
		return z, fmt.Errorf("failed to convert to root metadata: %w", err)
	}

	// Map to Ton Address type
	dstAddr, err := address.ParseAddr(metadata.MCMAddress)
	if err != nil {
		return z, fmt.Errorf("invalid mcms address: %w", err)
	}

	// Encode proofs
	pe, ok := e.Encoder.(ProofEncoder[mcms.Proof])
	if !ok {
		return z, errors.New("failed to assert ProofEncoder")
	}

	bindProof, err := pe.ToProof(proof)
	if err != nil {
		return z, fmt.Errorf("failed to encode proof: %w", err)
	}

	// Encode signatures
	se, ok := e.Encoder.(SignaturesEncoder[mcms.Signature])
	if !ok {
		return z, errors.New("failed to assert SignatureEncoder")
	}

	bindSignatures, err := se.ToSignatures(sortedSignatures, root)
	if err != nil {
		return z, fmt.Errorf("failed to encode signatures: %w", err)
	}

	qID, err := tvm.RandomQueryID()
	if err != nil {
		return z, fmt.Errorf("failed to generate random query ID: %w", err)
	}

	body, err := tlb.ToCell(mcms.SetRoot{
		QueryID: qID,

		Root:       tlbe.NewUint256(new(big.Int).SetBytes(root[:])),
		ValidUntil: uint64(validUntil),
		Metadata:   rm,

		MetadataProof: bindProof,
		Signatures:    bindSignatures,
	})
	if err != nil {
		return z, fmt.Errorf("failed to encode ExecuteBatch body: %w", err)
	}

	// Check the status of potential pending operation
	// Get current block
	blockID, err := e.client.CurrentMasterchainInfo(ctx)
	if err != nil {
		return z, fmt.Errorf("failed to get current block: %w", err)
	}

	// Load the full block to get timestamp and hash
	block, err := e.client.GetBlockData(ctx, blockID)
	if err != nil {
		return z, fmt.Errorf("failed to get block data: %w", err)
	}

	// Load the current on-chain time
	now := block.BlockInfo.GenUtime

	info, err := tvm.CallGetter(ctx, e.client, blockID, dstAddr, mcms.GetOpPendingInfo)
	if err != nil {
		return z, fmt.Errorf("failed to call mcms.GetOpPendingInfo getter: %w", err)
	}

	tx := TxOpts{
		Wallet:  e.wallet,
		DstAddr: dstAddr,
		Amount:  e.amount,
		Body:    body,
	}

	return SendTxAfter(ctx, tx, uint64(now), info.ValidAfter, DefaultWaitBuffer, e.wait)
}
