package solana

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
)

func FindSignerPDA(programID solana.PublicKey, pdaSeed PDASeed) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("multisig_signer"), pdaSeed[:]}
	pda, _, err := solana.FindProgramAddress(seeds, programID)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("unable to find multisig_signer pda: %w", err)
	}

	return pda, nil
}

func FindConfigPDA(programID solana.PublicKey, pdaSeed PDASeed) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("multisig_config"), pdaSeed[:]}
	pda, _, err := solana.FindProgramAddress(seeds, programID)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("unable to find multisig_config pda: %w", err)
	}

	return pda, nil
}

func FindConfigSignersPDA(programID solana.PublicKey, pdaSeed PDASeed) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("multisig_config_signers"), pdaSeed[:]}
	pda, _, err := solana.FindProgramAddress(seeds, programID)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("unable to find multisig_config_signers pda: %w", err)
	}

	return pda, nil
}

func FindRootMetadataPDA(programID solana.PublicKey, pdaSeed PDASeed) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("root_metadata"), pdaSeed[:]}
	pda, _, err := solana.FindProgramAddress(seeds, programID)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("unable to find root_metadata pda: %w", err)
	}

	return pda, nil
}

func FindExpiringRootAndOpCountPDA(programID solana.PublicKey, pdaSeed PDASeed) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("expiring_root_and_op_count"), pdaSeed[:]}
	pda, _, err := solana.FindProgramAddress(seeds, programID)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("unable to find expiring_root_and_op_count pda: %w", err)
	}

	return pda, nil
}

func FindRootSignaturesPDA(
	programID solana.PublicKey, pdaSeed PDASeed, root common.Hash, validUntil uint32,
) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("root_signatures"), pdaSeed[:], root[:], validUntilBytes(validUntil)}
	pda, _, err := solana.FindProgramAddress(seeds, programID)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("unable to find root_signatures pda: %w", err)
	}

	return pda, nil
}

func FindSeenSignedHashesPDA(
	programID solana.PublicKey, pdaSeed PDASeed, root common.Hash, validUntil uint32,
) (solana.PublicKey, error) {
	seeds := [][]byte{[]byte("seen_signed_hashes"), pdaSeed[:], root[:], validUntilBytes(validUntil)}
	pda, _, err := solana.FindProgramAddress(seeds, programID)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("unable to find seen_signed_hashes pda: %w", err)
	}

	return pda, nil
}

func validUntilBytes(validUntil uint32) []byte {
	const uint32Size = 4
	validUntilBytes := make([]byte, uint32Size)
	binary.LittleEndian.PutUint32(validUntilBytes, validUntil)

	return validUntilBytes
}
