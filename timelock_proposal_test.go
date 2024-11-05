package mcms

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
)

var validChainMetadata = map[types.ChainSelector]types.ChainMetadata{
	TestChain1: {
		StartingOpCount: 1,
		MCMAddress:      "0x123",
	},
}
var validTimelockAddresses = map[types.ChainSelector]string{
	TestChain1: "0x123",
}
var validOp = types.Operation{
	To:               "0x123",
	AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
	Data:             common.Hex2Bytes("0x"),
	OperationMetadata: types.OperationMetadata{
		ContractType: "Sample contract",
		Tags:         []string{"tag1", "tag2"},
	},
}

var validBatches = []types.BatchChainOperation{
	{
		ChainSelector: TestChain1,
		Batch: []types.Operation{
			validOp,
		},
	},
}

func TestTimelockProposal_Validation(t *testing.T) {
	t.Parallel()

	// Test table with every validation case for NewProposalWithTimeLock
	testCases := []struct {
		name           string
		version        string
		validUntil     uint32
		signatures     []types.Signature
		overridePrev   bool
		chainMetadata  map[types.ChainSelector]types.ChainMetadata
		description    string
		timelockAddr   map[types.ChainSelector]string
		batches        []types.BatchChainOperation
		timelockAction types.TimelockAction
		timelockDelay  string
		expectedError  error
	}{
		{
			name:           "Valid proposal",
			version:        "1.0",
			validUntil:     2004259681,
			signatures:     []types.Signature{},
			overridePrev:   false,
			chainMetadata:  validChainMetadata,
			description:    "description",
			timelockAddr:   validTimelockAddresses,
			batches:        validBatches,
			timelockAction: types.TimelockActionSchedule,
			timelockDelay:  "1h",
			expectedError:  nil,
		},
		{
			name:           "Invalid version",
			version:        "",
			validUntil:     2004259681,
			signatures:     []types.Signature{},
			overridePrev:   false,
			chainMetadata:  validChainMetadata,
			description:    "description",
			timelockAddr:   validTimelockAddresses,
			batches:        validBatches,
			timelockAction: types.TimelockActionSchedule,
			timelockDelay:  "1h",
			expectedError:  errors.New("Key: 'MCMSWithTimelockProposal.BaseProposal.Version' Error:Field validation for 'Version' failed on the 'required' tag"),
		},
		{
			name:           "Invalid validUntil",
			version:        "1.0",
			validUntil:     0,
			signatures:     []types.Signature{},
			overridePrev:   false,
			chainMetadata:  validChainMetadata,
			description:    "description",
			timelockAddr:   validTimelockAddresses,
			batches:        validBatches,
			timelockAction: types.TimelockActionSchedule,
			timelockDelay:  "1h",
			expectedError:  errors.New("Key: 'MCMSWithTimelockProposal.BaseProposal.ValidUntil' Error:Field validation for 'ValidUntil' failed on the 'required' tag"),
		},
		{
			name:           "Invalid chainMetadata",
			version:        "1.0",
			validUntil:     2004259681,
			signatures:     []types.Signature{},
			overridePrev:   false,
			chainMetadata:  nil,
			description:    "description",
			timelockAddr:   validTimelockAddresses,
			batches:        validBatches,
			timelockAction: types.TimelockActionSchedule,
			timelockDelay:  "1h",
			expectedError:  errors.New("Key: 'MCMSWithTimelockProposal.BaseProposal.ChainMetadata' Error:Field validation for 'ChainMetadata' failed on the 'required' tag"),
		},
		{
			name:           "Invalid timelockAddresses",
			version:        "1.0",
			validUntil:     2004259681,
			signatures:     []types.Signature{},
			overridePrev:   false,
			chainMetadata:  validChainMetadata,
			description:    "description",
			timelockAddr:   nil,
			batches:        validBatches,
			timelockAction: types.TimelockActionSchedule,
			timelockDelay:  "1h",
			expectedError:  errors.New("Key: 'MCMSWithTimelockProposal.TimelockAddresses' Error:Field validation for 'TimelockAddresses' failed on the 'required' tag"),
		},
		{
			name:           "Invalid batches",
			version:        "1.0",
			validUntil:     2004259681,
			signatures:     []types.Signature{},
			overridePrev:   false,
			chainMetadata:  validChainMetadata,
			description:    "description",
			timelockAddr:   validTimelockAddresses,
			batches:        nil,
			timelockAction: types.TimelockActionSchedule,
			timelockDelay:  "1h",
			expectedError:  errors.New("Key: 'MCMSWithTimelockProposal.Transactions' Error:Field validation for 'Transactions' failed on the 'required' tag"),
		},
		{
			name:           "Invalid timelockAction",
			version:        "1.0",
			validUntil:     2004259681,
			signatures:     []types.Signature{},
			overridePrev:   false,
			chainMetadata:  validChainMetadata,
			description:    "description",
			timelockAddr:   validTimelockAddresses,
			batches:        validBatches,
			timelockAction: types.TimelockAction("invalid"),
			timelockDelay:  "1h",
			expectedError:  errors.New("Key: 'MCMSWithTimelockProposal.Operation' Error:Field validation for 'Operation' failed on the 'oneof' tag"),
		},
		{
			name:           "Invalid timelockDelay",
			version:        "1.0",
			validUntil:     2004259681,
			signatures:     []types.Signature{},
			overridePrev:   false,
			chainMetadata:  validChainMetadata,
			description:    "description",
			timelockAddr:   validTimelockAddresses,
			batches:        validBatches,
			timelockAction: types.TimelockActionSchedule,
			timelockDelay:  "1nm",
			expectedError:  errors.New("invalid delay: 1nm"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewProposalWithTimeLock(
				tc.version,
				tc.validUntil,
				tc.signatures,
				tc.overridePrev,
				tc.chainMetadata,
				tc.description,
				tc.timelockAddr,
				tc.batches,
				tc.timelockAction,
				tc.timelockDelay,
			)
			if err != nil && tc.expectedError == nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if err != nil && tc.expectedError != nil {
				assert.Equal(t, tc.expectedError.Error(), err.Error())
			}
		})
	}
}

func TestTimelockProposal_ToMCMSProposal(t *testing.T) {
	t.Parallel()

	proposal, err := NewProposalWithTimeLock(
		"1.0",
		2004259681,
		[]types.Signature{},
		false,
		validChainMetadata,
		"description",
		validTimelockAddresses,
		validBatches,
		types.TimelockActionSchedule,
		"1h",
	)
	require.NoError(t, err)

	mcmsProposal, err := proposal.toMCMSOnlyProposal()
	require.NoError(t, err)

	assert.Equal(t, "1.0", mcmsProposal.Version)
	assert.Equal(t, uint32(2004259681), mcmsProposal.ValidUntil)
	assert.Equal(t, []types.Signature{}, mcmsProposal.Signatures)
	assert.False(t, mcmsProposal.OverridePreviousRoot)
	assert.Equal(t, validChainMetadata, mcmsProposal.ChainMetadata)
	assert.Equal(t, "description", mcmsProposal.Description)
	assert.Len(t, mcmsProposal.Transactions, 1)
}

const validJsonProposal = `{
  "chainMetadata": {
    "16015286601757825753": {
      "mcmAddress": "0x0000000000000000000000000000000000000000",
      "startingOpCount": 0
    }
  },
  "description": "Test proposal",
  "delay": "1h",
  "operation": "schedule",
  "overridePreviousRoot": true,
  "signatures": null,
  "timelockAddresses": {},
  "transactions": [
    {
      "batch": [
        {
          "additionalFields": {
            "value": 0
          },
          "contractType": "",
          "data": "ZGF0YQ==",
          "tags": null,
          "to": "0x0000000000000000000000000000000000000000"
        }
      ],
      "chainIdentifier": 16015286601757825753
    }
  ],
  "validUntil": 4128029039,
  "version": "MCMSWithTimelock"
}`

func TestMCMSWithTimelockProposal_MarshalJSON(t *testing.T) {
	t.Parallel()

	additionalFields, err := json.Marshal(struct {
		Value *big.Int `json:"value"`
	}{
		Value: big.NewInt(0),
	})
	require.NoError(t, err)
	tests := []struct {
		name       string
		proposal   MCMSWithTimelockProposal
		wantErr    bool
		expectJSON string
	}{
		{
			name: "successful marshalling",
			proposal: MCMSWithTimelockProposal{
				BaseProposal: BaseProposal{
					Version:     "MCMSWithTimelock",
					ValidUntil:  4128029039,
					Description: "Test proposal",
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						types.ChainSelector(cselectors.ETHEREUM_TESTNET_SEPOLIA.Selector): {
							StartingOpCount: 0,
							MCMAddress:      "0x0000000000000000000000000000000000000000",
						},
					},
					OverridePreviousRoot: true,
				},
				Operation:         types.TimelockActionSchedule,
				Delay:             "1h",
				TimelockAddresses: map[types.ChainSelector]string{},
				Transactions: []types.BatchChainOperation{
					{
						ChainSelector: types.ChainSelector(cselectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
						Batch: []types.Operation{
							{
								To:               "0x0000000000000000000000000000000000000000",
								AdditionalFields: additionalFields,
								Data:             []byte("data"),
							},
						},
					},
				},
			},
			wantErr:    false,
			expectJSON: validJsonProposal,
		},
		{
			name: "error during marshalling transactions",
			proposal: MCMSWithTimelockProposal{
				Transactions: []types.BatchChainOperation{
					{
						ChainSelector: types.ChainSelector(1),
						Batch:         nil, // This will cause an error because Batch should not be nil
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := json.Marshal(&tt.proposal)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.JSONEq(t, tt.expectJSON, string(got))
			}
		})
	}
}

func TestMCMSWithTimelockProposal_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	// Prepare the compact version of the validJsonProposal
	var compactBuffer bytes.Buffer
	err := json.Compact(&compactBuffer, []byte(validJsonProposal))
	require.NoError(t, err)

	// Use the compact JSON as the one-liner version
	compactJsonProposal := compactBuffer.String()

	// Preparing the additional fields
	additionalFields, err := json.Marshal(struct {
		Value *big.Int `json:"value"`
	}{
		Value: big.NewInt(0),
	})
	require.NoError(t, err)

	tests := []struct {
		name        string
		jsonData    string
		expectedErr error
		expected    MCMSWithTimelockProposal
	}{
		{
			name:        "successful unmarshalling",
			jsonData:    compactJsonProposal,
			expectedErr: nil,
			expected: MCMSWithTimelockProposal{
				BaseProposal: BaseProposal{
					Version:     "MCMSWithTimelock",
					ValidUntil:  4128029039,
					Description: "Test proposal",
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						types.ChainSelector(16015286601757825753): {
							StartingOpCount: 0,
							MCMAddress:      "0x0000000000000000000000000000000000000000",
						},
					},
					OverridePreviousRoot: true,
				},
				Operation:         types.TimelockActionSchedule,
				Delay:             "1h",
				TimelockAddresses: map[types.ChainSelector]string{},
				Transactions: []types.BatchChainOperation{
					{
						ChainSelector: types.ChainSelector(16015286601757825753),
						Batch: []types.Operation{
							{
								To:               "0x0000000000000000000000000000000000000000",
								AdditionalFields: additionalFields,
								Data:             []byte("data"),
							},
						},
					},
				},
			},
		},
		{
			name: "error during unmarshalling invalid JSON",
			jsonData: `{
  "chainMetadata": {
    "16015286601757825753": {
      "mcmAddress": "0x0000000000000000000000000000000000000000",
      "startingOpCount": 0
    }
  },
  "description": "Test proposal",
  "delay": "1h",
  "operation": "INVALID",
  "overridePreviousRoot": true,
  "signatures": null,
  "timelockAddresses": {},
  "transactions": [
    {
      "batch": [
        {
          "additionalFields": {
            "value": 0
          },
          "contractType": "",
          "data": "ZGF0YQ==",
          "tags": null,
          "to": "0x0000000000000000000000000000000000000000"
        }
      ],
      "chainIdentifier": 16015286601757825753
    }
  ],
  "validUntil": 4128029039,
  "version": "MCMSWithTimelock"
}`, // invalid operation field
			expectedErr: errors.New("Key: 'MCMSWithTimelockProposal.Operation' Error:Field validation for 'Operation' failed on the 'oneof' tag"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got MCMSWithTimelockProposal
			err := json.Unmarshal([]byte(tt.jsonData), &got)
			if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}
