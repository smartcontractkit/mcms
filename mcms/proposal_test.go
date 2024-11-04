package mcms

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
)

var TestAddress = "0x1234567890abcdef"
var TestChain1 = types.ChainSelector(3379446385462418246)
var TestChain2 = types.ChainSelector(16015286601757825753)
var TestChain3 = types.ChainSelector(10344971235874465080)

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

func TestMCMSOnlyProposal_Validate_Success(t *testing.T) {
	t.Parallel()

	proposal, err := NewProposal(
		"1.0",
		2004259681,
		[]types.Signature{},
		false,
		map[types.ChainSelector]types.ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		"Sample description",
		[]types.ChainOperation{
			{
				ChainSelector: TestChain1,
				Operation: types.Operation{
					To:               TestAddress,
					AdditionalFields: json.RawMessage([]byte(`{"value": "0"}`)),
					Data:             common.Hex2Bytes("0x"),
					OperationMetadata: types.OperationMetadata{
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
		[]types.Signature{},
		false,
		map[types.ChainSelector]types.ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		"Sample description",
		[]types.ChainOperation{
			{
				ChainSelector: TestChain1,
				Operation: types.Operation{
					To:               TestAddress,
					AdditionalFields: json.RawMessage([]byte(`{"value": "0"}`)),
					Data:             common.Hex2Bytes("0x"),
					OperationMetadata: types.OperationMetadata{
						ContractType: "Sample contract",
						Tags:         []string{"tag1", "tag2"},
					},
				},
			},
		},
	)

	require.Error(t, err)
	require.EqualError(t, err, "Key: 'MCMSProposal.BaseProposal.Version' Error:Field validation for 'Version' failed on the 'required' tag")
	assert.Nil(t, proposal)
}

func TestMCMSOnlyProposal_Validate_InvalidValidUntil(t *testing.T) {
	t.Parallel()

	proposal, err := NewProposal(
		"1.0",
		0,
		[]types.Signature{},
		false,
		map[types.ChainSelector]types.ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		"Sample description",
		[]types.ChainOperation{
			{
				ChainSelector: TestChain1,
				Operation: types.Operation{
					To:               TestAddress,
					AdditionalFields: json.RawMessage([]byte(`{"value": "0"}`)),
					Data:             common.Hex2Bytes("0x"),
					OperationMetadata: types.OperationMetadata{
						ContractType: "Sample contract",
						Tags:         []string{"tag1", "tag2"},
					},
				},
			},
		},
	)

	require.Error(t, err)
	require.EqualError(t, err, "Key: 'MCMSProposal.BaseProposal.ValidUntil' Error:Field validation for 'ValidUntil' failed on the 'required' tag")
	assert.Nil(t, proposal)
}

func TestMCMSOnlyProposal_Validate_InvalidChainMetadata(t *testing.T) {
	t.Parallel()

	proposal, err := NewProposal(
		"1.0",
		2004259681,
		[]types.Signature{},
		false,
		map[types.ChainSelector]types.ChainMetadata{},
		"Sample description",
		[]types.ChainOperation{
			{
				ChainSelector: TestChain1,
				Operation: types.Operation{
					To:               TestAddress,
					AdditionalFields: json.RawMessage([]byte(`{"value": "0"}`)),
					Data:             common.Hex2Bytes("0x"),
					OperationMetadata: types.OperationMetadata{
						ContractType: "Sample contract",
						Tags:         []string{"tag1", "tag2"},
					},
				},
			},
		},
	)

	require.Error(t, err)
	require.EqualError(t, err, "Key: 'MCMSProposal.BaseProposal.ChainMetadata' Error:Field validation for 'ChainMetadata' failed on the 'min' tag")
	assert.Nil(t, proposal)
}

func TestMCMSOnlyProposal_Validate_NoTransactions(t *testing.T) {
	t.Parallel()

	proposal, err := NewProposal(
		"1.0",
		2004259681,
		[]types.Signature{},
		false,
		map[types.ChainSelector]types.ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		"Sample description",
		[]types.ChainOperation{},
	)

	require.Error(t, err)
	require.EqualError(t, err, "Key: 'MCMSProposal.Transactions' Error:Field validation for 'Transactions' failed on the 'min' tag")
	assert.Nil(t, proposal)
}

func TestMCMSOnlyProposal_Validate_MissingChainMetadataForTransaction(t *testing.T) {
	t.Parallel()

	proposal, err := NewProposal(
		"1.0",
		2004259681,
		[]types.Signature{},
		false,
		map[types.ChainSelector]types.ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		"Sample description",
		[]types.ChainOperation{
			{
				ChainSelector: 3,
				Operation: types.Operation{
					To:               TestAddress,
					AdditionalFields: json.RawMessage([]byte(`{"value": "0"}`)),
					Data:             common.Hex2Bytes("0x"),
					OperationMetadata: types.OperationMetadata{
						ContractType: "Sample contract",
						Tags:         []string{"tag1", "tag2"},
					},
				},
			},
		},
	)

	require.Error(t, err)
	require.EqualError(t, err, "missing metadata for chain 3")
	assert.Nil(t, proposal)
}

func TestProposalFromFile(t *testing.T) {
	t.Parallel()

	mcmsProposal := MCMSProposal{
		BaseProposal: BaseProposal{
			Version:              "1",
			ValidUntil:           2004259681,
			Signatures:           []types.Signature{},
			OverridePreviousRoot: false,
			Description:          "Test Proposal",
			ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
				TestChain1: {
					StartingOpCount: 0,
					MCMAddress:      TestAddress,
				},
			},
		},
		Transactions: []types.ChainOperation{
			{
				ChainSelector: TestChain1,
				Operation: types.Operation{
					To:               TestAddress,
					AdditionalFields: json.RawMessage([]byte(`{}`)),
					Data:             common.Hex2Bytes("0x"),
				},
			},
		},
	}

	tempFile, err := os.CreateTemp("", "mcms.json")
	require.NoError(t, err)

	proposalBytes, err := json.Marshal(mcmsProposal)
	require.NoError(t, err)
	err = os.WriteFile(tempFile.Name(), proposalBytes, 0600)
	require.NoError(t, err)

	fileProposal, err := NewProposalFromReader(tempFile)
	require.NoError(t, err)
	assert.Equal(t, mcmsProposal, *fileProposal)
}

func Test_Proposal_ChainSelectors(t *testing.T) {
	t.Parallel()

	proposal := MCMSProposal{
		BaseProposal: BaseProposal{
			ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
				TestChain1: {},
				TestChain2: {},
				TestChain3: {},
			},
		},
	}

	want := []types.ChainSelector{TestChain1, TestChain3, TestChain2} // Sorted in ascending order
	assert.Equal(t, want, proposal.ChainSelectors())
}
