package mcms

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-playground/validator/v10"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

var (
	TestAddress = "0x1234567890abcdef"
	TestChain1  = types.ChainSelector(cselectors.GETH_TESTNET.Selector)                    // 3379446385462418246
	TestChain2  = types.ChainSelector(cselectors.ETHEREUM_TESTNET_SEPOLIA.Selector)        // 16015286601757825753
	TestChain3  = types.ChainSelector(cselectors.ETHEREUM_TESTNET_SEPOLIA_BASE_1.Selector) // 10344971235874465080
)

func Test_BaseProposal_AppendSignature(t *testing.T) {
	t.Parallel()

	signature := types.Signature{}

	proposal := BaseProposal{}

	assert.Empty(t, proposal.Signatures)

	proposal.AppendSignature(signature)

	assert.Equal(t, []types.Signature{signature}, proposal.Signatures)
}

func Test_NewProposal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    string
		want    Proposal
		wantErr string
	}{
		{
			name: "success: initializes a proposal from an io.Reader",
			give: `{
				"version": "v1",
				"kind": "Proposal",
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
			want: Proposal{
				BaseProposal: BaseProposal{
					Version:    "v1",
					Kind:       types.KindProposal,
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
				"version": "v1",
				"kind": "Proposal",
				"validUntil": 2004259681,
				"chainMetadata": {},
				"transactions": [
					{}
				]
			}`,
			wantErr: "Key: 'Proposal.BaseProposal.ChainMetadata' Error:Field validation for 'ChainMetadata' failed on the 'min' tag",
		},
		{
			name: "failure: invalid proposal kind",
			give: `{
				"version": "v1",
				"kind": "TimelockProposal",
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
			wantErr: "invalid proposal kind: TimelockProposal, value accepted is Proposal",
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

func Test_WriteProposal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		give       *Proposal
		giveWriter func() io.Writer // Use this to overwrite the default writer
		want       string
		wantErr    string
	}{
		{
			name: "success: writes a proposal to an io.Writer",
			give: &Proposal{
				BaseProposal: BaseProposal{
					Version:    "1",
					Kind:       types.KindProposal,
					ValidUntil: 2004259681,
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						TestChain1: {},
					},
				},
				Transactions: []types.ChainOperation{
					{ChainSelector: TestChain1},
				},
			},
			want: `{
				"version": "1",
				"kind": "Proposal",
				"description": "",
				"validUntil": 2004259681,
				"overridePreviousRoot": false,
				"signatures": null,
				"chainMetadata": {
					"3379446385462418246": {
						"mcmAddress": "",
						"startingOpCount": 0
					}
				},
				"transactions": [
					{
						"chainSelector": 3379446385462418246,
						"to": "",
						"additionalFields": null,
						"data": null,
						"tags": null,
						"contractType": ""
					}
				]
			}`,
		},
		{
			name: "success: writes a proposal to an io.Writer",
			giveWriter: func() io.Writer {
				return newFakeWriter(0, errors.New("write error"))
			},
			give:    &Proposal{},
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

			err := WriteProposal(w, tt.give)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.JSONEq(t, tt.want, b.String())
			}
		})
	}
}

func Test_Proposal_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		give     Proposal
		wantErrs []string
	}{
		{
			name: "valid",
			give: Proposal{
				BaseProposal: BaseProposal{
					Version:    "v1",
					Kind:       types.KindProposal,
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
			wantErrs: []string{},
		},
		{
			name: "required fields validation",
			give: Proposal{
				BaseProposal: BaseProposal{},
			},
			wantErrs: []string{
				"Key: 'Proposal.BaseProposal.Version' Error:Field validation for 'Version' failed on the 'required' tag",
				"Key: 'Proposal.BaseProposal.ValidUntil' Error:Field validation for 'ValidUntil' failed on the 'required' tag",
				"Key: 'Proposal.BaseProposal.ChainMetadata' Error:Field validation for 'ChainMetadata' failed on the 'required' tag",
				"Key: 'Proposal.Transactions' Error:Field validation for 'Transactions' failed on the 'required' tag",
				"Key: 'Proposal.BaseProposal.Kind' Error:Field validation for 'Kind' failed on the 'required' tag",
			},
		},
		{
			name: "min validation",
			give: Proposal{
				BaseProposal: BaseProposal{
					Version:       "v1",
					ValidUntil:    2004259681,
					Signatures:    []types.Signature{},
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{},
				},
				Transactions: []types.ChainOperation{},
			},
			wantErrs: []string{
				"Key: 'Proposal.BaseProposal.ChainMetadata' Error:Field validation for 'ChainMetadata' failed on the 'min' tag",
				"Key: 'Proposal.Transactions' Error:Field validation for 'Transactions' failed on the 'min' tag",
				"Key: 'Proposal.BaseProposal.Kind' Error:Field validation for 'Kind' failed on the 'required' tag",
			},
		},
		{
			name: "invalid chain metadata",
			give: Proposal{
				BaseProposal: BaseProposal{
					Version:    "v1",
					Kind:       types.KindProposal,
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

func Test_Proposal_GetEncoders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    Proposal
		want    map[types.ChainSelector]sdk.Encoder
		wantErr string
	}{
		{
			name: "success",
			give: Proposal{
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
			give: Proposal{
				BaseProposal: BaseProposal{
					OverridePreviousRoot: false,
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						types.ChainSelector(1): {},
					},
				},
			},
			wantErr: "unable to create encoder: invalid chain ID: 1",
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

	proposal := Proposal{
		BaseProposal: BaseProposal{},
	}

	assert.False(t, proposal.useSimulatedBackend)

	proposal.UseSimulatedBackend(true)
	assert.True(t, proposal.useSimulatedBackend)
}

func Test_Proposal_ChainSelectors(t *testing.T) {
	t.Parallel()

	proposal := Proposal{
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
		give     Proposal
		wantRoot common.Hash
		wantErr  string
	}{
		{
			name: "success: generates a merkle tree",
			give: Proposal{
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
			give: Proposal{
				BaseProposal: BaseProposal{
					OverridePreviousRoot: false,
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						types.ChainSelector(1): {},
					},
				},
			},
			wantErr: "merkle tree generation error: unable to create encoder: invalid chain ID: 1",
		},
		{
			name: "failure: missing metadata when fetching nonces",
			give: Proposal{
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
			give: Proposal{
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
			give: Proposal{
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

	proposal := Proposal{
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
		give    Proposal
		want    []uint64
		wantErr string
	}{
		{
			name: "success: returns the nonces for each transaction",
			give: Proposal{
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
			give: Proposal{
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
			}
		})
	}
}
