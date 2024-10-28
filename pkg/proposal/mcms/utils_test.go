package mcms

import (
	"encoding/hex"
	"encoding/json"
	"github.com/smartcontractkit/mcms/pkg/proposal/mcms/types"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"math"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromFile(t *testing.T) {
	t.Parallel()

	// Create a temporary file for testing
	file, err := os.CreateTemp("", "testfile")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	// Define a sample struct
	type SampleStruct struct {
		Name  string `json:"name"`
		Age   int    `json:"age"`
		Email string `json:"email"`
	}

	// Create a sample JSON file
	sampleData := SampleStruct{
		Name:  "John Doe",
		Age:   30,
		Email: "johndoe@example.com",
	}
	sampleJSON, err := json.Marshal(sampleData)
	require.NoError(t, err)
	err = os.WriteFile(file.Name(), sampleJSON, 0600)
	require.NoError(t, err)

	// Call the FromFile function
	var result SampleStruct
	err = FromFile(file.Name(), &result)
	require.NoError(t, err)

	// Assert the result
	assert.Equal(t, sampleData, result)
}

func TestProposalFromFile(t *testing.T) {
	t.Parallel()
	additionalFields, err := json.Marshal(struct {
		Value *big.Int `json:"value"`
	}{
		Value: big.NewInt(0),
	})
	require.NoError(t, err)
	mcmsProposal := MCMSProposal{
		Version:    "1",
		ValidUntil: 4128029039,
		Signatures: []Signature{},
		Transactions: []types.ChainOperation{
			{
				ChainIdentifier: types.ChainIdentifier(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
				Operation: types.Operation{
					AdditionalFields: additionalFields,
				},
			},
		},
		OverridePreviousRoot: false,
		Description:          "Test Proposal",
		ChainMetadata: map[types.ChainIdentifier]ChainMetadata{
			types.ChainIdentifier(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector): {
				StartingOpCount: 0,
				MCMAddress:      common.HexToAddress("0x5b38da6a701c568545dcfcb03fcb875f56beddc4"),
			},
		},
	}

	tempFile, err := os.CreateTemp("", "mcms.json")
	require.NoError(t, err)

	proposalBytes, err := json.Marshal(mcmsProposal)
	require.NoError(t, err)
	err = os.WriteFile(tempFile.Name(), proposalBytes, 0600)
	require.NoError(t, err)

	fileProposal, err := NewProposalFromFile(tempFile.Name())
	require.NoError(t, err)
	assert.EqualValues(t, mcmsProposal, *fileProposal)
}

func TestWriteProposalToFile(t *testing.T) {
	t.Parallel()
	additionalFields, err := json.Marshal(struct {
		Value *big.Int `json:"value"`
	}{
		Value: big.NewInt(0),
	})
	require.NoError(t, err)
	// Define a sample proposal struct
	proposal := MCMSProposal{
		Version:    "1",
		ValidUntil: 4128029039,
		Signatures: []Signature{},
		Transactions: []types.ChainOperation{
			{
				ChainIdentifier: types.ChainIdentifier(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
				Operation: types.Operation{
					AdditionalFields: additionalFields,
				},
			},
		},
		OverridePreviousRoot: false,
		Description:          "Test Proposal",
		ChainMetadata: map[types.ChainIdentifier]ChainMetadata{
			types.ChainIdentifier(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector): {
				StartingOpCount: 0,
				MCMAddress:      common.HexToAddress("0x5b38da6a701c568545dcfcb03fcb875f56beddc4"),
			},
		},
	}

	// Create a temporary file
	tempFile, err := os.CreateTemp("", "proposal.json")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Write the proposal to file
	err = WriteProposalToFile(proposal, tempFile.Name())
	require.NoError(t, err)

	// Read back the file and assert the content
	var fileProposal MCMSProposal
	err = FromFile(tempFile.Name(), &fileProposal)
	require.NoError(t, err)
	assert.Equal(t, proposal, fileProposal)
}

func TestSafeCastIntToUint32(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		value       int
		expected    uint32
		expectError bool
	}{
		{name: "Valid int within range", value: 42, expected: 42, expectError: false},
		{name: "Negative int", value: -1, expected: 0, expectError: true},
		{name: "Int exceeds uint32 max value", value: int(math.MaxUint32 + 1), expected: 0, expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := SafeCastIntToUint32(tt.value)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestABIDecode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		abiStr      string
		dataHex     string
		expected    []any
		expectError bool
	}{
		{
			name:    "Decode single uint256",
			abiStr:  `[{"type":"uint256"}]`,
			dataHex: "000000000000000000000000000000000000000000000000000000000000001e", // 30 in uint256
			expected: []any{
				big.NewInt(30),
			},
			expectError: false,
		},
		{
			name:        "Decode address",
			abiStr:      `[{"type":"address"}]`,
			dataHex:     "0000000000000000000000005b38da6a701c568545dcfcb03fcb875f56beddc4", // valid Ethereum address
			expected:    []any{common.HexToAddress("0x5b38da6a701c568545dcfcb03fcb875f56beddc4")},
			expectError: false,
		},
		{
			name:   "Decode string",
			abiStr: `[{"type":"string"}]`,
			dataHex: "0000000000000000000000000000000000000000000000000000000000000020" + // offset (32 bytes)
				"000000000000000000000000000000000000000000000000000000000000000b" + // string length (11 bytes)
				"48656c6c6f20576f726c64000000000000000000000000000000000000000000", // "Hello World"
			expected:    []any{"Hello World"},
			expectError: false,
		},
		{
			name:        "Invalid data",
			abiStr:      `[{"type":"uint256"}]`,
			dataHex:     "00000000000000000000000000000000", // Too short for uint256
			expected:    nil,
			expectError: true,
		},
		{
			name:        "Invalid ABI string",
			abiStr:      `[{"type":"invalid"}]`,                                             // Invalid ABI type
			dataHex:     "000000000000000000000000000000000000000000000000000000000000001e", // 30 in uint256
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := hex.DecodeString(tt.dataHex)
			require.NoError(t, err)

			result, err := ABIDecode(tt.abiStr, data)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSafeCastInt64ToUint32(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		value       int64
		expected    uint32
		expectError bool
	}{
		{name: "Valid int64 within range", value: 42, expected: 42, expectError: false},
		{name: "Negative int64", value: -1, expected: 0, expectError: true},
		{name: "Int64 exceeds uint32 max value", value: int64(math.MaxUint32 + 1), expected: 0, expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := SafeCastInt64ToUint32(tt.value)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSafeCastUint64ToUint8(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		value       uint64
		expected    uint8
		expectError bool
	}{
		{name: "Valid uint64 within range", value: 42, expected: 42, expectError: false},
		{name: "Uint64 exceeds uint8 max value", value: uint64(math.MaxUint8 + 1), expected: 0, expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := SafeCastUint64ToUint8(tt.value)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
