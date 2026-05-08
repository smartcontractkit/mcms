package stellar

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stellar/go-stellar-sdk/strkey"
	"github.com/stellar/go-stellar-sdk/xdr"
)

// parseContractID parses a Stellar contract identifier as either:
//   - A contract strkey (base32, typically starting with 'C'), or
//   - 64 hex characters (optional 0x prefix) representing the raw 32-byte contract id.
func parseContractID(s string) (xdr.ContractId, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return xdr.ContractId{}, fmt.Errorf("empty contract id")
	}

	if common.IsHexHash(s) {
		return xdr.ContractId(common.HexToHash(s)), nil
	}

	raw, err := strkey.Decode(strkey.VersionByteContract, s)
	if err != nil {
		return xdr.ContractId{}, fmt.Errorf("decode contract strkey: %w", err)
	}
	if len(raw) != stellarContractIDBytes {
		return xdr.ContractId{}, fmt.Errorf("contract id must be 32 bytes, got %d", len(raw))
	}
	var h xdr.Hash
	copy(h[:], raw)

	return xdr.ContractId(h), nil
}
