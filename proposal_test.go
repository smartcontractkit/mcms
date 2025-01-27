package mcms

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
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
	TestAddress   = "0x1234567890abcdef"
	ValidProposal = `{
		"version": "v1",
		"kind": "Proposal",
		"validUntil": 2004259681,
		"chainMetadata": {
			"3379446385462418246": {}
		},
		"operations": [
			{
				"chainSelector": 3379446385462418246,
				"transaction": {
					"to": "0xsomeaddress",
					"data": "EjM=",
					"additionalFields": {"value": 0}
				}
			}
		]
	}`
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
			give: ValidProposal,
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
					{
						ChainSelector: chaintest.Chain1Selector,
						Transaction: types.Transaction{
							To:               "0xsomeaddress",
							Data:             []byte{0x12, 0x33},              // Representing "0x123" as bytes
							AdditionalFields: json.RawMessage(`{"value": 0}`), // JSON-encoded `{"value": 0}`
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
				"kind": "Proposal",
				"validUntil": 2004259681,
				"chainMetadata": {},
				"operations": []
			}`,
			wantErr: "Key: 'Proposal.BaseProposal.ChainMetadata' Error:Field validation for 'ChainMetadata' failed on the 'min' tag\nKey: 'Proposal.Operations' Error:Field validation for 'Operations' failed on the 'min' tag",
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
		giveWriter func() io.Writer // Use this to overwrite the default writer
		want       string
		wantErr    string
	}{
		{
			name: "success: writes a proposal to an io.Writer",
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
							"to": "0x123",
							"additionalFields": {"value": 0},
							"data": "",
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
			wantErr: "write error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			builder := NewProposalBuilder()
			builder.SetVersion("v1").
				SetValidUntil(2004259681).
				AddChainMetadata(chaintest.Chain1Selector, types.ChainMetadata{}).
				AddOperation(types.Operation{
					ChainSelector: chaintest.Chain1Selector,
					Transaction: types.Transaction{
						To:               "0x123",
						AdditionalFields: json.RawMessage(`{"value": 0}`),
						Data:             common.Hex2Bytes("0x1"),
					},
				})

			give, err := builder.Build()
			require.NoError(t, err)

			var w io.Writer
			b := new(strings.Builder)

			if tt.giveWriter != nil {
				w = tt.giveWriter()
			} else {
				w = b
			}

			err = WriteProposal(w, give)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.JSONEq(t, tt.want, b.String())
			}
		})
	}
}

func TestLoadProposal(t *testing.T) {
	t.Parallel()
	t.Run("valid file path with KindProposal", func(t *testing.T) {
		t.Parallel()

		tempFile, err := os.CreateTemp("", "valid_proposal.txt")
		assert.Error(t, err)

		err = os.WriteFile(tempFile.Name(), []byte(ValidProposal), 0644)
		assert.Error(t, err)
		defer os.Remove(tempFile.Name())

		proposal, err := LoadProposal(types.KindProposal, tempFile.Name())

		assert.NoError(t, err)
		assert.NotNil(t, proposal)
	})

	t.Run("valid file path with KindTimelockProposal", func(t *testing.T) {
		t.Parallel()

		tempFile, err := os.CreateTemp("", "valid_timelock_proposal.txt")
		assert.Error(t, err)

		err = os.WriteFile(tempFile.Name(), []byte(ValidTimelockProposal), 0644)
		assert.Error(t, err)
		defer os.Remove(tempFile.Name())

		proposal, err := LoadProposal(types.KindTimelockProposal, tempFile.Name())

		assert.NoError(t, err)
		assert.NotNil(t, proposal)
	})

	t.Run("unknown proposal type", func(t *testing.T) {
		t.Parallel()

		filePath := "testdata/valid_proposal.txt"
		err := os.WriteFile(filePath, []byte(ValidProposal), 0644)
		assert.Error(t, err)
		defer os.Remove(filePath)

		proposal, err := LoadProposal(types.ProposalKind("unknown"), filePath)

		assert.Error(t, err)
		assert.Nil(t, proposal)
		assert.Equal(t, "unknown proposal type", err.Error())
	})

	t.Run("invalid file path", func(t *testing.T) {
		t.Parallel()

		filePath := "invalid_path.txt"

		proposal, err := LoadProposal(types.KindProposal, filePath)

		assert.Error(t, err)
		assert.Nil(t, proposal)
	})
}

func Test_Proposal_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		giveFunc func(*Proposal)
		wantErrs []string
	}{
		{
			name: "valid",
			giveFunc: func(*Proposal) {
				// NOOP: All  are valid
			},
		},
		{
			name: "required fields validation",
			giveFunc: func(p *Proposal) {
				// Overwrite the timelock proposal with an empty one
				*p = Proposal{
					BaseProposal: BaseProposal{},
				}
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
			giveFunc: func(p *Proposal) {
				p.ChainMetadata = map[types.ChainSelector]types.ChainMetadata{}
				p.Operations = []types.Operation{}
			},
			wantErrs: []string{
				"Key: 'Proposal.BaseProposal.ChainMetadata' Error:Field validation for 'ChainMetadata' failed on the 'min' tag",
				"Key: 'Proposal.Operations' Error:Field validation for 'Operations' failed on the 'min' tag",
			},
		},
		{
			name: "oneof validation",
			giveFunc: func(p *Proposal) {
				p.Version = "invalidversion"
				p.Kind = "invalidkind"
			},
			wantErrs: []string{
				"Key: 'Proposal.BaseProposal.Version' Error:Field validation for 'Version' failed on the 'oneof' tag",
				"Key: 'Proposal.BaseProposal.Kind' Error:Field validation for 'Kind' failed on the 'oneof' tag",
			},
		},
		{
			name: "all chain selectors in transactions must be present in chain metadata",
			giveFunc: func(p *Proposal) {
				p.Operations[0].ChainSelector = chaintest.Chain2Selector
			},
			wantErrs: []string{
				"missing metadata for chain 16015286601757825753",
			},
		},
		{
			name: "invalid chain family specific operation data (additional fields)",
			giveFunc: func(p *Proposal) {
				p.Operations[0].Transaction.AdditionalFields = json.RawMessage([]byte(`{"value": -100}`))
			},
			wantErrs: []string{
				"invalid EVM value: -100",
			},
		},
		{
			name: "proposal has expired (validUntil)",
			giveFunc: func(p *Proposal) {
				p.ValidUntil = 1
			},
			wantErrs: []string{
				"invalid valid until: 1",
			},
		},
		{
			name: "operations dive: required fields validation",
			giveFunc: func(p *Proposal) {
				p.Operations = []types.Operation{{}}
			},
			wantErrs: []string{
				"Key: 'Proposal.Operations[0].ChainSelector' Error:Field validation for 'ChainSelector' failed on the 'required' tag",
				"Key: 'Proposal.Operations[0].Transaction.To' Error:Field validation for 'To' failed on the 'required' tag",
				"Key: 'Proposal.Operations[0].Transaction.Data' Error:Field validation for 'Data' failed on the 'required' tag",
				"Key: 'Proposal.Operations[0].Transaction.AdditionalFields' Error:Field validation for 'AdditionalFields' failed on the 'required' tag",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// A valid proposal to be passed to the giveFunc for the test case to modify
			proposal := Proposal{
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
						},
					},
				},
			}

			tt.giveFunc(&proposal)

			err := proposal.Validate()

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
		setup   func(p *Proposal)
		want    map[types.ChainSelector]sdk.Encoder
		wantErr string
	}{
		{
			name: "success",
			setup: func(_ *Proposal) {
				// NOP: valid proposal.
			},
			want: map[types.ChainSelector]sdk.Encoder{
				chaintest.Chain1Selector: evm.NewEncoder(chaintest.Chain1Selector, 2, false, false),
				chaintest.Chain2Selector: evm.NewEncoder(chaintest.Chain2Selector, 1, false, false),
			},
		},
		{
			name: "failure: could not create encoder",
			setup: func(p *Proposal) {
				p.ChainMetadata = map[types.ChainSelector]types.ChainMetadata{
					types.ChainSelector(0): {},
				}
			},
			wantErr: "unable to create encoder: chain family not found for selector 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			builder := NewProposalBuilder()
			builder.SetVersion("v1").
				SetValidUntil(2552083725).
				AddChainMetadata(chaintest.Chain1Selector, types.ChainMetadata{}).
				AddChainMetadata(chaintest.Chain2Selector, types.ChainMetadata{}).
				AddOperation(types.Operation{
					ChainSelector: chaintest.Chain1Selector,
					Transaction: types.Transaction{
						To:               TestAddress,
						AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
						Data:             common.Hex2Bytes("0x1"),
					},
				}).
				AddOperation(types.Operation{
					ChainSelector: chaintest.Chain1Selector,
					Transaction: types.Transaction{
						To:               TestAddress,
						AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
						Data:             common.Hex2Bytes("0x2"),
					},
				}).
				AddOperation(types.Operation{
					ChainSelector: chaintest.Chain2Selector,
					Transaction: types.Transaction{
						To:               TestAddress,
						AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
						Data:             common.Hex2Bytes("0x3"),
					},
				})

			give, err := builder.Build()
			tt.setup(give)

			require.NoError(t, err)
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
	builder.SetVersion("v1").
		SetValidUntil(2552083725).
		AddOperation(types.Operation{
			ChainSelector: chaintest.Chain1Selector,
			Transaction: types.Transaction{
				To:               TestAddress,
				AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
				Data:             common.Hex2Bytes("0x"),
			},
		}).
		SetChainMetadata(map[types.ChainSelector]types.ChainMetadata{
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
		setup    func(p *Proposal)
		wantRoot common.Hash
		wantErr  string
	}{
		{
			name: "success: generates a merkle tree",
			setup: func(_ *Proposal) {
				// NOP: valid proposal.
			},
			wantRoot: common.HexToHash("0x4fdb98431759bbcab33cbd1b4034fea43ef360f11b1de4ca10fc20f8916bda19"),
		},
		{
			name: "failure: could not get encoders",
			setup: func(p *Proposal) {
				p.ChainMetadata = map[types.ChainSelector]types.ChainMetadata{
					types.ChainSelector(1): {StartingOpCount: 5},
				}
			},
			wantErr: "merkle tree generation error: unable to create encoder: chain family not found for selector 1",
		},
		{
			name: "failure: missing metadata when fetching nonces",
			setup: func(p *Proposal) {
				p.ChainMetadata = map[types.ChainSelector]types.ChainMetadata{
					chaintest.Chain1Selector: {StartingOpCount: 5},
				}
				p.Operations = []types.Operation{
					{ChainSelector: chaintest.Chain2Selector},
				}
			},
			wantErr: "merkle tree generation error: missing metadata for chain 16015286601757825753",
		},
		{
			name: "failure: nonce must be in the range of a uint32",
			setup: func(p *Proposal) {
				p.ChainMetadata = map[types.ChainSelector]types.ChainMetadata{
					chaintest.Chain2Selector: {StartingOpCount: math.MaxUint32 + 1},
					chaintest.Chain1Selector: {StartingOpCount: math.MaxUint32 + 1},
				}
			},
			wantErr: "merkle tree generation error: value 4294967296 exceeds uint32 range",
		},
		{
			name: "failure: could not hash operation due to invalid additional values",
			setup: func(p *Proposal) {
				p.Operations = []types.Operation{
					{
						ChainSelector: chaintest.Chain1Selector,
						Transaction: types.Transaction{
							AdditionalFields: json.RawMessage([]byte(``)),
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
			builder := NewProposalBuilder()
			builder.SetVersion("v1").
				SetValidUntil(2552083725).
				AddChainMetadata(chaintest.Chain1Selector, types.ChainMetadata{StartingOpCount: 5}).
				AddChainMetadata(chaintest.Chain2Selector, types.ChainMetadata{StartingOpCount: 10}).
				AddOperation(types.Operation{
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
				}).
				AddOperation(types.Operation{
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
			give, err := builder.Build()
			require.NoError(t, err)

			tt.setup(give)

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
	builder.SetVersion("v1").
		SetValidUntil(2552083725).
		AddChainMetadata(chaintest.Chain1Selector, types.ChainMetadata{}).
		AddChainMetadata(chaintest.Chain2Selector, types.ChainMetadata{}).
		SetOperations([]types.Operation{
			{
				ChainSelector: chaintest.Chain1Selector,
				Transaction: types.Transaction{
					To:               TestAddress,
					AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
					Data:             common.Hex2Bytes("0x1"),
				},
			},
			{
				ChainSelector: chaintest.Chain1Selector,
				Transaction: types.Transaction{
					To:               TestAddress,
					AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
					Data:             common.Hex2Bytes("0x2"),
				},
			},
			{
				ChainSelector: chaintest.Chain2Selector,
				Transaction: types.Transaction{
					To:               TestAddress,
					AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
					Data:             common.Hex2Bytes("0x3"),
				},
			},
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
		setup   func(b *Proposal)
		want    []uint64
		wantErr string
	}{
		{
			name: "success: returns the nonces for each transaction",
			setup: func(b *Proposal) {
				// NOP: valid proposal.
			},
			want: []uint64{5, 10, 6, 11, 7},
		},
		{
			name: "failure: chain metadata not found for transaction",
			setup: func(p *Proposal) {
				p.ChainMetadata = map[types.ChainSelector]types.ChainMetadata{
					chaintest.Chain1Selector: {StartingOpCount: 5},
				}
				p.Operations = []types.Operation{
					{ChainSelector: chaintest.Chain2Selector},
				}
			},
			wantErr: "missing metadata for chain 16015286601757825753",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			builder := NewProposalBuilder()
			builder.SetVersion("v1").
				SetValidUntil(2552083725).
				AddChainMetadata(chaintest.Chain1Selector, types.ChainMetadata{StartingOpCount: 5}).
				AddChainMetadata(chaintest.Chain2Selector, types.ChainMetadata{StartingOpCount: 10}).
				AddChainMetadata(chaintest.Chain3Selector, types.ChainMetadata{StartingOpCount: 15}).
				SetOperations([]types.Operation{
					{
						ChainSelector: chaintest.Chain1Selector,
						Transaction: types.Transaction{
							To:               TestAddress,
							AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
							Data:             common.Hex2Bytes("0x1"),
						},
					},
					{
						ChainSelector: chaintest.Chain2Selector,
						Transaction: types.Transaction{
							To:               TestAddress,
							AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
							Data:             common.Hex2Bytes("0x1"),
						},
					},
					{
						ChainSelector: chaintest.Chain1Selector,
						Transaction: types.Transaction{
							To:               TestAddress,
							AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
							Data:             common.Hex2Bytes("0x2"),
						},
					},
					{
						ChainSelector: chaintest.Chain2Selector,
						Transaction: types.Transaction{
							To:               TestAddress,
							AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
							Data:             common.Hex2Bytes("0x2"),
						},
					},
					{
						ChainSelector: chaintest.Chain1Selector,
						Transaction: types.Transaction{
							To:               TestAddress,
							AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
							Data:             common.Hex2Bytes("0x1"),
						},
					},
				})
			give, err := builder.Build()
			require.NoError(t, err)

			tt.setup(give)
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
