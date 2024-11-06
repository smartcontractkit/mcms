package mcms

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-playground/validator/v10"
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

func Test_Proposal_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		give     MCMSProposal
		wantErrs []string
	}{
		{
			name: "valid",
			give: MCMSProposal{
				BaseProposal: BaseProposal{
					Version:    "1",
					ValidUntil: 2004259681,
					Signatures: []types.Signature{},
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						TestChain1: {
							StartingOpCount: 1,
							MCMAddress:      TestAddress,
						},
					},
				},
				Transactions: []types.ChainOperation{
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
			},
			wantErrs: []string{},
		},
		{
			name: "required fields validation",
			give: MCMSProposal{
				BaseProposal: BaseProposal{},
			},
			wantErrs: []string{
				"Key: 'MCMSProposal.BaseProposal.Version' Error:Field validation for 'Version' failed on the 'required' tag",
				"Key: 'MCMSProposal.BaseProposal.ValidUntil' Error:Field validation for 'ValidUntil' failed on the 'required' tag",
				"Key: 'MCMSProposal.BaseProposal.ChainMetadata' Error:Field validation for 'ChainMetadata' failed on the 'required' tag",
				"Key: 'MCMSProposal.Transactions' Error:Field validation for 'Transactions' failed on the 'required' tag",
			},
		},
		{
			name: "min validation",
			give: MCMSProposal{
				BaseProposal: BaseProposal{
					Version:       "1",
					ValidUntil:    2004259681,
					Signatures:    []types.Signature{},
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{},
				},
				Transactions: []types.ChainOperation{},
			},
			wantErrs: []string{
				"Key: 'MCMSProposal.BaseProposal.ChainMetadata' Error:Field validation for 'ChainMetadata' failed on the 'min' tag",
				"Key: 'MCMSProposal.Transactions' Error:Field validation for 'Transactions' failed on the 'min' tag",
			},
		},
		{
			name: "invalid chain metadata",
			give: MCMSProposal{
				BaseProposal: BaseProposal{
					Version:    "1",
					ValidUntil: 2004259681,
					Signatures: []types.Signature{},
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						TestChain1: {
							StartingOpCount: 1,
							MCMAddress:      TestAddress,
						},
					},
				},
				Transactions: []types.ChainOperation{
					{ChainSelector: TestChain2},
				},
			},
			wantErrs: []string{
				"missing metadata for chain 16015286601757825753",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.give.Validate()

			if len(tt.wantErrs) > 0 {
				require.Error(t, err)

				var errs validator.ValidationErrors
				if errors.As(err, &errs) {
					assert.Len(t, errs, len(tt.wantErrs))

					got := []string{}
					for _, e := range errs {
						got = append(got, e.Error())
					}

					assert.ElementsMatch(t, tt.wantErrs, got)
				} else {
					assert.ElementsMatch(t, tt.wantErrs, []string{err.Error()})
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_NewProposal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    string
		want    MCMSProposal
		wantErr string
	}{
		{
			name: "success: initializes a proposal from an io.Reader",
			give: `{
				"version": "1",
				"validUntil": 2004259681,
				"chainMetadata": {
					"3379446385462418246": {}
				},
				"transactions": [
					{
						"chainSelector": 3379446385462418246
					}
				]
			}`,
			want: MCMSProposal{
				BaseProposal: BaseProposal{
					Version:    "1",
					ValidUntil: 2004259681,
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						TestChain1: {},
					},
				},
				Transactions: []types.ChainOperation{
					{ChainSelector: TestChain1},
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
				"version": "1",
				"validUntil": 2004259681,
				"chainMetadata": {},
				"transactions": [
					{}
				]
			}`,
			wantErr: "Key: 'MCMSProposal.BaseProposal.ChainMetadata' Error:Field validation for 'ChainMetadata' failed on the 'min' tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			give := strings.NewReader(tt.give)

			fileProposal, err := NewProposal(give)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, *fileProposal)
			}
		})
	}
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

func Test_Proposal_MerkleTree(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		give     MCMSProposal
		wantRoot common.Hash
		wantErr  string
	}{
		{
			name: "success: generates a merkle tree",
			give: MCMSProposal{
				BaseProposal: BaseProposal{
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						TestChain1: {StartingOpCount: 5},
						TestChain2: {StartingOpCount: 10},
					},
				},
				Transactions: []types.ChainOperation{
					{
						ChainSelector: TestChain1,
						Operation: types.Operation{
							To:               TestAddress,
							AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
							Data:             common.Hex2Bytes("0x"),
							OperationMetadata: types.OperationMetadata{
								ContractType: "Sample contract",
								Tags:         []string{"tag1", "tag2"},
							},
						},
					},
					{
						ChainSelector: TestChain2,
						Operation: types.Operation{
							To:               TestAddress,
							AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
							Data:             common.Hex2Bytes("0x"),
							OperationMetadata: types.OperationMetadata{
								ContractType: "Sample contract",
								Tags:         []string{"tag1", "tag2"},
							},
						},
					},
				},
			},
			wantRoot: common.HexToHash("0x4e899fcdb81dc7f0c1d6609366246807fd897c2afb531ca8f907472fd6e1ed02"),
		},
		{
			name: "failure: could not get encoders",
			give: MCMSProposal{
				BaseProposal: BaseProposal{
					OverridePreviousRoot: false,
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						types.ChainSelector(1): {},
					},
				},
			},
			wantErr: "merkle tree generation error: invalid chain ID: 1",
		},
		{
			name: "failure: missing metadata when fetching nonces",
			give: MCMSProposal{
				BaseProposal: BaseProposal{
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						TestChain1: {StartingOpCount: 5},
					},
				},
				Transactions: []types.ChainOperation{
					{ChainSelector: TestChain2},
				},
			},
			wantErr: "merkle tree generation error: missing metadata for chain 16015286601757825753",
		},
		{
			name: "failure: nonce must be in the range of a uint32",
			give: MCMSProposal{
				BaseProposal: BaseProposal{
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						TestChain1: {StartingOpCount: uint64(math.MaxUint32 + 1)},
					},
				},
				Transactions: []types.ChainOperation{
					{ChainSelector: TestChain1},
				},
			},
			wantErr: "merkle tree generation error: value 4294967296 exceeds uint32 range",
		},
		{
			name: "failure: could not hash operation due to invalid additional values",
			give: MCMSProposal{
				BaseProposal: BaseProposal{
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						TestChain1: {StartingOpCount: 5},
					},
				},
				Transactions: []types.ChainOperation{
					{
						ChainSelector: TestChain1,
						Operation: types.Operation{
							AdditionalFields: json.RawMessage([]byte(``)),
						},
					},
				},
			},
			wantErr: "merkle tree generation error: unexpected end of JSON input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.give.MerkleTree()

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantRoot, got.Root)

				// Test the cache works by calling the function a second time
				got, err = tt.give.MerkleTree()
				require.NoError(t, err)
				assert.Equal(t, tt.wantRoot, got.Root)
			}
		})
	}
}

func Test_Proposal_TransactionCounts(t *testing.T) {
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

func Test_Proposal_TransactionNonces(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    MCMSProposal
		want    []uint64
		wantErr string
	}{
		{
			name: "success: returns the nonces for each transaction",
			give: MCMSProposal{
				BaseProposal: BaseProposal{
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						TestChain1: {StartingOpCount: 5},
						TestChain2: {StartingOpCount: 10},
						TestChain3: {StartingOpCount: 15},
					},
				},
				Transactions: []types.ChainOperation{
					{ChainSelector: TestChain1},
					{ChainSelector: TestChain2},
					{ChainSelector: TestChain1},
					{ChainSelector: TestChain2},
					{ChainSelector: TestChain3},
				},
			},
			want: []uint64{5, 10, 6, 11, 15},
		},
		{
			name: "failure: chain metadata not found for transaction",
			give: MCMSProposal{
				BaseProposal: BaseProposal{
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{},
				},
				Transactions: []types.ChainOperation{
					{ChainSelector: TestChain1},
				},
			},
			wantErr: "missing metadata for chain 3379446385462418246",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.give.TransactionNonces()

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)

				// Test the cache works by calling the function a second time
				got, err = tt.give.TransactionNonces()
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
