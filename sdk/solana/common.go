package solana

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	solanaCommon "github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/common"
)

const (
	// FIXME: should we reuse these from sdk/evm/utils or duplicate them here?
	SignatureVOffset    = 27
	SignatureVThreshold = 2
)

func FindSignerPDA(programID solana.PublicKey, msigID PDASeed) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("multisig_signer"), msigID[:]}
	return findPDA(programID, seeds)
}

func FindConfigPDA(programID solana.PublicKey, msigID PDASeed) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("multisig_config"), msigID[:]}
	return findPDA(programID, seeds)
}

func FindConfigSignersPDA(programID solana.PublicKey, msigID PDASeed) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("multisig_config_signers"), msigID[:]}
	return findPDA(programID, seeds)
}

func FindRootMetadataPDA(programID solana.PublicKey, msigID PDASeed) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("root_metadata"), msigID[:]}
	return findPDA(programID, seeds)
}

func FindExpiringRootAndOpCountPDA(programID solana.PublicKey, pdaSeed PDASeed) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("expiring_root_and_op_count"), pdaSeed[:]}
	return findPDA(programID, seeds)
}

func FindRootSignaturesPDA(
	programID solana.PublicKey, msigID PDASeed, root common.Hash, validUntil uint32,
) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("root_signatures"), msigID[:], root[:], validUntilBytes(validUntil)}
	return findPDA(programID, seeds)
}

func FindSeenSignedHashesPDA(
	programID solana.PublicKey, msigID PDASeed, root common.Hash, validUntil uint32,
) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("seen_signed_hashes"), msigID[:], root[:], validUntilBytes(validUntil)}
	return findPDA(programID, seeds)
}

func FindTimelockConfigPDA(
	programID solana.PublicKey, timelockID PDASeed) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("timelock_config"), timelockID[:]}
	return findPDA(programID, seeds)
}

func FindTimelockOperationPDA(
	programID solana.PublicKey, timelockID PDASeed, opID [32]byte) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("timelock_operation"), timelockID[:], opID[:]}
	return findPDA(programID, seeds)
}

func FindTimelockSignerPDA(
	programID solana.PublicKey, timelockID PDASeed) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("timelock_signer"), timelockID[:]}
	return findPDA(programID, seeds)
}

func findPDA(programID solana.PublicKey, seeds [][]byte) (solana.PublicKey, error) {
	pda, _, err := solana.FindProgramAddress(seeds, programID)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("unable to find %s pda: %w", string(seeds[0]), err)
	}

	return pda, nil
}

func validUntilBytes(validUntil uint32) []byte {
	const uint32Size = 4
	vuBytes := make([]byte, uint32Size)
	binary.LittleEndian.PutUint32(vuBytes, validUntil)

	return vuBytes
}

type instructionBuilder[I solana.Instruction] interface {
	ValidateAndBuild() (I, error)
}

// sendAndConfirmBuiltIx contains the common logic for sending and confirming instructions.
func sendAndConfirmBuiltIx(
	ctx context.Context,
	client *rpc.Client,
	auth solana.PrivateKey,
	builtInstruction solana.Instruction,
	commitmentType rpc.CommitmentType,
) (string, error) {
	result, err := solanaCommon.SendAndConfirm(ctx, client, []solana.Instruction{builtInstruction}, auth, commitmentType)
	if err != nil {
		return "", fmt.Errorf("unable to send instruction: %w", err)
	}
	if result.Transaction == nil {
		return "", fmt.Errorf("nil transaction in instruction result")
	}

	transaction, err := result.Transaction.GetTransaction()
	if err != nil {
		return "", fmt.Errorf("unable to get transaction from instruction result: %w", err)
	}

	return transaction.Signatures[0].String(), nil
}

// sendAndConfirm handles general instructions.
func sendAndConfirm[I solana.Instruction, B instructionBuilder[I]](
	ctx context.Context,
	client *rpc.Client,
	auth solana.PrivateKey,
	builder B,
	commitmentType rpc.CommitmentType,
) (string, error) {
	builtInstruction, err := builder.ValidateAndBuild()
	if err != nil {
		return "", fmt.Errorf("unable to validate and build instruction: %w", err)
	}

	// Pass the built instruction to the shared function
	return sendAndConfirmBuiltIx(ctx, client, auth, builtInstruction, commitmentType)
}

func chunkIndexes(numItems int, chunkSize int) [][2]int {
	indexes := make([][2]int, 0)

	for i := 0; i < numItems; i += chunkSize {
		end := i + chunkSize
		if end > numItems {
			end = numItems
		}
		indexes = append(indexes, [2]int{i, end})
	}

	return indexes
}
