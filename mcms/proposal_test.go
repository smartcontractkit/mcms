package mcms

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

var TestAddress = "0x1234567890abcdef"
var TestChain1 = types.ChainSelector(3379446385462418246)
var TestChain2 = types.ChainSelector(16015286601757825753)
var TestChain3 = types.ChainSelector(10344971235874465080)

func Test_Proposal_Validate_Success(t *testing.T) {
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

func Test_Proposal_AddSignature(t *testing.T) {
	t.Parallel()

	signature := types.Signature{}

	proposal := MCMSProposal{}

	assert.Empty(t, proposal.Signatures)

	proposal.AddSignature(signature)

	assert.Equal(t, []types.Signature{signature}, proposal.Signatures)
}

func Test_Proposal_GetEncoders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    MCMSProposal
		want    map[types.ChainSelector]sdk.Encoder
		wantErr string
	}{
		{
			name: "success",
			give: MCMSProposal{
				BaseProposal: BaseProposal{
					OverridePreviousRoot: false,
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						TestChain1: {},
						TestChain2: {},
					},
				},
				Transactions: []types.ChainOperation{
					{ChainSelector: TestChain1},
					{ChainSelector: TestChain1},
					{ChainSelector: TestChain2},
				},
			},
			want: map[types.ChainSelector]sdk.Encoder{
				TestChain1: evm.NewEVMEncoder(2, 1337, false),
				TestChain2: evm.NewEVMEncoder(1, 11155111, false),
			},
		},
		{
			name: "failure: could not create encoder",
			give: MCMSProposal{
				BaseProposal: BaseProposal{
					OverridePreviousRoot: false,
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						types.ChainSelector(1): {},
					},
				},
			},
			wantErr: "invalid chain ID: 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encoders, err := tt.give.GetEncoders()

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, encoders)
			}
		})
	}
}

func Test_Proposal_UseSimulatedBackend(t *testing.T) {
	t.Parallel()

	proposal := MCMSProposal{
		BaseProposal: BaseProposal{},
	}

	assert.False(t, proposal.useSimulatedBackend)

	proposal.UseSimulatedBackend(true)
	assert.True(t, proposal.useSimulatedBackend)
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

func Test_Proposal_CalculateTransactionCounts(t *testing.T) {
	t.Parallel()

	transactions := []types.ChainOperation{
		{ChainSelector: TestChain1},
		{ChainSelector: TestChain1},
		{ChainSelector: TestChain2},
	}

	proposal := MCMSProposal{
		Transactions: transactions,
	}

	got := proposal.TransactionCounts()

	assert.Equal(t, map[types.ChainSelector]uint64{
		TestChain1: 2,
		TestChain2: 1,
	}, got)
}
