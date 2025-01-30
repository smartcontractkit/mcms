package solana

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/timelock"
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
	programID solana.PublicKey, timelockID PDASeed,
) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("timelock_config"), timelockID[:]}
	return findPDA(programID, seeds)
}

func FindTimelockOperationPDA(
	programID solana.PublicKey, timelockID PDASeed, opID [32]byte,
) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("timelock_operation"), timelockID[:], opID[:]}
	return findPDA(programID, seeds)
}

func FindTimelockBypasserOperationPDA(
	programID solana.PublicKey, timelockID PDASeed, opID [32]byte) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("timelock_bypasser_operation"), timelockID[:], opID[:]}
	return findPDA(programID, seeds)
}

func FindTimelockSignerPDA(
	programID solana.PublicKey, timelockID PDASeed,
) (solana.PublicKey, error) {
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

type mcmInstructionBuilder interface {
	ValidateAndBuild() (*mcm.Instruction, error)
}

type timelockInstructionBuilder interface {
	ValidateAndBuild() (*timelock.Instruction, error)
}

func validateAndBuildSolanaInstruction(instructionBuilder any) (solana.Instruction, error) {
	var err error
	var builtInstruction solana.Instruction

	switch builder := instructionBuilder.(type) {
	case mcmInstructionBuilder:
		builtInstruction, err = builder.ValidateAndBuild()
		if err != nil {
			return nil, fmt.Errorf("unable to validate and build instruction: %w", err)
		}
	case timelockInstructionBuilder:
		builtInstruction, err = builder.ValidateAndBuild()
		if err != nil {
			return nil, fmt.Errorf("unable to validate and build instruction: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported instruction builder: %T", instructionBuilder)
	}

	return builtInstruction, nil
}

type SendAndConfirmFn func(
	ctx context.Context,
	client *rpc.Client,
	auth solana.PrivateKey,
	builder any,
	commitmentType rpc.CommitmentType,
) (string, *rpc.GetTransactionResult, error)

// sendAndConfirm contains the default logic for sending and confirming instructions.
func sendAndConfirm(
	ctx context.Context,
	client *rpc.Client,
	auth solana.PrivateKey,
	instructionBuilder any,
	commitmentType rpc.CommitmentType,
) (string, *rpc.GetTransactionResult, error) {
	instruction, err := validateAndBuildSolanaInstruction(instructionBuilder)
	if err != nil {
		return "", nil, fmt.Errorf("unable to validate and build instruction: %w", err)
	}

	result, err := solanaCommon.SendAndConfirm(ctx, client, []solana.Instruction{instruction}, auth, commitmentType)
	if err != nil {
		return "", nil, fmt.Errorf("unable to send instruction: %w", err)
	}
	if result.Transaction == nil {
		return "", nil, fmt.Errorf("nil transaction in instruction result")
	}

	transaction, err := result.Transaction.GetTransaction()
	if err != nil {
		return "", nil, fmt.Errorf("unable to get transaction from instruction result: %w", err)
	}

	return transaction.Signatures[0].String(), result, nil
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
