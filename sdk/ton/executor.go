package ton

import (
	"context"
	"errors"
	"fmt"
	"math/big"

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

// sdk.Executor implementation for TON chains, allowing for the execution of operations on the MCMS contract
type executor struct {
	sdk.Encoder
	sdk.Inspector

	wallet *wallet.Wallet

	// Transaction opts
	amount tlb.Coins
}

// NewExecutor creates a new Executor for TON chains
func NewExecutor(encoder sdk.Encoder, client ton.APIClientWrapped, w *wallet.Wallet, amount tlb.Coins) (sdk.Executor, error) {
	if lo.IsNil(encoder) {
		return nil, errors.New("failed to create sdk.Executor - encoder (sdk.Encoder) is nil")
	}

	if lo.IsNil(client) {
		return nil, errors.New("failed to create sdk.Executor - client (ton.APIClientWrapped) is nil")
	}

	if w == nil {
		return nil, errors.New("failed to create sdk.Executor - wallet (*wallet.Wallet) is nil")
	}

	return &executor{
		Encoder:   encoder,
		Inspector: NewInspector(client),
		wallet:    w,
		amount:    amount,
	}, nil
}

func (e *executor) ExecuteOperation(
	ctx context.Context,
	metadata types.ChainMetadata,
	nonce uint32,
	proof []common.Hash,
	op types.Operation,
) (types.TransactionResult, error) {
	oe, ok := e.Encoder.(OperationEncoder[mcms.Op])
	if !ok {
		return types.TransactionResult{}, errors.New("failed to assert OperationEncoder")
	}

	bindOp, err := oe.ToOperation(nonce, metadata, op)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to convert to operation: %w", err)
	}

	// Encode proofs
	pe, ok := e.Encoder.(ProofEncoder[mcms.Proof])
	if !ok {
		return types.TransactionResult{}, errors.New("failed to assert ProofEncoder")
	}

	bindProof, err := pe.ToProof(proof)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to encode proof: %w", err)
	}

	qID, err := tvm.RandomQueryID()
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to generate random query ID: %w", err)
	}

	body, err := tlb.ToCell(mcms.Execute{
		QueryID: qID,

		Op:    bindOp,
		Proof: bindProof,
	})
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to encode ExecuteBatch body: %w", err)
	}

	// Map to Ton Address type
	dstAddr, err := address.ParseAddr(metadata.MCMAddress)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("invalid mcms address: %w", err)
	}

	skipSend := false // TODO: expose via executor options

	return SendTx(ctx, TxOpts{
		Wallet:   e.wallet,
		DstAddr:  dstAddr,
		Amount:   e.amount,
		Body:     body,
		SkipSend: skipSend,
	})
}

func (e *executor) SetRoot(
	ctx context.Context,
	metadata types.ChainMetadata,
	proof []common.Hash,
	root [32]byte,
	validUntil uint32,
	sortedSignatures []types.Signature,
) (types.TransactionResult, error) {
	rme, ok := e.Encoder.(RootMetadataEncoder[mcms.RootMetadata])
	if !ok {
		return types.TransactionResult{}, errors.New("failed to assert RootMetadataEncoder")
	}

	rm, err := rme.ToRootMetadata(metadata)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to convert to root metadata: %w", err)
	}

	// Map to Ton Address type
	dstAddr, err := address.ParseAddr(metadata.MCMAddress)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("invalid mcms address: %w", err)
	}

	// Encode proofs
	pe, ok := e.Encoder.(ProofEncoder[mcms.Proof])
	if !ok {
		return types.TransactionResult{}, errors.New("failed to assert ProofEncoder")
	}

	bindProof, err := pe.ToProof(proof)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to encode proof: %w", err)
	}

	// Encode signatures
	se, ok := e.Encoder.(SignaturesEncoder[mcms.Signature])
	if !ok {
		return types.TransactionResult{}, errors.New("failed to assert SignatureEncoder")
	}

	bindSignatures, err := se.ToSignatures(sortedSignatures, root)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to encode signatures: %w", err)
	}

	qID, err := tvm.RandomQueryID()
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to generate random query ID: %w", err)
	}

	body, err := tlb.ToCell(mcms.SetRoot{
		QueryID: qID,

		Root:       tlbe.NewUint256(new(big.Int).SetBytes(root[:])),
		ValidUntil: validUntil,
		Metadata:   rm,

		MetadataProof: bindProof,
		Signatures:    bindSignatures,
	})
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to encode ExecuteBatch body: %w", err)
	}

	skipSend := false // TODO: expose via executor options

	return SendTx(ctx, TxOpts{
		Wallet:   e.wallet,
		DstAddr:  dstAddr,
		Amount:   e.amount,
		Body:     body,
		SkipSend: skipSend,
	})
}
