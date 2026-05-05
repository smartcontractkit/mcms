package stellar

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/common"
	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-stellar/bindings"
	stellarmcms "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/mcms"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Executor = (*Executor)(nil)

// Executor submits MCMS execute/set_root calls via [bindings.Invoker] (e.g. chainlink-stellar Deployer).
type Executor struct {
	*Encoder
	*Inspector
	invoker bindings.Invoker
}

// NewExecutor builds an Executor sharing invoker with read and write paths.
func NewExecutor(encoder *Encoder, invoker bindings.Invoker) *Executor {
	return &Executor{
		Encoder:   encoder,
		Inspector: NewInspector(invoker),
		invoker:   invoker,
	}
}

// ExecuteOperation invokes Soroban `execute` with a [stellarmcms.StellarOp] and Merkle proof.
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

	if uint64(nonce) >= uint40MaxExclusive {
		return types.TransactionResult{}, fmt.Errorf("%w: nonce %d", ErrUint40Overflow, nonce)
	}

	chainID, err := ChainNetworkID(e.ChainSelector)
	if err != nil {
		return types.TransactionResult{}, err
	}

	multisig, err := ParseContractID(metadata.MCMAddress)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("mcmAddress: %w", err)
	}

	to, err := ParseContractID(op.Transaction.To)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("transaction.to: %w", err)
	}

	valueWord, err := parseValueWord(op.Transaction.AdditionalFields)
	if err != nil {
		return types.TransactionResult{}, err
	}

	stellarOp := stellarmcms.StellarOp{
		ChainId:  [32]byte(chainID),
		Data:     op.Transaction.Data,
		Multisig: multisig,
		Nonce:    uint64(nonce),
		To:       to,
		Value:    valueWord,
	}

	mcmsClient, err := newMCMSClient(e.invoker, metadata.MCMAddress)
	if err != nil {
		return types.TransactionResult{}, err
	}

	mp := merkleProofFromHashes(proof)

	if err := mcmsClient.Execute(ctx, stellarOp, mp); err != nil {
		return types.TransactionResult{ChainFamily: chainsel.FamilyStellar}, err
	}

	return stellarTransactionResult(e.invoker), nil
}

// SetRoot invokes Soroban `set_root` with metadata and ECDSA signatures (contract ABI layout).
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

	if len(sortedSignatures) > math.MaxUint8 {
		return types.TransactionResult{}, fmt.Errorf("too many signatures (max %d)", math.MaxUint8)
	}

	rootMeta, err := e.stellarRootMetadata(metadata)
	if err != nil {
		return types.TransactionResult{}, err
	}

	mcmsClient, err := newMCMSClient(e.invoker, metadata.MCMAddress)
	if err != nil {
		return types.TransactionResult{}, err
	}

	sigVec := signatureVecFrom(sortedSignatures)
	mp := merkleProofFromHashes(proof)

	if err := mcmsClient.SetRoot(ctx, root, validUntil, rootMeta, mp, sigVec); err != nil {
		return types.TransactionResult{ChainFamily: chainsel.FamilyStellar}, err
	}

	return stellarTransactionResult(e.invoker), nil
}

func (e *Executor) stellarRootMetadata(metadata types.ChainMetadata) (stellarmcms.StellarRootMetadata, error) {
	var zero stellarmcms.StellarRootMetadata

	if metadata.StartingOpCount >= uint40MaxExclusive {
		return zero, fmt.Errorf("%w: startingOpCount %d", ErrUint40Overflow, metadata.StartingOpCount)
	}

	post := metadata.StartingOpCount + e.TxCount
	if post >= uint40MaxExclusive {
		return zero, fmt.Errorf("%w: postOpCount (starting+txCount) %d", ErrUint40Overflow, post)
	}

	chainID, err := ChainNetworkID(e.ChainSelector)
	if err != nil {
		return zero, err
	}

	multisig, err := ParseContractID(metadata.MCMAddress)
	if err != nil {
		return zero, fmt.Errorf("mcmAddress: %w", err)
	}

	return stellarmcms.StellarRootMetadata{
		ChainId:              [32]byte(chainID),
		Multisig:             multisig,
		OverridePreviousRoot: e.OverridePreviousRoot,
		PreOpCount:           metadata.StartingOpCount,
		PostOpCount:          post,
	}, nil
}

func merkleProofFromHashes(proof []common.Hash) stellarmcms.MerkleProof {
	inner := make([][32]byte, len(proof))
	for i, p := range proof {
		inner[i] = p
	}

	return stellarmcms.MerkleProof{Inner: inner}
}

func signatureVecFrom(sorted []types.Signature) stellarmcms.SignatureVec {
	inner := make([]stellarmcms.Signature, len(sorted))
	for i, s := range sorted {
		inner[i] = stellarmcms.Signature{
			R: s.R,
			S: s.S,
			V: uint32(s.V),
		}
	}

	return stellarmcms.SignatureVec{Inner: inner}
}
