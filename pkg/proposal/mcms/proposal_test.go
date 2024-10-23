package mcms

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/mcms/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
)

var TestAddress = common.HexToAddress("0x1234567890abcdef")
var TestChain1 = ChainIdentifier(3379446385462418246)
var TestChain2 = ChainIdentifier(16015286601757825753)
var TestChain3 = ChainIdentifier(10344971235874465080)

func TestMCMSOnlyProposal_Validate_Success(t *testing.T) {
	t.Parallel()

	proposal, err := NewProposal(
		"1.0",
		2004259681,
		[]Signature{},
		false,
		map[ChainIdentifier]ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		"Sample description",
		[]ChainOperation{
			{
				ChainIdentifier: TestChain1,
				Operation: Operation{
					To:           TestAddress,
					Value:        big.NewInt(0),
					Data:         common.Hex2Bytes("0x"),
					ContractType: "Sample contract",
					Tags:         []string{"tag1", "tag2"},
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
		map[ChainIdentifier]ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		"Sample description",
		[]ChainOperation{
			{
				ChainIdentifier: TestChain1,
				Operation: Operation{
					To:           TestAddress,
					Value:        big.NewInt(0),
					Data:         common.Hex2Bytes("0x"),
					ContractType: "Sample contract",
					Tags:         []string{"tag1", "tag2"},
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
		map[ChainIdentifier]ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		"Sample description",
		[]ChainOperation{
			{
				ChainIdentifier: TestChain1,
				Operation: Operation{
					To:           TestAddress,
					Value:        big.NewInt(0),
					Data:         common.Hex2Bytes("0x"),
					ContractType: "Sample contract",
					Tags:         []string{"tag1", "tag2"},
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
		map[ChainIdentifier]ChainMetadata{},
		"Sample description",
		[]ChainOperation{
			{
				ChainIdentifier: TestChain1,
				Operation: Operation{
					To:           TestAddress,
					Value:        big.NewInt(0),
					Data:         common.Hex2Bytes("0x"),
					ContractType: "Sample contract",
					Tags:         []string{"tag1", "tag2"},
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
		map[ChainIdentifier]ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		"",
		[]ChainOperation{
			{
				ChainIdentifier: TestChain1,
				Operation: Operation{
					To:           TestAddress,
					Value:        big.NewInt(0),
					Data:         common.Hex2Bytes("0x"),
					ContractType: "Sample contract",
					Tags:         []string{"tag1", "tag2"},
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
		map[ChainIdentifier]ChainMetadata{
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
		map[ChainIdentifier]ChainMetadata{
			TestChain1: {
				StartingOpCount: 1,
				MCMAddress:      TestAddress,
			},
		},
		"Sample description",
		[]ChainOperation{
			{
				ChainIdentifier: 3,
				Operation: Operation{
					To:           TestAddress,
					Value:        big.NewInt(0),
					Data:         common.Hex2Bytes("0x"),
					ContractType: "Sample contract",
					Tags:         []string{"tag1", "tag2"},
				},
			},
		},
	)

	require.Error(t, err)
	require.EqualError(t, err, "missing chain metadata for chain 3")
	assert.Nil(t, proposal)
}

func TestMCMSProposal_MarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		proposal    MCMSProposal
		expectError bool
	}{
		{
			name: "valid proposal",
			proposal: MCMSProposal{
				Version:              "v1.0",
				ValidUntil:           uint32(4128029039),
				Signatures:           []Signature{},
				OverridePreviousRoot: false,
				ChainMetadata: map[ChainIdentifier]ChainMetadata{
					ChainIdentifier(1): {
						StartingOpCount: 0,
						MCMAddress:      common.HexToAddress("0x0000000000000000000000000000000000000000"),
					},
				},
				Description: "Test Proposal",
				Transactions: []ChainOperation{
					{
						Operation:       Operation{},
						ChainIdentifier: ChainIdentifier(1),
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid proposal - missing version",
			proposal: MCMSProposal{
				Version:              "",
				ValidUntil:           uint32(4128029039),
				Signatures:           []Signature{},
				OverridePreviousRoot: false,
				ChainMetadata: map[ChainIdentifier]ChainMetadata{
					ChainIdentifier(1): {
						StartingOpCount: 0,
						MCMAddress:      common.HexToAddress("0x0000000000000000000000000000000000000000"),
					},
				},
				Description:  "Test Proposal",
				Transactions: []ChainOperation{},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := json.Marshal(tt.proposal)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMCMSProposal_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name            string
		jsonData        string
		expectError     bool
		expectedErrType error
	}{
		{
			name: "valid proposal",
			jsonData: `{
				"version": "v1.0",
				"validUntil": 4128029039,
				"signatures": [],
				"overridePreviousRoot": false,
				"chainMetadata": {
					"1": {
						"startingOpCount": 0,
						"mcmAddress": "0x0000000000000000000000000000000000000000"
					}
				},
				"description": "Test Proposal",
                "transactions": [
                  {
                     "chainIdentifier": 1,
                     "additionalFields": null
                  }]
			}`,
			expectError: false,
		},
		{
			name: "invalid proposal - missing version",
			jsonData: `{
				"version": "",
				"validUntil": 4128029039,
				"signatures": [],
				"overridePreviousRoot": false,
				"chainMetadata": {
					"1": {
						"startingOpCount": 0,
						"mcmAddress": "0x0000000000000000000000000000000000000000"
					}
				},
				"description": "Test Proposal",
                "transactions": [
                  {
                     "chainIdentifier": 1,
                     "additionalFields": null
                  }]
			}`,
			expectError:     true,
			expectedErrType: &errors.InvalidVersionError{},
		},
		{
			name: "invalid proposal - missing chain metadata",
			jsonData: `{
				"version": "v1.0",
				"validUntil": 4128029039,
				"signatures": [],
				"overridePreviousRoot": false,
				"chainMetadata": {},
				"description": "Test Proposal",
                "transactions": [
                  {
                     "chainIdentifier": 1,
                     "additionalFields": null
                  }]
			}`,
			expectError:     true,
			expectedErrType: &errors.NoChainMetadataError{},
		},
		{
			name: "invalid proposal - null AdditionalFields",
			jsonData: `{
				"version": "v1.0",
				"validUntil": 4128029039,
				"signatures": [],
				"overridePreviousRoot": false,
				"chainMetadata": {
					"1": {
						"startingOpCount": 0,
						"mcmAddress": "0x0000000000000000000000000000000000000000"
					}
				},
				"description": "Test Proposal",
                "transactions": [
                  {
                     "chainIdentifier": 1,
                     "additionalFields": null
                  }]
			}`,
			expectError: false, // This should not throw an error, but set AdditionalFields to nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var proposal MCMSProposal
			err := json.Unmarshal([]byte(tt.jsonData), &proposal)

			if tt.expectError {
				require.Error(t, err)
				require.IsType(t, tt.expectedErrType, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
