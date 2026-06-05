package canton

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

// Domain separators matching Canton's Crypto.daml
var (
	opLeafDomainSeparator = hex.EncodeToString(crypto.Keccak256(
		[]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_OP_CANTON")))
	metadataLeafDomainSeparator = hex.EncodeToString(crypto.Keccak256(
		[]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA_CANTON")))
)

// AdditionalFields represents the additional fields in Canton MCMS operations
type AdditionalFields struct {
	TargetInstanceAddress string   `json:"targetInstanceAddress"` // Format: "instanceId@partyId"
	FunctionName          string   `json:"functionName"`
	TargetCid             string   `json:"targetCid"`
	ContractIds           []string `json:"contractIds"`
	// TargetTemplateID is the Daml template ID of the target contract (e.g. "#pkg:Module:Entity").
	// When TargetCid is empty at execution time, the SDK uses TargetTemplateID + TargetInstanceAddress
	// to dynamically resolve the active contract ID from the ledger.
	TargetTemplateID string `json:"targetTemplateId,omitempty"`
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

// ToRootMetadata resolves Canton root metadata for SetRoot and merkle hashing.
// Like EVM ToGethRootMetadata, postOpCount is derived from StartingOpCount + TxCount at sign time.
// multisigId and related fields must be present in chainMetadata.additionalFields (typically from GetRootMetadata).
func (e *Encoder) ToRootMetadata(metadata types.ChainMetadata) (AdditionalFieldsMetadata, error) {
	fields, err := parseAdditionalFieldsMetadata(metadata.AdditionalFields)
	if err != nil {
		return AdditionalFieldsMetadata{}, err
	}

	if err := fields.Validate(); err != nil {
		return AdditionalFieldsMetadata{}, fmt.Errorf("canton chain metadata additionalFields required or incomplete: %w", err)
	}

	return fields, nil
}

func parseAdditionalFieldsMetadata(raw json.RawMessage) (AdditionalFieldsMetadata, error) {
	if len(raw) == 0 {
		return AdditionalFieldsMetadata{}, fmt.Errorf("canton chain metadata additionalFields are required")
	}

	var fields AdditionalFieldsMetadata
	if err := json.Unmarshal(raw, &fields); err != nil {
		return AdditionalFieldsMetadata{}, fmt.Errorf("unmarshal metadata additional fields: %w", err)
	}

	return fields, nil
}

// HashOperation hashes an operation to get its Merkle leaf
// Matches Canton's hashOpLeafNative from Crypto.daml with domain separator and length prefixes
func (e *Encoder) HashOperation(opCount uint32, metadata types.ChainMetadata, op types.Operation) (common.Hash, error) {
	metadataFields, err := e.ToRootMetadata(metadata)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to resolve metadata additional fields: %w", err)
	}

	// Unmarshal Canton-specific operation fields
	var opFields AdditionalFields
	if unmarshalErr := json.Unmarshal(op.Transaction.AdditionalFields, &opFields); unmarshalErr != nil {
		return common.Hash{}, fmt.Errorf("failed to unmarshal operation additional fields: %w", unmarshalErr)
	}
	if validateErr := opFields.Validate(); validateErr != nil {
		return common.Hash{}, fmt.Errorf("invalid operation additional fields: %w", validateErr)
	}

	operationData := operationDataHex(op.Transaction.Data)

	// Convert variable-length fields to hex
	multisigIdHex := asciiToHex(metadataFields.MultisigId)
	targetAddressHex := asciiToHex(opFields.TargetInstanceAddress)
	functionNameHex := asciiToHex(opFields.FunctionName)

	// Build the encoded data following Canton's hashOpLeafNative with domain separator and length prefixes
	encoded := opLeafDomainSeparator +
		padLeft32(intToHex(int(metadataFields.ChainId))) +
		padLeft32(intToHex(len(metadataFields.MultisigId))) + // Length prefix for multisigId
		multisigIdHex +
		padLeft32(intToHex(int(opCount))) +
		padLeft32(intToHex(len(opFields.TargetInstanceAddress))) + // Length prefix for targetInstanceAddress
		targetAddressHex +
		padLeft32(intToHex(len(opFields.FunctionName))) + // Length prefix for functionName
		functionNameHex +
		padLeft32(intToHex(len(operationData)/hexEncodedByteLen)) + // Length prefix for operationData (byte count)
		operationData

	// Decode hex string and hash
	data, err := hex.DecodeString(encoded)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to decode hex string: %w", err)
	}

	return crypto.Keccak256Hash(data), nil
}

// HashMetadata hashes metadata to get its Merkle leaf
// Matches Canton's hashMetadataLeafNative from Crypto.daml with domain separator and length prefixes
func (e *Encoder) HashMetadata(metadata types.ChainMetadata) (common.Hash, error) {
	metadataFields, err := e.ToRootMetadata(metadata)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to resolve metadata additional fields: %w", err)
	}

	// Build override flag
	overrideFlag := "00"
	if e.OverridePreviousRoot {
		overrideFlag = "01"
	}

	// Convert multisigId to hex
	multisigIdHex := asciiToHex(metadataFields.MultisigId)

	// Build the encoded data with domain separator and length prefix for multisigId
	encoded := metadataLeafDomainSeparator +
		padLeft32(intToHex(int(metadataFields.ChainId))) +
		padLeft32(intToHex(len(metadataFields.MultisigId))) + // Length prefix for multisigId
		multisigIdHex +
		padLeft32(uint64ToHex(metadata.StartingOpCount)) +
		padLeft32(uint64ToHex(metadata.StartingOpCount+e.TxCount)) +
		overrideFlag

	// Decode hex string and hash
	data, err := hex.DecodeString(encoded)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to decode hex string: %w", err)
	}

	return crypto.Keccak256Hash(data), nil
}

// Helper functions matching Canton Crypto.daml

// padLeft32 pads hex string to 64 chars (32 bytes). Panics if input exceeds 32 bytes,
// matching Canton's Crypto.daml behavior.
func padLeft32(hexStr string) string {
	if len(hexStr) > hexWordLen {
		panic(fmt.Sprintf("padLeft32: input exceeds 32 bytes: %d hex chars", len(hexStr)))
	}
	if len(hexStr) == hexWordLen {
		return hexStr
	}

	return strings.Repeat("0", hexWordLen-len(hexStr)) + hexStr
}

// intToHex converts a non-negative int to hex string (without padding). Panics on negative input,
// matching Canton's Crypto.daml behavior.
func intToHex(n int) string {
	if n < 0 {
		panic("intToHex: negative numbers not supported")
	}

	return fmt.Sprintf("%x", n)
}

func uint64ToHex(n uint64) string {
	return strconv.FormatUint(n, 16)
}

// asciiToHex converts ASCII string to hex
func asciiToHex(s string) string {
	return hex.EncodeToString([]byte(s))
}
