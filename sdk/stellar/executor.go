package stellar

import (
	"context"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/common"
	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-stellar/bindings"
	mcmsbindings "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/mcms"
)

var _ sdk.Executor = (*Executor)(nil)

// Executor implements sdk.Executor for the Stellar (Soroban) MCMS contract.
// It wraps an Encoder, Inspector, and the Soroban invoker to execute
// operations and set Merkle roots on-chain.
type Executor struct {
	*Encoder
	*Inspector
	invoker bindings.Invoker
}

// NewExecutor creates a new Executor for Stellar chains.
func NewExecutor(encoder *Encoder, inspector *Inspector) (*Executor, error) {
	return &Executor{
		Encoder:   encoder,
		Inspector: inspector,
		invoker:   inspector.invoker,
	}, nil
}

// ExecuteOperation executes an MCMS operation (one transaction) on the Stellar chain.
func (e *Executor) ExecuteOperation(
	ctx context.Context,
	metadata types.ChainMetadata,
	nonce uint32,
	proof []common.Hash,
	op types.Operation,
) (types.TransactionResult, error) {
	chainNetworkID, err := chainNetworkID(e.ChainSelector)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("execute: chain network id: %w", err)
	}

	// Build the StellarOp from the SDK types.Operation.
	_, err = parseContractID(metadata.MCMAddress)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("execute: parse mcm address: %w", err)
	}
	toContractID, err := parseContractID(op.Transaction.To)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("execute: parse target address: %w", err)
	}
	payload, err := DecodeSorobanInvokePayload(op.Transaction.Data)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("execute: decode invoke payload: %w", err)
	}
	stellarOp := mcmsbindings.StellarOp{
		NetworkId:       chainNetworkID,
		Multisig:        metadata.MCMAddress,
		Nonce:           uint64(nonce),
		Target:          op.Transaction.To,
		Function:        payload.Function,
		ArgsXdr:         payload.ArgsXDR,
		EncodingVersion: encodingVersion,
	}
	_ = toContractID // validates the target is a contract address

	// Convert proof hashes.
	proofInner := make([][32]byte, len(proof))
	for i, p := range proof {
		proofInner[i] = p
	}
	merkleProof := mcmsbindings.MerkleProof{Inner: proofInner}

	client := mcmsbindings.NewMcmsClient(e.invoker, metadata.MCMAddress)
	if err := client.Execute(ctx, stellarOp, merkleProof); err != nil {
		return types.TransactionResult{}, fmt.Errorf("stellar mcms execute: %w", err)
	}

	return types.NewTransactionResult("", nil, chainsel.FamilyStellar), nil
}

// SetRoot sets a new Merkle root in the MCMS contract on the Stellar chain.
func (e *Executor) SetRoot(
	ctx context.Context,
	metadata types.ChainMetadata,
	proof []common.Hash,
	root [32]byte,
	validUntil uint32,
	sortedSignatures []types.Signature,
) (types.TransactionResult, error) {
	// Avoid re-submitting the same root.
	currentRoot, currentValidUntil, err := e.GetRoot(ctx, metadata.MCMAddress)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("set_root: get current root: %w", err)
	}
	if currentRoot == root && currentValidUntil == validUntil {
		return types.TransactionResult{}, fmt.Errorf("stellar set_root: root already set (0x%x)", root)
	}

	if len(sortedSignatures) > math.MaxUint8 {
		return types.TransactionResult{}, fmt.Errorf("too many signatures (%d > %d)", len(sortedSignatures), math.MaxUint8)
	}

	chainNetworkID, err := chainNetworkID(e.ChainSelector)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("set_root: chain network id: %w", err)
	}

	// Build StellarRootMetadata.
	_, err = parseContractID(metadata.MCMAddress)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("set_root: parse mcm address: %w", err)
	}
	stellarMeta := mcmsbindings.StellarRootMetadata{
		NetworkId:            chainNetworkID,
		Multisig:             metadata.MCMAddress,
		PreOpCount:           metadata.StartingOpCount,
		PostOpCount:          metadata.StartingOpCount + e.TxCount,
		OverridePreviousRoot: e.OverridePreviousRoot,
		ConfigVersion:        1,
		EncodingVersion:      encodingVersion,
	}

	// Convert proof.
	proofInner := make([][32]byte, len(proof))
	for i, p := range proof {
		proofInner[i] = p
	}
	metaProof := mcmsbindings.MerkleProof{Inner: proofInner}

	// Convert signatures from mcms/types.Signature to binding Signature.
	sigVec := mcmsbindings.SignatureVec{
		Inner: make([]mcmsbindings.Signature, len(sortedSignatures)),
	}
	for i, sig := range sortedSignatures {
		sigVec.Inner[i] = mcmsbindings.Signature{
			R: sig.R,
			S: sig.S,
			V: uint32(sig.V), // uint8 → uint32 is safe
		}
	}

	client := mcmsbindings.NewMcmsClient(e.invoker, metadata.MCMAddress)
	if err := client.SetRoot(ctx, root, validUntil, stellarMeta, metaProof, sigVec); err != nil {
		return types.TransactionResult{}, fmt.Errorf("stellar mcms set_root: %w", err)
	}

	return types.NewTransactionResult("", nil, chainsel.FamilyStellar), nil
}
