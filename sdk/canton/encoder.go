package canton

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

// AdditionalFields represents the additional fields in Canton MCMS operations
type AdditionalFields struct {
	TargetInstanceId string   `json:"targetInstanceId"`
	FunctionName     string   `json:"functionName"`
	OperationData    string   `json:"operationData"`
	TargetCid        string   `json:"targetCid"`
	ContractIds      []string `json:"contractIds"`
}

var _ sdk.Encoder = &Encoder{}

type Encoder struct {
	ChainSelector        types.ChainSelector
	TxCount              uint64
	OverridePreviousRoot bool
}

func NewEncoder(
	chainSelector types.ChainSelector,
	txCount uint64,
	overridePreviousRoot bool,
) *Encoder {
	return &Encoder{
		ChainSelector:        chainSelector,
		TxCount:              txCount,
		OverridePreviousRoot: overridePreviousRoot,
	}
}

// HashOperation hashes an operation to get its Merkle leaf
// Matches Canton's hashOpLeafNative from Crypto.daml
func (e *Encoder) HashOperation(opCount uint32, metadata types.ChainMetadata, op types.Operation) (common.Hash, error) {
	// Unmarshal Canton-specific metadata
	var metadataFields AdditionalFieldsMetadata
	if err := json.Unmarshal(metadata.AdditionalFields, &metadataFields); err != nil {
		return common.Hash{}, fmt.Errorf("failed to unmarshal metadata additional fields: %w", err)
	}

	// Unmarshal Canton-specific operation fields
	var opFields AdditionalFields
	if err := json.Unmarshal(op.Transaction.AdditionalFields, &opFields); err != nil {
		return common.Hash{}, fmt.Errorf("failed to unmarshal operation additional fields: %w", err)
	}

	// Build the encoded data following Canton's hashOpLeafNative:
	encoded := padLeft32(intToHex(int(metadataFields.ChainId))) +
		asciiToHex(metadataFields.MultisigId) +
		padLeft32(intToHex(int(opCount))) +
		asciiToHex(opFields.TargetInstanceId) +
		asciiToHex(opFields.FunctionName) +
		opFields.OperationData

	// Decode hex string and hash
	data, err := hex.DecodeString(encoded)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to decode hex string: %w", err)
	}

	return crypto.Keccak256Hash(data), nil
}

// HashMetadata hashes metadata to get its Merkle leaf
// Matches Canton's hashMetadataLeafNative from Crypto.daml
func (e *Encoder) HashMetadata(metadata types.ChainMetadata) (common.Hash, error) {
	// Unmarshal Canton-specific metadata
	var metadataFields AdditionalFieldsMetadata
	if err := json.Unmarshal(metadata.AdditionalFields, &metadataFields); err != nil {
		return common.Hash{}, fmt.Errorf("failed to unmarshal metadata additional fields: %w", err)
	}

	// Build override flag
	overrideFlag := "00"
	if metadataFields.OverridePreviousRoot {
		overrideFlag = "01"
	}

	encoded := padLeft32(intToHex(int(metadataFields.ChainId))) +
		asciiToHex(metadataFields.MultisigId) +
		padLeft32(intToHex(int(metadataFields.PreOpCount))) +
		padLeft32(intToHex(int(metadataFields.PostOpCount))) +
		overrideFlag

	// Decode hex string and hash
	data, err := hex.DecodeString(encoded)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to decode hex string: %w", err)
	}

	return crypto.Keccak256Hash(data), nil
}

// Helper functions matching Canton Crypto.daml

// padLeft32 pads hex string to 64 chars (32 bytes)
func padLeft32(hexStr string) string {
	if len(hexStr) >= 64 {
		return hexStr[:64]
	}
	return strings.Repeat("0", 64-len(hexStr)) + hexStr
}

// intToHex converts int to hex string (without padding)
func intToHex(n int) string {
	if n == 0 {
		return "0"
	}
	return fmt.Sprintf("%x", n)
}

// asciiToHex converts ASCII string to hex
func asciiToHex(s string) string {
	return hex.EncodeToString([]byte(s))
}
