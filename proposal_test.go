package mcms

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

var (
	TestAddress = "0x1234567890abcdef"
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
				"operations": [
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
						chaintest.Chain1Selector: {},
					},
				},
				Operations: []types.Operation{
					{ChainSelector: chaintest.Chain1Selector},
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
				"operations": [
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
				"operations": [
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
		give       func() *Proposal
		giveWriter func() io.Writer // Use this to overwrite the default writer
		want       string
		wantErr    string
	}{
		{
			name: "success: writes a proposal to an io.Writer",
			give: func() *Proposal {

				builder := NewProposalBuilder()
				builder.SetVersion("v1")
				builder.SetValidUntil(2004259681)
				builder.AddChainMetadata(chaintest.Chain1Selector, types.ChainMetadata{})
				builder.AddOperation(types.Operation{ChainSelector: chaintest.Chain1Selector})
				proposal, err := builder.Build()
				require.NoError(t, err)
				return proposal
			},
			want: `{
				"version": "v1",
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
				"operations": [
					{
						"chainSelector": 3379446385462418246,
						"transaction": {
							"to": "",
							"additionalFields": null,
							"data": null,
							"tags": null,
							"contractType": ""
						}
					}
				]
			}`,
		},
		{
			name: "failure: writer returns error",
			giveWriter: func() io.Writer {
				return newFakeWriter(0, errors.New("write error"))
			},
			give: func() *Proposal {
				return &Proposal{}
			},
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
			give := tt.give()
			err := WriteProposal(w, give)

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
						chaintest.Chain1Selector: {
							StartingOpCount: 1,
							MCMAddress:      TestAddress,
						},
					},
				},
				Operations: []types.Operation{
					{
						ChainSelector: chaintest.Chain1Selector,
						Transaction: types.Transaction{
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
				"Key: 'Proposal.Operations' Error:Field validation for 'Operations' failed on the 'required' tag",
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
				Operations: []types.Operation{},
			},
			wantErrs: []string{
				"Key: 'Proposal.BaseProposal.ChainMetadata' Error:Field validation for 'ChainMetadata' failed on the 'min' tag",
				"Key: 'Proposal.Operations' Error:Field validation for 'Operations' failed on the 'min' tag",
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
						chaintest.Chain1Selector: {
							StartingOpCount: 1,
							MCMAddress:      TestAddress,
						},
					},
				},
				Operations: []types.Operation{
					{ChainSelector: chaintest.Chain2Selector},
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
		give    func() Proposal
		want    map[types.ChainSelector]sdk.Encoder
		wantErr string
	}{
		{
			name: "success",
			give: func() Proposal {
				builder := NewProposalBuilder()
				builder.SetVersion("v1")
				builder.SetValidUntil(2552083725)
				builder.AddChainMetadata(chaintest.Chain1Selector, types.ChainMetadata{})
				builder.AddChainMetadata(chaintest.Chain2Selector, types.ChainMetadata{})
				builder.AddOperation(types.Operation{ChainSelector: chaintest.Chain1Selector})
				builder.AddOperation(types.Operation{ChainSelector: chaintest.Chain1Selector})
				builder.AddOperation(types.Operation{ChainSelector: chaintest.Chain2Selector})
				proposal, err := builder.Build()
				require.NoError(t, err)
				return *proposal

			},
			want: map[types.ChainSelector]sdk.Encoder{
				chaintest.Chain1Selector: evm.NewEVMEncoder(chaintest.Chain1Selector, 2, false, false),
				chaintest.Chain2Selector: evm.NewEVMEncoder(chaintest.Chain2Selector, 1, false, false),
			},
		},
		{
			name: "failure: could not create encoder",
			give: func() Proposal {
				builder := NewProposalBuilder()
				builder.SetVersion("v1")
				builder.SetValidUntil(2552083725)
				builder.AddChainMetadata(types.ChainSelector(0), types.ChainMetadata{})
				builder.SetOverridePreviousRoot(false)
				builder.AddOperation(types.Operation{ChainSelector: types.ChainSelector(0)})
				proposal, err := builder.Build()
				fmt.Println(err)
				require.NoError(t, err)
				return *proposal

			},
			wantErr: "unable to create encoder: chain family not found for selector 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			give := tt.give()
			encoders, err := give.GetEncoders()

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
	builder := NewProposalBuilder()
	builder.SetVersion("v1")
	builder.SetValidUntil(2552083725)
	builder.AddOperation(types.Operation{ChainSelector: chaintest.Chain1Selector})
	builder.SetChainMetadata(map[types.ChainSelector]types.ChainMetadata{
		chaintest.Chain1Selector: {},
		chaintest.Chain2Selector: {},
		chaintest.Chain3Selector: {},
	})
	proposal, err := builder.Build()
	require.NoError(t, err)

	// Sorted in ascending order
	want := []types.ChainSelector{
		chaintest.Chain1Selector,
		chaintest.Chain3Selector,
		chaintest.Chain2Selector,
	}
	assert.Equal(t, want, proposal.ChainSelectors())
}

func Test_Proposal_MerkleTree(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		give     func() Proposal
		wantRoot common.Hash
		wantErr  string
	}{
		{
			name: "success: generates a merkle tree",
			give: func() Proposal {
				builder := NewProposalBuilder()
				builder.SetVersion("v1")
				builder.SetValidUntil(2552083725)
				builder.AddChainMetadata(chaintest.Chain1Selector, types.ChainMetadata{StartingOpCount: 5})
				builder.AddChainMetadata(chaintest.Chain2Selector, types.ChainMetadata{StartingOpCount: 10})
				builder.AddOperation(types.Operation{
					ChainSelector: chaintest.Chain1Selector,
					Transaction: types.Transaction{
						To:               TestAddress,
						AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
						Data:             common.Hex2Bytes("0x"),
						OperationMetadata: types.OperationMetadata{
							ContractType: "Sample contract",
							Tags:         []string{"tag1", "tag2"},
						},
					},
				})
				builder.AddOperation(types.Operation{
					ChainSelector: chaintest.Chain2Selector,
					Transaction: types.Transaction{
						To:               TestAddress,
						AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
						Data:             common.Hex2Bytes("0x"),
						OperationMetadata: types.OperationMetadata{
							ContractType: "Sample contract",
							Tags:         []string{"tag1", "tag2"},
						},
					},
				})
				proposal, err := builder.Build()
				require.NoError(t, err)
				return *proposal

			},
			wantRoot: common.HexToHash("0x4e899fcdb81dc7f0c1d6609366246807fd897c2afb531ca8f907472fd6e1ed02"),
		},
		{
			name: "failure: could not get encoders",
			give: func() Proposal {
				builder := NewProposalBuilder()
				builder.SetVersion("v1")
				builder.SetValidUntil(2552083725)
				builder.AddChainMetadata(types.ChainSelector(1), types.ChainMetadata{StartingOpCount: 5})
				builder.AddOperation(types.Operation{ChainSelector: types.ChainSelector(1)})
				proposal, err := builder.Build()
				require.NoError(t, err)
				return *proposal

			},
			wantErr: "merkle tree generation error: unable to create encoder: chain family not found for selector 1",
		},
		{
			name: "failure: missing metadata when fetching nonces",
			give: func() Proposal {
				return Proposal{
					BaseProposal: BaseProposal{
						ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
							chaintest.Chain1Selector: {StartingOpCount: 5},
						},
					},
					Operations: []types.Operation{
						{ChainSelector: chaintest.Chain2Selector},
					},
				}
			},
			wantErr: "merkle tree generation error: missing metadata for chain 16015286601757825753",
		},
		{
			name: "failure: nonce must be in the range of a uint32",
			give: func() Proposal {
				builder := NewProposalBuilder()
				builder.SetVersion("v1")
				builder.SetValidUntil(2552083725)
				builder.AddChainMetadata(chaintest.Chain1Selector, types.ChainMetadata{StartingOpCount: math.MaxUint32 + 1})
				builder.AddOperation(types.Operation{ChainSelector: chaintest.Chain1Selector})
				proposal, err := builder.Build()
				require.NoError(t, err)
				return *proposal
			},
			wantErr: "merkle tree generation error: value 4294967296 exceeds uint32 range",
		},
		{
			name: "failure: could not hash operation due to invalid additional values",
			give: func() Proposal {
				return Proposal{
					BaseProposal: BaseProposal{
						ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
							chaintest.Chain1Selector: {StartingOpCount: 5},
						},
					},
					Operations: []types.Operation{
						{
							ChainSelector: chaintest.Chain1Selector,
							Transaction: types.Transaction{
								AdditionalFields: json.RawMessage([]byte(``)),
							},
						},
					},
				}
			},
			wantErr: "merkle tree generation error: unexpected end of JSON input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			give := tt.give()
			got, err := give.MerkleTree()

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
	builder := NewProposalBuilder()
	builder.SetVersion("v1")
	builder.SetValidUntil(2552083725)
	builder.AddChainMetadata(chaintest.Chain1Selector, types.ChainMetadata{})
	builder.AddChainMetadata(chaintest.Chain2Selector, types.ChainMetadata{})
	builder.SetOperations([]types.Operation{
		{ChainSelector: chaintest.Chain1Selector},
		{ChainSelector: chaintest.Chain1Selector},
		{ChainSelector: chaintest.Chain2Selector},
	})
	proposal, err := builder.Build()
	require.NoError(t, err)
	got := proposal.TransactionCounts()

	assert.Equal(t, map[types.ChainSelector]uint64{
		chaintest.Chain1Selector: 2,
		chaintest.Chain2Selector: 1,
	}, got)
}

func Test_Proposal_TransactionNonces(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    func() Proposal
		want    []uint64
		wantErr string
	}{
		{
			name: "success: returns the nonces for each transaction",
			give: func() Proposal {
				builder := NewProposalBuilder()
				builder.SetVersion("v1")
				builder.SetValidUntil(2552083725)
				builder.AddChainMetadata(chaintest.Chain1Selector, types.ChainMetadata{StartingOpCount: 5})
				builder.AddChainMetadata(chaintest.Chain2Selector, types.ChainMetadata{StartingOpCount: 10})
				builder.AddChainMetadata(chaintest.Chain3Selector, types.ChainMetadata{StartingOpCount: 15})
				builder.SetOperations([]types.Operation{
					{ChainSelector: chaintest.Chain1Selector},
					{ChainSelector: chaintest.Chain2Selector},
					{ChainSelector: chaintest.Chain1Selector},
					{ChainSelector: chaintest.Chain2Selector},
					{ChainSelector: chaintest.Chain3Selector},
				})
				proposal, err := builder.Build()
				require.NoError(t, err)
				return *proposal
			},
			want: []uint64{5, 10, 6, 11, 15},
		},
		{
			name: "failure: chain metadata not found for transaction",
			give: func() Proposal {
				return Proposal{
					BaseProposal: BaseProposal{
						ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
							chaintest.Chain1Selector: {StartingOpCount: 5},
						},
					},
					Operations: []types.Operation{
						{ChainSelector: chaintest.Chain2Selector},
					},
				}
			},
			wantErr: "missing metadata for chain 16015286601757825753",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			give := tt.give()
			got, err := give.TransactionNonces()

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}
