package mcms

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
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

func Test_NewTimelockProposal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    string
		want    TimelockProposal
		wantErr string
	}{
		{
			name: "success: initializes a proposal from an io.Reader",
			give: `{
				"version": "v1",
				"kind": "TimelockProposal",
				"validUntil": 2004259681,
				"chainMetadata": {
					"16015286601757825753": {
						"mcmAddress": "0x0000000000000000000000000000000000000000",
						"startingOpCount": 0
					}
				},
				"description": "Test proposal",
				"overridePreviousRoot": false,
				"operation": "schedule",
				"delay": "1h",
				"timelockAddresses": {
					"16015286601757825753": "0x01"
				},
				"transactions": [
					{
						"chainSelector": 16015286601757825753,
						"batch": [
							{
								"to": "0x0000000000000000000000000000000000000000",
								"additionalFields": {"value": 0},
								"data": "ZGF0YQ=="
							}
						]
					}
				]
			}`,
			want: TimelockProposal{
				BaseProposal: BaseProposal{
					Version:     "v1",
					Kind:        string(types.KindTimelockProposal),
					ValidUntil:  2004259681,
					Description: "Test proposal",
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						TestChain2: {
							StartingOpCount: 0,
							MCMAddress:      "0x0000000000000000000000000000000000000000",
						},
					},
					OverridePreviousRoot: false,
				},
				Operation: types.TimelockActionSchedule,
				Delay:     "1h",
				TimelockAddresses: map[types.ChainSelector]string{
					TestChain2: "0x01",
				},
				Transactions: []types.BatchChainOperation{
					{
						ChainSelector: TestChain2,
						Batch: []types.Operation{
							{
								To:               "0x0000000000000000000000000000000000000000",
								AdditionalFields: []byte(`{"value": 0}`),
								Data:             []byte("data"),
							},
						},
					},
				},
			},
		},
		{
			name:    "failure: could not unmarshal JSON",
			give:    `invalid`,
			wantErr: "invalid character 'i' looking for beginning of value",
		},
		{
			name: "failure: invalid proposal",
			give: `{
				"version": "v1",
				"kind": "TimelockProposal",
				"validUntil": 2004259681,
				"chainMetadata": {},
				"description": "Test proposal",
				"overridePreviousRoot": false,
				"operation": "schedule",
				"delay": "1h",
				"timelockAddresses": {
					"16015286601757825753": "0x01"
				},
				"transactions": [
					{
						"chainSelector": 16015286601757825753,
						"batch": [
							{
								"to": "0x0000000000000000000000000000000000000000",
								"additionalFields": {"value": 0},
								"data": "ZGF0YQ=="
							}
						]
					}
				]
			}`,
			wantErr: "Key: 'TimelockProposal.BaseProposal.ChainMetadata' Error:Field validation for 'ChainMetadata' failed on the 'min' tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			give := strings.NewReader(tt.give)

			got, err := NewTimelockProposal(give)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, *got)
			}
		})
	}
}

func Test_WriteTimelockProposal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		give       *TimelockProposal
		giveWriter func() io.Writer // Use this to overwrite the default writer
		want       string
		wantErr    string
	}{
		{
			name: "success: writes a proposal to an io.Writer",
			give: &TimelockProposal{
				BaseProposal: BaseProposal{
					Version:     "v1",
					Kind:        string(types.KindTimelockProposal),
					ValidUntil:  2004259681,
					Description: "Test proposal",
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						TestChain2: {
							StartingOpCount: 0,
							MCMAddress:      "0x0000000000000000000000000000000000000000",
						},
					},
					OverridePreviousRoot: false,
				},
				Operation:         types.TimelockActionSchedule,
				Delay:             "1h",
				TimelockAddresses: map[types.ChainSelector]string{},
				Transactions: []types.BatchChainOperation{
					{
						ChainSelector: TestChain2,
						Batch: []types.Operation{
							{
								To:               "0x0000000000000000000000000000000000000000",
								AdditionalFields: []byte(`{"value": 0}`),
								Data:             []byte("data"),
							},
						},
					},
				},
			},
			want: `{
				"version": "v1",
				"kind": "TimelockProposal",
				"validUntil": 2004259681,
				"chainMetadata": {
					"16015286601757825753": {
						"mcmAddress": "0x0000000000000000000000000000000000000000",
						"startingOpCount": 0
					}
				},
				"description": "Test proposal",
				"overridePreviousRoot": false,
				"operation": "schedule",
				"delay": "1h",
				"signatures": null,
				"timelockAddresses": {},
				"transactions": [
					{
						"chainSelector": 16015286601757825753,
						"batch": [
							{
								"to": "0x0000000000000000000000000000000000000000",
								"additionalFields": {"value": 0},
								"data": "ZGF0YQ==",
								"contractType": "",
								"tags": null
							}
						]
					}
				]
			}`,
		},
		{
			name: "success: writes a proposal to an io.Writer",
			giveWriter: func() io.Writer {
				return newFakeWriter(0, errors.New("write error"))
			},
			give:    &TimelockProposal{},
			wantErr: "write error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var w io.Writer
			b := new(strings.Builder)

			if tt.giveWriter != nil {
				w = tt.giveWriter()
			} else {
				w = b
			}

			err := WriteTimelockProposal(w, tt.give)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.JSONEq(t, tt.want, b.String())
			}
		})
	}
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
			version:        "v1",
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
			expectedError:  errors.New("Key: 'TimelockProposal.BaseProposal.Version' Error:Field validation for 'Version' failed on the 'required' tag"),
		},
		{
			name:           "Invalid validUntil",
			version:        "v1",
			validUntil:     0,
			signatures:     []types.Signature{},
			overridePrev:   false,
			chainMetadata:  validChainMetadata,
			description:    "description",
			timelockAddr:   validTimelockAddresses,
			batches:        validBatches,
			timelockAction: types.TimelockActionSchedule,
			timelockDelay:  "1h",
			expectedError:  errors.New("Key: 'TimelockProposal.BaseProposal.ValidUntil' Error:Field validation for 'ValidUntil' failed on the 'required' tag"),
		},
		{
			name:           "Invalid chainMetadata",
			version:        "v1",
			validUntil:     2004259681,
			signatures:     []types.Signature{},
			overridePrev:   false,
			chainMetadata:  nil,
			description:    "description",
			timelockAddr:   validTimelockAddresses,
			batches:        validBatches,
			timelockAction: types.TimelockActionSchedule,
			timelockDelay:  "1h",
			expectedError:  errors.New("Key: 'TimelockProposal.BaseProposal.ChainMetadata' Error:Field validation for 'ChainMetadata' failed on the 'required' tag"),
		},
		{
			name:           "Invalid timelockAddresses",
			version:        "v1",
			validUntil:     2004259681,
			signatures:     []types.Signature{},
			overridePrev:   false,
			chainMetadata:  validChainMetadata,
			description:    "description",
			timelockAddr:   nil,
			batches:        validBatches,
			timelockAction: types.TimelockActionSchedule,
			timelockDelay:  "1h",
			expectedError:  errors.New("Key: 'TimelockProposal.TimelockAddresses' Error:Field validation for 'TimelockAddresses' failed on the 'required' tag"),
		},
		{
			name:           "Invalid batches",
			version:        "v1",
			validUntil:     2004259681,
			signatures:     []types.Signature{},
			overridePrev:   false,
			chainMetadata:  validChainMetadata,
			description:    "description",
			timelockAddr:   validTimelockAddresses,
			batches:        nil,
			timelockAction: types.TimelockActionSchedule,
			timelockDelay:  "1h",
			expectedError:  errors.New("Key: 'TimelockProposal.Transactions' Error:Field validation for 'Transactions' failed on the 'required' tag"),
		},
		{
			name:           "Invalid timelockAction",
			version:        "v1",
			validUntil:     2004259681,
			signatures:     []types.Signature{},
			overridePrev:   false,
			chainMetadata:  validChainMetadata,
			description:    "description",
			timelockAddr:   validTimelockAddresses,
			batches:        validBatches,
			timelockAction: types.TimelockAction("invalid"),
			timelockDelay:  "1h",
			expectedError:  errors.New("Key: 'TimelockProposal.Operation' Error:Field validation for 'Operation' failed on the 'oneof' tag"),
		},
		{
			name:           "Invalid timelockDelay",
			version:        "v1",
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

func TestTimelockProposal_Convert(t *testing.T) {
	t.Parallel()

	proposal, err := NewProposalWithTimeLock(
		"v1",
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

	mcmsProposal, err := proposal.Convert()
	require.NoError(t, err)

	assert.Equal(t, "v1", mcmsProposal.Version)
	assert.Equal(t, uint32(2004259681), mcmsProposal.ValidUntil)
	assert.Equal(t, []types.Signature{}, mcmsProposal.Signatures)
	assert.False(t, mcmsProposal.OverridePreviousRoot)
	assert.Equal(t, validChainMetadata, mcmsProposal.ChainMetadata)
	assert.Equal(t, "description", mcmsProposal.Description)
	assert.Len(t, mcmsProposal.Transactions, 1)
}
