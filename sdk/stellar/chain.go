package stellar

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/types"
)

// ChainNetworkID returns the 32-byte Stellar network id (SHA-256 of passphrase, hex-encoded in chain-selectors)
// for the given MCMS chain selector.
func ChainNetworkID(sel types.ChainSelector) (common.Hash, error) {
	chainIDHex, err := chainsel.StellarChainIdFromSelector(uint64(sel))
	if err != nil {
		return common.Hash{}, fmt.Errorf("stellar chain id for selector %d: %w", sel, err)
	}
	chainIDHex = strings.TrimPrefix(strings.TrimPrefix(chainIDHex, "0x"), "0X")
	if len(chainIDHex) != stellarChainHexCharLen {
		return common.Hash{}, fmt.Errorf("unexpected stellar chain id length %d (want 64 hex chars)", len(chainIDHex))
	}
	raw, err := hex.DecodeString(chainIDHex)
	if err != nil {
		return common.Hash{}, fmt.Errorf("decode stellar chain id hex: %w", err)
	}

	return common.BytesToHash(raw), nil
}
