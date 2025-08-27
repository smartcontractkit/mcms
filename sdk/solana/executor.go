package solana

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/contracts/tests/config"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/mcm"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Executor = (*Executor)(nil)

var ErrSignedHashAlreadySeen = func(root [32]byte) error { return fmt.Errorf("SignedHashAlreadySeen: 0x%x", root) }

const maxPreloadSignaturesAttempts = 3

// Executor is an Executor implementation for Solana chains, allowing for the execution of
// operations on the MCMS contract
type Executor struct {
	*Encoder
	*Inspector
	client         *rpc.Client
	auth           solana.PrivateKey
	sendAndConfirm SendAndConfirmFn
}

// NewExecutor creates a new Executor for Solana chains
func NewExecutor(encoder *Encoder, client *rpc.Client, auth solana.PrivateKey) *Executor {
	return &Executor{
		Encoder:        encoder,
		Inspector:      NewInspector(client),
		client:         client,
		auth:           auth,
		sendAndConfirm: sendAndConfirm,
	}
}

func (e Executor) withSendAndConfirmFn(fn SendAndConfirmFn) *Executor {
	e.sendAndConfirm = fn
	return &e
}

// ExecuteOperation executes an operation on the MCMS program on the Solana chain
func (e *Executor) ExecuteOperation(
	ctx context.Context,
	metadata types.ChainMetadata,
	nonce uint32,
	proof []common.Hash,
	op types.Operation,
) (types.TransactionResult, error) {
	programID, msigID, err := ParseContractAddress(metadata.MCMAddress)
	if err != nil {
		return types.TransactionResult{}, err
	}
	selector := uint64(e.ChainSelector)
	byteProof := make([][32]byte, 0, len(proof))
	for _, p := range proof {
		byteProof = append(byteProof, p)
	}

	mcm.SetProgramID(programID) // see https://github.com/gagliardetto/solana-go/issues/254
	configPDA, err := FindConfigPDA(programID, msigID)
	if err != nil {
		return types.TransactionResult{}, err
	}
	rootMetadataPDA, err := FindRootMetadataPDA(programID, msigID)
	if err != nil {
		return types.TransactionResult{}, err
	}
	expiringRootAndOpCountPDA, err := FindExpiringRootAndOpCountPDA(programID, msigID)
	if err != nil {
		return types.TransactionResult{}, err
	}
	signedPDA, err := FindSignerPDA(programID, msigID)
	if err != nil {
		return types.TransactionResult{}, err
	}

	// Unmarshal the AdditionalFields from the operation
	var additionalFields AdditionalFields
	if err = json.Unmarshal(op.Transaction.AdditionalFields, &additionalFields); err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to unmarshal additional fields: %w", err)
	}
	toProgramID, err := ParseProgramID(op.Transaction.To)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to prase program id from To field: %w", err)
	}

	ix := mcm.NewExecuteInstruction(
		msigID,
		selector,
		uint64(nonce),
		op.Transaction.Data,
		byteProof,
		configPDA,
		rootMetadataPDA,
		expiringRootAndOpCountPDA,
		toProgramID,
		signedPDA,
		e.auth.PublicKey(),
	)
	ix.AccountMetaSlice = append(ix.AccountMetaSlice, additionalFields.Accounts...)

	signature, tx, err := e.sendAndConfirm(ctx, e.client, e.auth, ix, rpc.CommitmentConfirmed)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to call execute operation instruction: %w", err)
	}

	return types.TransactionResult{
		Hash:        signature,
		ChainFamily: chain_selectors.FamilySolana,
		RawData:     tx,
	}, nil
}

// SetRoot sets the merkle root in the MCM contract on the Solana chain
func (e *Executor) SetRoot(
	ctx context.Context,
	metadata types.ChainMetadata,
	proof []common.Hash,
	root [32]byte,
	validUntil uint32,
	sortedSignatures []types.Signature,
) (types.TransactionResult, error) {
	sameRoot, err := e.equalCurrentRoot(ctx, metadata.MCMAddress, root)
	if err != nil {
		return types.TransactionResult{}, err
	}
	if sameRoot {
		return types.TransactionResult{}, ErrSignedHashAlreadySeen(root)
	}

	programID, pdaSeed, err := ParseContractAddress(metadata.MCMAddress)
	if err != nil {
		return types.TransactionResult{}, err
	}

	if len(sortedSignatures) > math.MaxUint8 {
		return types.TransactionResult{}, fmt.Errorf("too many signatures (max %d)", math.MaxUint8)
	}

	// FIXME: global variables are bad, mmkay?
	// see https://github.com/gagliardetto/solana-go/issues/254
	mcm.SetProgramID(programID)

	configPDA, err := FindConfigPDA(programID, pdaSeed)
	if err != nil {
		return types.TransactionResult{}, err
	}
	rootMetadataPDA, err := FindRootMetadataPDA(programID, pdaSeed)
	if err != nil {
		return types.TransactionResult{}, err
	}
	expiringRootAndOpCountPDA, err := FindExpiringRootAndOpCountPDA(programID, pdaSeed)
	if err != nil {
		return types.TransactionResult{}, err
	}
	rootSignaturesPDA, err := FindRootSignaturesPDA(programID, pdaSeed, root, validUntil, e.auth.PublicKey())
	if err != nil {
		return types.TransactionResult{}, err
	}
	seenSignedHashesPDA, err := FindSeenSignedHashesPDA(programID, pdaSeed, root, validUntil)
	if err != nil {
		return types.TransactionResult{}, err
	}

	err = e.preloadSignatures(ctx, pdaSeed, root, validUntil, sortedSignatures, rootSignaturesPDA, 0)
	if err != nil {
		return types.TransactionResult{}, err
	}

	setRootInstruction := mcm.NewSetRootInstruction(
		pdaSeed,
		root,
		validUntil,
		e.solanaMetadata(metadata, configPDA),
		solanaProof(proof),
		rootSignaturesPDA,
		rootMetadataPDA,
		seenSignedHashesPDA,
		expiringRootAndOpCountPDA,
		configPDA,
		e.auth.PublicKey(),
		solana.SystemProgramID)
	signature, tx, err := e.sendAndConfirm(ctx, e.client, e.auth, setRootInstruction, rpc.CommitmentConfirmed)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to set root: %w", err)
	}

	return types.TransactionResult{
		Hash:        signature,
		ChainFamily: chain_selectors.FamilySolana,
		RawData:     tx,
	}, nil
}

// preloadSignatures preloads the signatures into the MCM program by looping can calling the
// append signatures instruction and concluding with the finalize signatures instruction.
func (e *Executor) preloadSignatures(
	ctx context.Context,
	mcmName [32]byte,
	root [32]byte,
	validUntil uint32,
	sortedSignatures []types.Signature,
	signaturesPDA solana.PublicKey,
	attempt int,
) error {
	initSignaturesInstruction := mcm.NewInitSignaturesInstruction(mcmName, root, validUntil,
		uint8(len(sortedSignatures)), signaturesPDA, e.auth.PublicKey(), solana.SystemProgramID) //nolint:gosec
	_, _, err := e.sendAndConfirm(ctx, e.client, e.auth, initSignaturesInstruction, rpc.CommitmentConfirmed)
	if err != nil {
		if isAccountAlreadyInUseError(err) {
			return e.retryPreloadSignatures(ctx, mcmName, root, validUntil, sortedSignatures, signaturesPDA, attempt)
		}

		return fmt.Errorf("unable to initialize signatures: %w", err)
	}

	solanaSignatures := solanaSignatures(sortedSignatures)

	for i, chunkIndex := range chunkIndexes(len(solanaSignatures), config.MaxAppendSignatureBatchSize) {
		appendSignaturesInstruction := mcm.NewAppendSignaturesInstruction(mcmName, root, validUntil,
			solanaSignatures[chunkIndex[0]:chunkIndex[1]], signaturesPDA, e.auth.PublicKey())
		_, _, serr := e.sendAndConfirm(ctx, e.client, e.auth, appendSignaturesInstruction, rpc.CommitmentConfirmed)
		if serr != nil {
			return fmt.Errorf("unable to append signatures (%d): %w", i, serr)
		}
	}

	finalizeSignaturesInstruction := mcm.NewFinalizeSignaturesInstruction(mcmName, root, validUntil, signaturesPDA,
		e.auth.PublicKey())
	_, _, err = e.sendAndConfirm(ctx, e.client, e.auth, finalizeSignaturesInstruction, rpc.CommitmentConfirmed)
	if err != nil {
		return fmt.Errorf("unable to finalize signatures: %w", err)
	}

	return nil
}

// retryPreloadSignatures clears the signatures pda and then calls preloadSignatures
func (e *Executor) retryPreloadSignatures(
	ctx context.Context,
	mcmName [32]byte,
	root [32]byte,
	validUntil uint32,
	sortedSignatures []types.Signature,
	signaturesPDA solana.PublicKey,
	attempt int,
) error {
	if attempt >= maxPreloadSignaturesAttempts {
		return fmt.Errorf("maximum attempts to retry preload signatures reached (%d); aborting", attempt)
	}

	clearSignaturesInstruction := mcm.NewClearSignaturesInstruction(mcmName, root, validUntil, signaturesPDA,
		e.auth.PublicKey())
	_, _, err := e.sendAndConfirm(ctx, e.client, e.auth, clearSignaturesInstruction, rpc.CommitmentConfirmed)
	if err != nil {
		return fmt.Errorf("unable to clear signatures: %w", err)
	}

	return e.preloadSignatures(ctx, mcmName, root, validUntil, sortedSignatures, signaturesPDA, attempt+1)
}

// solanaMetadata returns the root metadata input for the MCM program
func (e *Executor) solanaMetadata(metadata types.ChainMetadata, configPDA [32]byte) mcm.RootMetadataInput {
	return mcm.RootMetadataInput{
		ChainId:              uint64(e.ChainSelector),
		Multisig:             solana.PublicKey(configPDA),
		PreOpCount:           metadata.StartingOpCount,
		PostOpCount:          metadata.StartingOpCount + e.TxCount,
		OverridePreviousRoot: e.OverridePreviousRoot,
	}
}

func (e *Executor) equalCurrentRoot(ctx context.Context, mcmAddress string, newRoot [32]byte) (bool, error) {
	currentRoot, _, err := e.GetRoot(ctx, mcmAddress)
	if err != nil {
		return false, fmt.Errorf("failed to get root: %w", err)
	}

	return currentRoot == newRoot, nil
}

// solanaProof converts a proof coming as a slice of common.Hash to a slice of [32]byte.
func solanaProof(proof []common.Hash) [][32]uint8 {
	sproof := make([][32]uint8, len(proof))
	for i := range proof {
		sproof[i] = proof[i]
	}

	return sproof
}

// solanaSignatures converts a slice of types.Signature to a slice of mcm.Signature
func solanaSignatures(signatures []types.Signature) []mcm.Signature {
	solanaSignatures := make([]mcm.Signature, len(signatures))
	for i, signature := range signatures {
		v := signature.V
		if v < SignatureVThreshold {
			v += SignatureVOffset
		}

		solanaSignatures[i] = mcm.Signature{R: signature.R, S: signature.S, V: v}
	}

	return solanaSignatures
}

var accountAlreadyInUsePattern = regexp.MustCompile(`Allocate: account Address.*already in use`)

func isAccountAlreadyInUseError(err error) bool {
	return accountAlreadyInUsePattern.MatchString(err.Error())
}
