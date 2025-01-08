package solana

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	bindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	solanaCommon "github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/common"
)

const (
	// FIXME: should we reuse these from sdk/evm/utils or duplicate them here?
	SignatureVOffset    = 27
	SignatureVThreshold = 2
)

func FindSignerPDA(programID solana.PublicKey, pdaSeed PDASeed) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("multisig_signer"), pdaSeed[:]}
	return findPDA(programID, seeds)
}

func FindConfigPDA(programID solana.PublicKey, pdaSeed PDASeed) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("multisig_config"), pdaSeed[:]}
	return findPDA(programID, seeds)
}

func FindConfigSignersPDA(programID solana.PublicKey, pdaSeed PDASeed) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("multisig_config_signers"), pdaSeed[:]}
	return findPDA(programID, seeds)
}

func FindRootMetadataPDA(programID solana.PublicKey, pdaSeed PDASeed) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("root_metadata"), pdaSeed[:]}
	return findPDA(programID, seeds)
}

func FindExpiringRootAndOpCountPDA(programID solana.PublicKey, pdaSeed PDASeed) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("expiring_root_and_op_count"), pdaSeed[:]}
	return findPDA(programID, seeds)
}

func FindRootSignaturesPDA(
	programID solana.PublicKey, pdaSeed PDASeed, root common.Hash, validUntil uint32,
) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("root_signatures"), pdaSeed[:], root[:], validUntilBytes(validUntil)}
	return findPDA(programID, seeds)
}

func FindSeenSignedHashesPDA(
	programID solana.PublicKey, pdaSeed PDASeed, root common.Hash, validUntil uint32,
) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("seen_signed_hashes"), pdaSeed[:], root[:], validUntilBytes(validUntil)}
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

// FIXME: this should probably happen right after deployment; should we move to a
// "test" package and remove the call from SetConfig, SetRoot and Execute?
func initializeMcmProgram(
	ctx context.Context, client *rpc.Client, auth solana.PrivateKey,
	chainSelector uint64, mcmAddress solana.PublicKey, mcmName [32]byte,
	configPDA, rootMetadataPDA, expiringRootAndOpCountPDA solana.PublicKey,
) error {
	var configAccount bindings.MultisigConfig
	err := solanaCommon.GetAccountDataBorshInto(ctx, client, configPDA, rpc.CommitmentConfirmed, &configAccount)
	if err == nil {
		// fmt.Printf("MCM ALREADY INITIALIZED\n")
		return nil
	}

	data, err := client.GetAccountInfoWithOpts(ctx, mcmAddress, &rpc.GetAccountInfoOpts{
		Commitment: rpc.CommitmentConfirmed,
	})
	if err != nil {
		return fmt.Errorf("unable to get account info: %w", err)
	}

	var programData struct {
		DataType uint32
		Address  solana.PublicKey
	}
	err = bin.UnmarshalBorsh(&programData, data.Bytes())
	if err != nil {
		return fmt.Errorf("unable to unmarshal borsh: %w", err)
	}

	instruction := bindings.NewInitializeInstruction(
		chainSelector,
		mcmName,
		configPDA,
		auth.PublicKey(),
		solana.SystemProgramID,
		mcmAddress,
		programData.Address,
		rootMetadataPDA,
		expiringRootAndOpCountPDA,
	)

	_, err = sendAndConfirm(ctx, client, auth, instruction, rpc.CommitmentConfirmed)
	if err != nil {
		return fmt.Errorf("unable to initialize mcm program: %w", err)
	}

	// check that the config info was indeed saved
	err = solanaCommon.GetAccountDataBorshInto(ctx, client, configPDA, rpc.CommitmentConfirmed, &configAccount)
	if err != nil {
		return fmt.Errorf("unable to get account data borsh: %w", err)
	}
	if chainSelector != configAccount.ChainId {
		return fmt.Errorf("chain selector does not match: %v vs %v", chainSelector, configAccount.ChainId)
	}
	if auth.PublicKey() != configAccount.Owner {
		return fmt.Errorf("owner does not match: %v vs %v", auth.PublicKey(), configAccount.Owner)
	}

	return nil
}

type instructionBuilder interface {
	ValidateAndBuild() (*bindings.Instruction, error)
}

func sendAndConfirm[B instructionBuilder](
	ctx context.Context,
	client *rpc.Client,
	auth solana.PrivateKey,
	instructionBuilder B,
	commitmentType rpc.CommitmentType,
) (string, error) {
	builtInstruction, err := instructionBuilder.ValidateAndBuild()
	if err != nil {
		return "", fmt.Errorf("unable to validate and build instruction: %w", err)
	}

	result, err := solanaCommon.SendAndConfirm(ctx, client, []solana.Instruction{builtInstruction}, auth,
		commitmentType)
	if err != nil {
		return "", fmt.Errorf("unable to send instruction: %w", err)
	}
	if result.Transaction == nil {
		return "", fmt.Errorf("nil transacion in instruction result")
	}

	transaction, err := result.Transaction.GetTransaction()
	if err != nil {
		return "", fmt.Errorf("unable to get transaction from instruction result: %w", err)
	}

	return transaction.Signatures[0].String(), nil
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
