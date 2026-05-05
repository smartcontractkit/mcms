package stellar

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/stellar/go/strkey"
)

// ParseContractID parses a Stellar contract identifier as either:
//   - A contract strkey (base32, typically starting with 'C'), or
//   - 64 hex characters (optional 0x prefix) representing the raw 32-byte contract id.
func ParseContractID(s string) ([32]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return [32]byte{}, fmt.Errorf("empty contract id")
	}

	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		s = s[2:]
	}

	if len(s) == 64 && isHex(s) {
		raw, err := hex.DecodeString(s)
		if err != nil {
			return [32]byte{}, fmt.Errorf("decode hex contract id: %w", err)
		}
		var out [32]byte
		copy(out[:], raw)
		return out, nil
	}

	raw, err := strkey.Decode(strkey.VersionByteContract, s)
	if err != nil {
		return [32]byte{}, fmt.Errorf("decode contract strkey: %w", err)
	}
	if len(raw) != 32 {
		return [32]byte{}, fmt.Errorf("contract id must be 32 bytes, got %d", len(raw))
	}
	var out [32]byte
	copy(out[:], raw)
	return out, nil
}

func isHex(s string) bool {
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		case c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}
