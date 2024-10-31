package core

import (
	"encoding/json"

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
