package canton

import (
	"encoding/hex"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

// TimelockCallForHash is used for computing the operation ID hash.
// Field semantics match mcms.TimelockCall (TargetInstanceAddress, FunctionName, OperationData).
type TimelockCallForHash struct {
	TargetInstanceAddress string // Format: "instanceId@partyId"
	FunctionName          string
	OperationData         string
}

// HashTimelockOpId computes the operation ID for timelock operations.
// Matches Canton's hashTimelockOpId: keccak256(encodedCalls || predecessor || salt).
// predecessor and salt should be hex-encoded (e.g. 64-char hex for 32-byte hashes).
func HashTimelockOpId(calls []TimelockCallForHash, predecessor, salt string) string {
	var sb strings.Builder

	// Length prefix for calls array (32-byte padded)
	sb.WriteString(padLeft32(intToHex(len(calls))))

	// Encode each call with 32-byte padded length prefixes
	for _, call := range calls {
		// Length prefix for targetInstanceAddress (UTF-8 byte count)
		sb.WriteString(padLeft32(intToHex(len(call.TargetInstanceAddress))))
		sb.WriteString(asciiToHex(call.TargetInstanceAddress))
		// Length prefix for functionName (UTF-8 byte count)
		sb.WriteString(padLeft32(intToHex(len(call.FunctionName))))
		sb.WriteString(asciiToHex(call.FunctionName))
		// Length prefix for operationData (byte count = hex length / 2)
		opData := encodeOperationDataForHash(call.OperationData)
		sb.WriteString(padLeft32(intToHex(len(opData) / hexEncodedByteLen)))
		sb.WriteString(opData)
	}

	// Length prefix for predecessor (byte count = hex length / 2)
	sb.WriteString(padLeft32(intToHex(len(predecessor) / hexEncodedByteLen)))
	sb.WriteString(predecessor)

	// Length prefix for salt (byte count = hex length / 2)
	sb.WriteString(padLeft32(intToHex(len(salt) / hexEncodedByteLen)))
	sb.WriteString(salt)

	data, err := hex.DecodeString(sb.String())
	if err != nil {
		panic("HashTimelockOpId: invalid hex encoding: " + err.Error())
	}

	return hex.EncodeToString(crypto.Keccak256(data))
}

// encodeOperationDataForHash normalizes operationData for hashing:
// - If operationData is valid hex (even length, hex digits only), use as-is (matches Daml BytesHex type).
// - Otherwise, treat as ASCII and hex-encode it (SDK convenience for non-hex inputs).
func encodeOperationDataForHash(operationData string) string {
	if isValidHex(operationData) {
		return operationData
	}

	return asciiToHex(operationData)
}

func isValidHex(s string) bool {
	if len(s)%hexEncodedByteLen != 0 {
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
