package ton

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/common"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
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
func NewExecutor(encoder sdk.Encoder, client ton.APIClientWrapped, wallet *wallet.Wallet, amount tlb.Coins) (sdk.Executor, error) {
	if IsNil(encoder) {
		return nil, errors.New("failed to create sdk.Executor - encoder (sdk.Encoder) is nil")
	}

	if IsNil(client) {
		return nil, errors.New("failed to create sdk.Executor - client (ton.APIClientWrapped) is nil")
	}

	if wallet == nil {
		return nil, errors.New("failed to create sdk.Executor - wallet (*wallet.Wallet) is nil")
	}

	return &executor{
		Encoder:   encoder,
		Inspector: NewInspector(client, NewConfigTransformer()),
		wallet:    wallet,
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

	qID, err := RandomQueryID()
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

	msg := &wallet.Message{
		Mode: wallet.PayGasSeparately | wallet.IgnoreErrors,
		InternalMessage: &tlb.InternalMessage{
			IHRDisabled: true,
			Bounce:      true,
			DstAddr:     dstAddr,
			Amount:      e.amount,
			Body:        body,
		},
	}

	tx, _, err := e.wallet.SendWaitTransaction(ctx, msg)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to execute op: %w", err)
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

	qID, err := RandomQueryID()
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to generate random query ID: %w", err)
	}

	body, err := tlb.ToCell(mcms.SetRoot{
		QueryID: qID,

		Root:       new(big.Int).SetBytes(root[:]),
		ValidUntil: validUntil,
		Metadata:   rm,

		MetadataProof: bindProof,
		Signatures:    bindSignatures,
	})
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to encode ExecuteBatch body: %w", err)
	}

	msg := &wallet.Message{
		Mode: wallet.PayGasSeparately | wallet.IgnoreErrors,
		InternalMessage: &tlb.InternalMessage{
			IHRDisabled: true,
			Bounce:      true,
			DstAddr:     dstAddr,
			Amount:      e.amount,
			Body:        body,
		},
	}

	tx, _, err := e.wallet.SendWaitTransaction(ctx, msg)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to set root: %w", err)
	}

	return types.TransactionResult{
		Hash:        hex.EncodeToString(tx.Hash),
		ChainFamily: chain_selectors.FamilyTon,
		RawData:     tx,
	}, nil
}

// IsNil checks if a value is nil or if it's a reference type with a nil underlying value.
// Notice: vendoring github:samber/lo
func IsNil(x any) bool {
	if x == nil {
		return true
	}
	v := reflect.ValueOf(x)
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
