package solana

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/contracts/tests/config"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Executor = (*Executor)(nil)

// Executor is an Executor implementation for Solana chains, allowing for the execution of
// operations on the MCMS contract
type Executor struct {
	*Encoder
	*Inspector
	client *rpc.Client
	auth   solana.PrivateKey
}

// NewExecutor creates a new Executor for Solana chains
func NewExecutor(client *rpc.Client, auth solana.PrivateKey, encoder *Encoder) *Executor {
	return &Executor{
		Encoder:   encoder,
		Inspector: NewInspector(client),
		client:    client,
		auth:      auth,
	}
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
	toProgramID, _, err := ParseContractAddress(op.Transaction.To)
	if errors.Is(err, ErrInvalidContractAddressFormat) {
		var pkerr error
		toProgramID, pkerr = solana.PublicKeyFromBase58(op.Transaction.To)
		if pkerr != nil {
			return types.TransactionResult{}, fmt.Errorf("unable to get hash from base58 To address: %w", err)
		}
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
	// Append the accounts from the AdditionalFields
	ix.AccountMetaSlice = append(ix.AccountMetaSlice, additionalFields.Accounts...)
	signature, tx, err := sendAndConfirm(ctx, e.client, e.auth, ix, rpc.CommitmentConfirmed)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to call execute operation instruction: %w", err)
	}

	return types.TransactionResult{
		Hash:           signature,
		ChainFamily:    chain_selectors.FamilySolana,
		RawTransaction: tx,
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
	rootSignaturesPDA, err := FindRootSignaturesPDA(programID, pdaSeed, root, validUntil)
	if err != nil {
		return types.TransactionResult{}, err
	}
	seenSignedHashesPDA, err := FindSeenSignedHashesPDA(programID, pdaSeed, root, validUntil)
	if err != nil {
		return types.TransactionResult{}, err
	}

	err = e.preloadSignatures(ctx, pdaSeed, root, validUntil, sortedSignatures, rootSignaturesPDA)
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
	signature, tx, err := sendAndConfirm(ctx, e.client, e.auth, setRootInstruction, rpc.CommitmentConfirmed)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to set root: %w", err)
	}

	return types.TransactionResult{
		Hash:           signature,
		ChainFamily:    chain_selectors.FamilySolana,
		RawTransaction: tx,
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
) error {
	initSignaturesInstruction := mcm.NewInitSignaturesInstruction(mcmName, root, validUntil,
		uint8(len(sortedSignatures)), signaturesPDA, e.auth.PublicKey(), solana.SystemProgramID) //nolint:gosec
	_, _, err := sendAndConfirm(ctx, e.client, e.auth, initSignaturesInstruction, rpc.CommitmentConfirmed)
	if err != nil {
		return fmt.Errorf("unable to initialize signatures: %w", err)
	}

	solanaSignatures := solanaSignatures(sortedSignatures)

	for i, chunkIndex := range chunkIndexes(len(solanaSignatures), config.MaxAppendSignatureBatchSize) {
		appendSignaturesInstruction := mcm.NewAppendSignaturesInstruction(mcmName, root, validUntil,
			solanaSignatures[chunkIndex[0]:chunkIndex[1]], signaturesPDA, e.auth.PublicKey())
		_, _, serr := sendAndConfirm(ctx, e.client, e.auth, appendSignaturesInstruction, rpc.CommitmentConfirmed)
		if serr != nil {
			return fmt.Errorf("unable to append signatures (%d): %w", i, serr)
		}
	}

	finalizeSignaturesInstruction := mcm.NewFinalizeSignaturesInstruction(mcmName, root, validUntil, signaturesPDA,
		e.auth.PublicKey())
	_, _, err = sendAndConfirm(ctx, e.client, e.auth, finalizeSignaturesInstruction, rpc.CommitmentConfirmed)
	if err != nil {
		return fmt.Errorf("unable to finalize signatures: %w", err)
	}

	return nil
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
