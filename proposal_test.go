package mcms

import (
	"encoding/json"
	"errors"
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
