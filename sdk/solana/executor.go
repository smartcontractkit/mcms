package solana

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/smartcontractkit/chainlink-ccip/chains/solana/contracts/tests/config"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/mcms"

	"github.com/smartcontractkit/mcms/types"
)

const (
	// FIXME: should we reuse these from sdk/evm/utils or duplicate them here?
	SignatureVOffset    = 27
	SignatureVThreshold = 2
)

// Executor is an Executor implementation for EVM chains, allowing for the execution of operations on the MCMS contract
type Executor struct {
	*Encoder
	*Inspector
	client *rpc.Client
	auth   solana.PrivateKey
}

// NewExecutor creates a new Executor for EVM chains
func NewExecutor(client *rpc.Client, auth solana.PrivateKey, encoder *Encoder) *Executor {
	return &Executor{
		Encoder:   encoder,
		Inspector: NewInspector(client),
		client:    client,
		auth:      auth,
	}
}

func (e *Executor) ExecuteOperation(
	metadata types.ChainMetadata,
	nonce uint32,
	proofs []common.Hash,
	op types.Operation,
) (string, error) {
	programID, msigName, err := ParseContractAddress(metadata.MCMAddress)
	if err != nil {
		return "", err
	}
	chainID := uint64(e.ChainSelector)
	byteProofs := [][32]byte{}
	for _, p := range proofs {
		byteProofs = append(byteProofs, p)
	}

	mcm.SetProgramID(programID) // see https://github.com/gagliardetto/solana-go/issues/254
	configPDA := mcms.McmConfigAddress(msigName)
	rootMetadataPDA := mcms.RootMetadataAddress(msigName)
	expiringRootAndOpCountPDA := mcms.ExpiringRootAndOpCountAddress(msigName)

	// Unmarshal the AdditionalFields from the operation
	var additionalFields AdditionalFields
	if err := json.Unmarshal(op.Transaction.AdditionalFields, &additionalFields); err != nil {
		return "", err
	}
	to, err := solana.PublicKeyFromBase58(op.Transaction.To)
	if err != nil {
		return "", err
	}

	ix := mcm.NewExecuteInstruction(
		msigName,
		chainID,
		uint64(nonce),
		op.Transaction.Data,
		byteProofs,

		configPDA,
		rootMetadataPDA,
		expiringRootAndOpCountPDA,
		to,
		mcms.McmSignerAddress(msigName),
		e.auth.PublicKey(),
	)
	// TODO: this should come as a param
	ctx := context.Background()
	signature, err := sendAndConfirm(ctx, e.client, e.auth, ix, rpc.CommitmentConfirmed)
	if err != nil {
		return "", fmt.Errorf("unable to call execute operation instruction: %w", err)
	}
	return signature, nil
}

func (e *Executor) SetRoot(
	metadata types.ChainMetadata,
	proof []common.Hash,
	root [32]byte,
	validUntil uint32,
	sortedSignatures []types.Signature,
) (string, error) {
	programID, msigName, err := ParseContractAddress(metadata.MCMAddress)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithCancel(context.Background()) // FIXME: add context as a method parameter?
	defer cancel()

	if len(sortedSignatures) > math.MaxUint8 {
		return "", fmt.Errorf("too many signatures (max %d)", math.MaxUint8)
	}

	// FIXME: global variables are bad, mmkay?
	config.TestChainID = uint64(e.ChainSelector)

	mcm.SetProgramID(programID) // see https://github.com/gagliardetto/solana-go/issues/254

	mcmAddress := solana.MustPublicKeyFromBase58(metadata.MCMAddress)
	configPDA := mcms.McmConfigAddress(msigName)
	rootMetadataPDA := mcms.RootMetadataAddress(msigName)
	expiringRootAndOpCountPDA := mcms.ExpiringRootAndOpCountAddress(msigName)
	signaturesPDA := mcms.RootSignaturesAddress(msigName, root, validUntil)
	seenSignedHashesPDA := mcms.SeenSignedHashesAddress(msigName, root, validUntil)

	err = initializeMcmProgram(ctx, e.client, e.auth, uint64(e.ChainSelector), mcmAddress, msigName,
		configPDA, rootMetadataPDA, expiringRootAndOpCountPDA)
	if err != nil {
		return "", fmt.Errorf("unable to initialize mcm program: %w", err)
	}

	err = e.preloadSignatures(ctx, msigName, root, validUntil, sortedSignatures, signaturesPDA)
	if err != nil {
		return "", fmt.Errorf("unable to preload signatures: %w", err)
	}

	setRootInstruction := mcm.NewSetRootInstruction(msigName, root, validUntil,
		e.solanaMetadata(metadata, configPDA), solanaProof(proof),
		signaturesPDA, rootMetadataPDA, seenSignedHashesPDA, expiringRootAndOpCountPDA, configPDA,
		e.auth.PublicKey(), solana.SystemProgramID)
	signature, err := sendAndConfirm(ctx, e.client, e.auth, setRootInstruction, rpc.CommitmentConfirmed)
	if err != nil {
		return "", fmt.Errorf("unable to set root: %w", err)
	}

	return signature, nil
}

func (e *Executor) preloadSignatures(
	ctx context.Context,
	mcmName [32]byte,
	root [32]byte,
	validUntil uint32,
	sortedSignatures []types.Signature,
	signaturesPDA solana.PublicKey,
) error {
	initSignaturesInstruction := mcm.NewInitSignaturesInstruction(mcmName, root, validUntil,
		uint8(len(sortedSignatures)), signaturesPDA, e.auth.PublicKey(), solana.SystemProgramID)
	_, err := sendAndConfirm(ctx, e.client, e.auth, initSignaturesInstruction, rpc.CommitmentConfirmed)
	if err != nil {
		return fmt.Errorf("unable to initialize signatures: %w", err)
	}

	solanaSignatures := solanaSignatures(sortedSignatures)

	for i, chunkIndex := range chunkIndexes(len(solanaSignatures), config.MaxAppendSignatureBatchSize) {
		appendSignaturesInstruction := mcm.NewAppendSignaturesInstruction(mcmName, root, validUntil,
			solanaSignatures[chunkIndex[0]:chunkIndex[1]], signaturesPDA, e.auth.PublicKey())
		_, err := sendAndConfirm(ctx, e.client, e.auth, appendSignaturesInstruction, rpc.CommitmentConfirmed)
		if err != nil {
			return fmt.Errorf("unable to append signatures (%d): %w", i, err)
		}
	}

	finalizeSignaturesInstruction := mcm.NewFinalizeSignaturesInstruction(mcmName, root, validUntil, signaturesPDA,
		e.auth.PublicKey())
	_, err = sendAndConfirm(ctx, e.client, e.auth, finalizeSignaturesInstruction, rpc.CommitmentConfirmed)
	if err != nil {
		return fmt.Errorf("unable to finalize signatures: %w", err)
	}

	return nil
}

func (e *Executor) solanaMetadata(metadata types.ChainMetadata, configPDA [32]byte) mcm.RootMetadataInput {
	return mcm.RootMetadataInput{
		ChainId:              uint64(e.ChainSelector),
		Multisig:             solana.PublicKey(configPDA),
		PreOpCount:           metadata.StartingOpCount,
		PostOpCount:          metadata.StartingOpCount + e.TxCount,
		OverridePreviousRoot: e.OverridePreviousRoot,
	}
}

func solanaProof(proof []common.Hash) [][32]uint8 {
	sproof := make([][32]uint8, len(proof))
	for i := range proof {
		sproof[i] = proof[i]
	}
	return sproof
}

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
