package mcms

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
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
	additionalFields, err := json.Marshal(struct {
		Value *big.Int `json:"value"`
	}{
		Value: big.NewInt(0),
	})
	require.NoError(t, err)
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
					To:               TestAddress,
					Value:            big.NewInt(0),
					AdditionalFields: additionalFields,
					Data:             common.Hex2Bytes("0x"),
					ContractType:     "Sample contract",
					Tags:             []string{"tag1", "tag2"},
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
	additionalFields, err := json.Marshal(struct {
		Value *big.Int `json:"value"`
	}{
		Value: big.NewInt(0),
	})
	require.NoError(t, err)
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
					ChainIdentifier(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector): {
						StartingOpCount: 0,
						MCMAddress:      common.HexToAddress("0x0000000000000000000000000000000000000000"),
					},
				},
				Description: "Test Proposal",
				Transactions: []ChainOperation{
					{
						Operation: Operation{
							AdditionalFields: additionalFields,
						},
						ChainIdentifier: ChainIdentifier(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
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
		name        string
		jsonData    string
		expectError bool
		expectedErr error
	}{
		{
			name: "valid proposal",
			jsonData: `{
				"version": "v1.0",
				"validUntil": 4128029039,
				"signatures": [],
				"overridePreviousRoot": false,
				"chainMetadata": {
					"16015286601757825753": {
						"startingOpCount": 0,
						"mcmAddress": "0x0000000000000000000000000000000000000000"
					}
				},
				"description": "Test Proposal",
                "transactions": [
                  {
                     "chainIdentifier": 16015286601757825753,
                     "additionalFields": {"value": 0}
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
					"16015286601757825753": {
						"startingOpCount": 0,
						"mcmAddress": "0x0000000000000000000000000000000000000000"
					}
				},
				"description": "Test Proposal",
                "transactions": [
                  {
                     "chainIdentifier": 16015286601757825753,
                     "additionalFields": null
                  }]
			}`,
			expectError: true,
			expectedErr: &errors.InvalidVersionError{},
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
                     "chainIdentifier": 16015286601757825753,
                     "additionalFields": null
                  }]
			}`,
			expectError: true,
			expectedErr: &errors.NoChainMetadataError{},
		},
		{
			name: "invalid proposal - null AdditionalFields",
			jsonData: `{
				"version": "v1.0",
				"validUntil": 4128029039,
				"signatures": [],
				"overridePreviousRoot": false,
				"chainMetadata": {
					"16015286601757825753": {
						"startingOpCount": 0,
						"mcmAddress": "0x0000000000000000000000000000000000000000"
					}
				},
				"description": "Test Proposal",
                "transactions": [
                  {
                     "chainIdentifier": 16015286601757825753,
                     "additionalFields": null
                  }]
			}`,
			expectError: true,
			expectedErr: fmt.Errorf("failed to unmarshal EVM additional fields: unexpected end of JSON input"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var proposal MCMSProposal
			err := json.Unmarshal([]byte(tt.jsonData), &proposal)

			if tt.expectError {
				require.Error(t, err)
				require.EqualError(t, tt.expectedErr, err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
