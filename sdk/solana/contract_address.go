package solana

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/gagliardetto/solana-go"
)

type PDASeed [32]byte // FIXME: shouldn't this be defined in chainlink-ccip/chains/solana?

// ContractAddress returns a string representation of a solana contract id
// which is a combination of the program id and the seed <PROGRAM_ID>.<SEED>
func ContractAddress(programID solana.PublicKey, pdaSeed PDASeed) string {
	return fmt.Sprintf("%s.%s", programID.String(), bytes.Trim(pdaSeed[:], "\x00"))
}

func ParseContractAddress(address string) (solana.PublicKey, PDASeed, error) {
	const numParts = 2
	parts := strings.SplitN(address, ".", numParts)
	if len(parts) != numParts {
		return solana.PublicKey{}, PDASeed{}, fmt.Errorf("invalid solana contract address format: %q", address)
	}

	programID, err := solana.PublicKeyFromBase58(parts[0])
	if err != nil {
		return solana.PublicKey{}, PDASeed{}, fmt.Errorf("unable to parse solana program id: %w", err)
	}

	allSeedBytes := []byte(parts[1])
	if len(allSeedBytes) > len(PDASeed{}) {
		return solana.PublicKey{}, PDASeed{}, fmt.Errorf("pda seed is too long (max %d bytes)", len(PDASeed{}))
	}

	var pdaSeed PDASeed
	copy(pdaSeed[:], []byte(parts[1])[:])

	return programID, pdaSeed, nil
}
