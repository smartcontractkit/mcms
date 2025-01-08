package solana

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
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
