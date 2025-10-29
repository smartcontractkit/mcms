package ton

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"

	"github.com/ethereum/go-ethereum/common"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	commonton "github.com/smartcontractkit/chainlink-ton/pkg/ccip/bindings/common"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
	"github.com/smartcontractkit/chainlink-ton/pkg/ccip/bindings/ocr"
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
func NewExecutor(encoder sdk.Encoder, client *ton.APIClient, wallet *wallet.Wallet, amount tlb.Coins) sdk.Executor {
	return &executor{
		Encoder:   encoder,
		Inspector: NewInspector(client, NewConfigTransformer()),
		wallet:    wallet,
		amount:    amount,
	}
}

func (e *executor) ExecuteOperation(
	ctx context.Context,
	metadata types.ChainMetadata,
	nonce uint32,
	proof []common.Hash,
	op types.Operation,
) (types.TransactionResult, error) {
	if e.Encoder == nil {
		return types.TransactionResult{}, errors.New("executor was created without an encoder")
	}

	oe, ok := e.Encoder.(OperationEncoder[mcms.Op])
	if !ok {
		return types.TransactionResult{}, fmt.Errorf("failed to assert OperationEncoder")
	}

	bindOp, err := oe.ToOperation(nonce, metadata, op)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to convert to operation: %w", err)
	}

	// Encode proofs
	pe, ok := e.Encoder.(ProofEncoder[mcms.Proof])
	if !ok {
		return types.TransactionResult{}, fmt.Errorf("failed to assert ProofEncoder")
	}

	bindProof, err := pe.ToProof(proof)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to encode proof: %w", err)
	}

	body, err := tlb.ToCell(mcms.Execute{
		QueryID: rand.Uint64(),

		Op:    bindOp,
		Proof: commonton.SnakeData[mcms.Proof](bindProof),
	})
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to encode ExecuteBatch body: %w", err)
	}

	// Map to Ton Address type
	dstAddr, err := address.ParseAddr(metadata.MCMAddress)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("invalid timelock address: %w", err)
	}

	msg := &wallet.Message{
		Mode: wallet.PayGasSeparately,
		InternalMessage: &tlb.InternalMessage{
			IHRDisabled: true,
			Bounce:      true,
			DstAddr:     dstAddr,
			Amount:      e.amount,
			Body:        body,
		},
	}

	// TODO: do we wait for execution trace?
	tx, _, err := e.wallet.SendWaitTransaction(ctx, msg)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to set config: %w", err)
	}

	return types.TransactionResult{
		Hash:        hex.EncodeToString(tx.Hash),
		ChainFamily: chain_selectors.FamilyTon,
		RawData:     tx,
	}, err
}

func (e *executor) SetRoot(
	ctx context.Context,
	metadata types.ChainMetadata,
	proof []common.Hash,
	root [32]byte,
	validUntil uint32,
	sortedSignatures []types.Signature,
) (types.TransactionResult, error) {
	if e.Encoder == nil {
		return types.TransactionResult{}, errors.New("Executor was created without an encoder")
	}

	// Map to Ton Address type
	dstAddr, err := address.ParseAddr(metadata.MCMAddress)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("invalid timelock address: %w", err)
	}

	rme, ok := e.Encoder.(RootMetadataEncoder[mcms.RootMetadata])
	if !ok {
		return types.TransactionResult{}, fmt.Errorf("failed to assert RootMetadataEncoder")
	}

	rm, err := rme.ToRootMetadata(metadata)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to convert to root metadata: %w", err)
	}

	// Encode proofs
	pe, ok := e.Encoder.(ProofEncoder[mcms.Proof])
	if !ok {
		return types.TransactionResult{}, fmt.Errorf("failed to assert ProofEncoder")
	}

	bindProof, err := pe.ToProof(proof)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to encode proof: %w", err)
	}

	// Encode signatures
	se, ok := e.Encoder.(SignaturesEncoder[ocr.SignatureEd25519])
	if !ok {
		return types.TransactionResult{}, fmt.Errorf("failed to assert SignatureEncoder")
	}

	bindSignatures, err := se.ToSignatures(sortedSignatures, root)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to encode signatures: %w", err)
	}

	body, err := tlb.ToCell(mcms.SetRoot{
		QueryID: rand.Uint64(),

		Root:       new(big.Int).SetBytes(root[:]),
		ValidUntil: validUntil,
		Metadata:   rm,

		MetadataProof: commonton.SnakeData[mcms.Proof](bindProof),
		Signatures:    commonton.SnakeData[ocr.SignatureEd25519](bindSignatures),
	})
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to encode ExecuteBatch body: %w", err)
	}

	msg := &wallet.Message{
		Mode: wallet.PayGasSeparately,
		InternalMessage: &tlb.InternalMessage{
			IHRDisabled: true,
			Bounce:      true,
			DstAddr:     dstAddr,
			Amount:      e.amount,
			Body:        body,
		},
	}

	// TODO: do we wait for execution trace?
	tx, _, err := e.wallet.SendWaitTransaction(ctx, msg)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to set config: %w", err)
	}

	return types.TransactionResult{
		Hash:        hex.EncodeToString(tx.Hash),
		ChainFamily: chain_selectors.FamilyTon,
		RawData:     tx,
	}, nil
}
