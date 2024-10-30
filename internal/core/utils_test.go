package core

import (
	"encoding/json"

	"math"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define a sample struct
type SampleStruct struct {
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email"`
}

func TestFromFile(t *testing.T) {
	t.Parallel()

	// Create a temporary file for testing
	file, err := os.CreateTemp("", "testfile")
	require.NoError(t, err)
	defer os.Remove(file.Name())

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

func TestWriteProposalToFile(t *testing.T) {
	t.Parallel()
	// additionalFields, err := json.Marshal(struct {
	// 	Value *big.Int `json:"value"`
	// }{
	// 	Value: big.NewInt(0),
	// })
	// require.NoError(t, err)
	// Define a sample proposal struct
	proposal := SampleStruct{
		Name:  "John Doe",
		Age:   30,
		Email: "johndoe@example.com",
	}

	// Create a temporary file
	tempFile, err := os.CreateTemp("", "proposal.json")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Write the proposal to file
	err = WriteProposalToFile(proposal, tempFile.Name())
	require.NoError(t, err)

	// Read back the file and assert the content
	var fileProposal SampleStruct
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
