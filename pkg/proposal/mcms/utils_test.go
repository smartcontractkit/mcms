package mcms

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromFile(t *testing.T) {
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
	err = os.WriteFile(file.Name(), sampleJSON, 0644)
	require.NoError(t, err)

	// Call the FromFile function
	var result SampleStruct
	err = FromFile(file.Name(), &result)
	require.NoError(t, err)

	// Assert the result
	assert.Equal(t, sampleData, result)
}

func TestProposalFromFile(t *testing.T) {
	mcmsProposal := MCMSProposal{
		Version:              "1",
		ValidUntil:           100,
		Signatures:           []Signature{},
		Transactions:         []ChainOperation{},
		OverridePreviousRoot: false,
		Description:          "Test Proposal",
		ChainMetadata:        make(map[ChainIdentifier]ChainMetadata),
	}

	tempFile, err := os.CreateTemp("", "mcms.json")
	require.NoError(t, err)

	proposalBytes, err := json.Marshal(mcmsProposal)
	require.NoError(t, err)
	err = os.WriteFile(tempFile.Name(), proposalBytes, 0644)
	require.NoError(t, err)

	fileProposal, err := NewProposalFromFile(tempFile.Name())
	require.NoError(t, err)
	assert.Equal(t, mcmsProposal, *fileProposal)
}
