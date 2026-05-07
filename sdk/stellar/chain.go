package stellar

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/types"
)

// chainNetworkID returns the 32-byte Stellar network id (SHA-256 of passphrase, hex-encoded in chain-selectors)
// for the given MCMS chain selector.
func chainNetworkID(sel types.ChainSelector) (common.Hash, error) {
	chainIDHex, err := chainsel.StellarChainIdFromSelector(uint64(sel))
	if err != nil {
		return common.Hash{}, fmt.Errorf("stellar chain id for selector %d: %w", sel, err)
	}
	if !common.IsHexHash(chainIDHex) {
		return common.Hash{}, fmt.Errorf("unexpected stellar chain id %q (want 64 hex chars, optional 0x prefix)", chainIDHex)
	}

	return common.HexToHash(chainIDHex), nil
}
