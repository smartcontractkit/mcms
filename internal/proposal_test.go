package internal

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
)

var TestAddress = "0x1234567890abcdef"
var TestChain1 = mcms.ChainSelector(3379446385462418246)
var TestChain2 = mcms.ChainSelector(16015286601757825753)
var TestChain3 = mcms.ChainSelector(10344971235874465080)

// func TestCalculateTransactionCounts(t *testing.T) {
// 	t.Parallel()

// 	transactions := []ChainOperation{
// 		{ChainSelector: TestChain1},
// 		{ChainSelector: TestChain1},
// 		{ChainSelector: TestChain2},
// 	}

// 	expected := map[ChainSelector]uint64{
// 		TestChain1: 2,
// 		TestChain2: 1,
// 	}

// 	result := calculateTransactionCounts(transactions)
// 	assert.Equal(t, expected, result)
// }

// func TestSortedChainIdentifiers(t *testing.T) {
// 	t.Parallel()

// 	chainMetadata := map[ChainIdentifier]ChainMetadata{
// 		TestChain2: {},
// 		TestChain1: {},
// 		TestChain3: {},
// 	}

// 	expected := []ChainIdentifier{TestChain1, TestChain3, TestChain2}

// 	result := sortedChainIdentifiers(chainMetadata)
// 	assert.Equal(t, expected, result)
// }

func TestMCMSOnlyProposal_Validate_Success(t *testing.T) {
	t.Parallel()

	proposal, err := NewProposal(
		"1.0",
		2004259681,
		[]mcms.Signature{},
		false,
		map[mcms.ChainSelector]mcms.ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		"Sample description",
		[]mcms.ChainOperation{
			{
				ChainSelector: TestChain1,
				Operation: mcms.Operation{
					To:               TestAddress,
					AdditionalFields: json.RawMessage([]byte(`{"value": "0"}`)),
					Data:             common.Hex2Bytes("0x"),
					OperationMetadata: mcms.OperationMetadata{
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
		[]mcms.Signature{},
		false,
		map[mcms.ChainSelector]mcms.ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		"Sample description",
		[]mcms.ChainOperation{
			{
				ChainSelector: TestChain1,
				Operation: mcms.Operation{
					To:               TestAddress,
					AdditionalFields: json.RawMessage([]byte(`{"value": "0"}`)),
					Data:             common.Hex2Bytes("0x"),
					OperationMetadata: mcms.OperationMetadata{
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
		[]mcms.Signature{},
		false,
		map[mcms.ChainSelector]mcms.ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		"Sample description",
		[]mcms.ChainOperation{
			{
				ChainSelector: TestChain1,
				Operation: mcms.Operation{
					To:               TestAddress,
					AdditionalFields: json.RawMessage([]byte(`{"value": "0"}`)),
					Data:             common.Hex2Bytes("0x"),
					OperationMetadata: mcms.OperationMetadata{
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
		[]mcms.Signature{},
		false,
		map[mcms.ChainSelector]mcms.ChainMetadata{},
		"Sample description",
		[]mcms.ChainOperation{
			{
				ChainSelector: TestChain1,
				Operation: mcms.Operation{
					To:               TestAddress,
					AdditionalFields: json.RawMessage([]byte(`{"value": "0"}`)),
					Data:             common.Hex2Bytes("0x"),
					OperationMetadata: mcms.OperationMetadata{
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
		[]mcms.Signature{},
		false,
		map[mcms.ChainSelector]mcms.ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		"",
		[]mcms.ChainOperation{
			{
				ChainSelector: TestChain1,
				Operation: mcms.Operation{
					To:               TestAddress,
					AdditionalFields: json.RawMessage([]byte(`{"value": "0"}`)),
					Data:             common.Hex2Bytes("0x"),
					OperationMetadata: mcms.OperationMetadata{
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
		[]mcms.Signature{},
		false,
		map[mcms.ChainSelector]mcms.ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		"Sample description",
		[]mcms.ChainOperation{},
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
		[]mcms.Signature{},
		false,
		map[mcms.ChainSelector]mcms.ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		"Sample description",
		[]mcms.ChainOperation{
			{
				ChainSelector: 3,
				Operation: mcms.Operation{
					To:               TestAddress,
					AdditionalFields: json.RawMessage([]byte(`{"value": "0"}`)),
					Data:             common.Hex2Bytes("0x"),
					OperationMetadata: mcms.OperationMetadata{
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
		Signatures:           []mcms.Signature{},
		Transactions:         []mcms.ChainOperation{},
		OverridePreviousRoot: false,
		Description:          "Test Proposal",
		ChainMetadata:        make(map[mcms.ChainSelector]mcms.ChainMetadata),
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
