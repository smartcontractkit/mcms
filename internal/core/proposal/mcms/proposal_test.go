package mcms

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var TestAddress = "0x1234567890abcdef"
var TestChain1 = ChainSelector(3379446385462418246)
var TestChain2 = ChainSelector(16015286601757825753)
var TestChain3 = ChainSelector(10344971235874465080)

func TestMCMSOnlyProposal_Validate_Success(t *testing.T) {
	t.Parallel()

	proposal, err := NewProposal(
		"1.0",
		2004259681,
		[]Signature{},
		false,
		map[ChainSelector]ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		"Sample description",
		[]ChainOperation{
			{
				ChainSelector: TestChain1,
				Operation: Operation{
					To:               TestAddress,
					AdditionalFields: json.RawMessage([]byte(`{"value": "0"}`)),
					Data:             common.Hex2Bytes("0x"),
					OperationMetadata: OperationMetadata{
						ContractType: "Sample contract",
						Tags:         []string{"tag1", "tag2"},
					},
				},
			},
		},
	)

	require.NoError(t, err)
	assert.NotNil(t, proposal)
}

func TestMCMSOnlyProposal_Validate_InvalidVersion(t *testing.T) {
	t.Parallel()

	proposal, err := NewProposal(
		"",
		2004259681,
		[]Signature{},
		false,
		map[ChainSelector]ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		"Sample description",
		[]ChainOperation{
			{
				ChainSelector: TestChain1,
				Operation: Operation{
					To:               TestAddress,
					AdditionalFields: json.RawMessage([]byte(`{"value": "0"}`)),
					Data:             common.Hex2Bytes("0x"),
					OperationMetadata: OperationMetadata{
						ContractType: "Sample contract",
						Tags:         []string{"tag1", "tag2"},
					},
				},
			},
		},
	)

	require.Error(t, err)
	require.EqualError(t, err, "invalid version: ")
	assert.Nil(t, proposal)
}

func TestMCMSOnlyProposal_Validate_InvalidValidUntil(t *testing.T) {
	t.Parallel()

	proposal, err := NewProposal(
		"1.0",
		0,
		[]Signature{},
		false,
		map[ChainSelector]ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		"Sample description",
		[]ChainOperation{
			{
				ChainSelector: TestChain1,
				Operation: Operation{
					To:               TestAddress,
					AdditionalFields: json.RawMessage([]byte(`{"value": "0"}`)),
					Data:             common.Hex2Bytes("0x"),
					OperationMetadata: OperationMetadata{
						ContractType: "Sample contract",
						Tags:         []string{"tag1", "tag2"},
					},
				},
			},
		},
	)

	require.Error(t, err)
	require.EqualError(t, err, "invalid valid until: 0")
	assert.Nil(t, proposal)
}

func TestMCMSOnlyProposal_Validate_InvalidChainMetadata(t *testing.T) {
	t.Parallel()

	proposal, err := NewProposal(
		"1.0",
		2004259681,
		[]Signature{},
		false,
		map[ChainSelector]ChainMetadata{},
		"Sample description",
		[]ChainOperation{
			{
				ChainSelector: TestChain1,
				Operation: Operation{
					To:               TestAddress,
					AdditionalFields: json.RawMessage([]byte(`{"value": "0"}`)),
					Data:             common.Hex2Bytes("0x"),
					OperationMetadata: OperationMetadata{
						ContractType: "Sample contract",
						Tags:         []string{"tag1", "tag2"},
					},
				},
			},
		},
	)

	require.Error(t, err)
	require.EqualError(t, err, "no chain metadata")
	assert.Nil(t, proposal)
}

func TestMCMSOnlyProposal_Validate_InvalidDescription(t *testing.T) {
	t.Parallel()

	proposal, err := NewProposal(
		"1.0",
		2004259681,
		[]Signature{},
		false,
		map[ChainSelector]ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		"",
		[]ChainOperation{
			{
				ChainSelector: TestChain1,
				Operation: Operation{
					To:               TestAddress,
					AdditionalFields: json.RawMessage([]byte(`{"value": "0"}`)),
					Data:             common.Hex2Bytes("0x"),
					OperationMetadata: OperationMetadata{
						ContractType: "Sample contract",
						Tags:         []string{"tag1", "tag2"},
					},
				},
			},
		},
	)

	require.Error(t, err)
	require.EqualError(t, err, "invalid description: ")
	assert.Nil(t, proposal)
}

func TestMCMSOnlyProposal_Validate_NoTransactions(t *testing.T) {
	t.Parallel()

	proposal, err := NewProposal(
		"1.0",
		2004259681,
		[]Signature{},
		false,
		map[ChainSelector]ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		"Sample description",
		[]ChainOperation{},
	)

	require.Error(t, err)
	require.EqualError(t, err, "no transactions")
	assert.Nil(t, proposal)
}

func TestMCMSOnlyProposal_Validate_MissingChainMetadataForTransaction(t *testing.T) {
	t.Parallel()

	proposal, err := NewProposal(
		"1.0",
		2004259681,
		[]Signature{},
		false,
		map[ChainSelector]ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		"Sample description",
		[]ChainOperation{
			{
				ChainSelector: 3,
				Operation: Operation{
					To:               TestAddress,
					AdditionalFields: json.RawMessage([]byte(`{"value": "0"}`)),
					Data:             common.Hex2Bytes("0x"),
					OperationMetadata: OperationMetadata{
						ContractType: "Sample contract",
						Tags:         []string{"tag1", "tag2"},
					},
				},
			},
		},
	)

	require.Error(t, err)
	require.EqualError(t, err, "missing chain metadata for chain 3")
	assert.Nil(t, proposal)
}

func TestProposalFromFile(t *testing.T) {
	t.Parallel()

	mcmsProposal := MCMSProposal{
		Version:              "1",
		ValidUntil:           100,
		Signatures:           []Signature{},
		Transactions:         []ChainOperation{},
		OverridePreviousRoot: false,
		Description:          "Test Proposal",
		ChainMetadata:        make(map[ChainSelector]ChainMetadata),
	}

	tempFile, err := os.CreateTemp("", "mcms.json")
	require.NoError(t, err)

	proposalBytes, err := json.Marshal(mcmsProposal)
	require.NoError(t, err)
	err = os.WriteFile(tempFile.Name(), proposalBytes, 0600)
	require.NoError(t, err)

	fileProposal, err := NewProposalFromFile(tempFile.Name())
	require.NoError(t, err)
	assert.Equal(t, mcmsProposal, *fileProposal)
}
