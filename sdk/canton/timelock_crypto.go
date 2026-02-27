package canton

import (
	"encoding/hex"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

// TimelockCallForHash is used for computing the operation ID hash.
// Field semantics match mcms.TimelockCall (TargetInstanceId, FunctionName, OperationData).
type TimelockCallForHash struct {
	TargetInstanceId string
	FunctionName     string
	OperationData    string
}

// HashTimelockOpId computes the operation ID for timelock operations.
// Matches Canton's hashTimelockOpId: keccak256(encodedCalls || predecessor || salt).
// predecessor and salt should be hex-encoded (e.g. 64-char hex for 32-byte hashes).
func HashTimelockOpId(calls []TimelockCallForHash, predecessor, salt string) string {
	var sb strings.Builder
	for _, call := range calls {
		sb.WriteString(asciiToHex(call.TargetInstanceId))
		sb.WriteString(asciiToHex(call.FunctionName))
		sb.WriteString(encodeOperationDataForHash(call.OperationData))
	}
	sb.WriteString(asciiToHex(predecessor))
	sb.WriteString(asciiToHex(salt))

	data, err := hex.DecodeString(sb.String())
	if err != nil {
		panic("HashTimelockOpId: invalid hex encoding: " + err.Error())
	}

	return hex.EncodeToString(crypto.Keccak256(data))
}

// encodeOperationDataForHash matches on-chain MCMS.Crypto.encodeOperationData:
// - If operationData is valid hex (even length, hex digits only), use as-is.
// - Otherwise, treat as ASCII and hex-encode it.
func encodeOperationDataForHash(operationData string) string {
	if isValidHex(operationData) {
		return operationData
	}
	return asciiToHex(operationData)
}

func isValidHex(s string) bool {
	if len(s)%2 != 0 {
		return false
	}
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
