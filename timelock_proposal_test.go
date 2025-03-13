package mcms

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	evmsdk "github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/sdk/mocks"
	"github.com/smartcontractkit/mcms/types"
)

var ValidTimelockProposal = `{
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
	"action": "schedule",
	"delay": "1h",
	"timelockAddresses": {
		"16015286601757825753": "0x01"
	},
	"operations": [
		{
			"chainSelector": 16015286601757825753,
			"transactions": [
				{
					"to": "0x0000000000000000000000000000000000000000",
					"additionalFields": {"value": 0},
					"data": "ZGF0YQ=="
				}
			]
		}
	]
}`

func Test_NewTimelockProposal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		give             string
		givePredecessors []string
		want             TimelockProposal
		wantErr          string
	}{
		{
			name:             "success: initializes a proposal from an io.Reader",
			give:             ValidTimelockProposal,
			givePredecessors: []string{},
			want: TimelockProposal{
				BaseProposal: BaseProposal{
					Version:     "v1",
					Kind:        types.KindTimelockProposal,
					ValidUntil:  2004259681,
					Description: "Test proposal",
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						chaintest.Chain2Selector: {
							StartingOpCount: 0,
							MCMAddress:      "0x0000000000000000000000000000000000000000",
						},
					},
					OverridePreviousRoot: false,
				},
				Action: types.TimelockActionSchedule,
				Delay:  types.MustParseDuration("1h"),
				TimelockAddresses: map[types.ChainSelector]string{
					chaintest.Chain2Selector: "0x01",
				},
				Operations: []types.BatchOperation{
					{
						ChainSelector: chaintest.Chain2Selector,
						Transactions: []types.Transaction{
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
			name:             "success: initializes a proposal with 1 predecessor from an io.Reader",
			give:             ValidTimelockProposal,
			givePredecessors: []string{ValidTimelockProposal},
			want: TimelockProposal{
				BaseProposal: BaseProposal{
					Version:     "v1",
					Kind:        types.KindTimelockProposal,
					ValidUntil:  2004259681,
					Description: "Test proposal",
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						chaintest.Chain2Selector: {
							StartingOpCount: 1,
							MCMAddress:      "0x0000000000000000000000000000000000000000",
						},
					},
					OverridePreviousRoot: false,
				},
				Action: types.TimelockActionSchedule,
				Delay:  types.MustParseDuration("1h"),
				TimelockAddresses: map[types.ChainSelector]string{
					chaintest.Chain2Selector: "0x01",
				},
				Operations: []types.BatchOperation{
					{
						ChainSelector: chaintest.Chain2Selector,
						Transactions: []types.Transaction{
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
			name:             "success: initializes a proposal with 2 predecessors from an io.Reader",
			give:             ValidTimelockProposal,
			givePredecessors: []string{ValidTimelockProposal, ValidTimelockProposal},
			want: TimelockProposal{
				BaseProposal: BaseProposal{
					Version:     "v1",
					Kind:        types.KindTimelockProposal,
					ValidUntil:  2004259681,
					Description: "Test proposal",
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						chaintest.Chain2Selector: {
							StartingOpCount: 2,
							MCMAddress:      "0x0000000000000000000000000000000000000000",
						},
					},
					OverridePreviousRoot: false,
				},
				Action: types.TimelockActionSchedule,
				Delay:  types.MustParseDuration("1h"),
				TimelockAddresses: map[types.ChainSelector]string{
					chaintest.Chain2Selector: "0x01",
				},
				Operations: []types.BatchOperation{
					{
						ChainSelector: chaintest.Chain2Selector,
						Transactions: []types.Transaction{
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
			name: "success: valid with 0 delay",
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
				"action": "schedule",
				"delay": "0s",
				"timelockAddresses": {
					"16015286601757825753": "0x01"
				},
				"operations": [
					{
						"chainSelector": 16015286601757825753,
						"transactions": [
							{
								"to": "0x0000000000000000000000000000000000000000",
								"additionalFields": {"value": 0},
								"data": "ZGF0YQ=="
							}
						]
					}
				]
			}`,
			givePredecessors: []string{},
			want: TimelockProposal{
				BaseProposal: BaseProposal{
					Version:     "v1",
					Kind:        types.KindTimelockProposal,
					ValidUntil:  2004259681,
					Description: "Test proposal",
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						chaintest.Chain2Selector: {
							StartingOpCount: 0,
							MCMAddress:      "0x0000000000000000000000000000000000000000",
						},
					},
					OverridePreviousRoot: false,
				},
				Action: types.TimelockActionSchedule,
				Delay:  types.MustParseDuration("0s"),
				TimelockAddresses: map[types.ChainSelector]string{
					chaintest.Chain2Selector: "0x01",
				},
				Operations: []types.BatchOperation{
					{
						ChainSelector: chaintest.Chain2Selector,
						Transactions: []types.Transaction{
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
			name:             "failure: could not unmarshal JSON",
			give:             `invalid`,
			givePredecessors: []string{},
			wantErr:          "failed to decode and validate target proposal: failed to decode proposal: invalid character 'i' looking for beginning of value",
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
				"action": "schedule",
				"delay": "1h0m0s",
				"timelockAddresses": {
					"16015286601757825753": "0x01"
				},
				"operations": [
					{
						"chainSelector": 16015286601757825753,
						"transactions": [
							{
								"to": "0x0000000000000000000000000000000000000000",
								"additionalFields": {"value": 0},
								"data": "ZGF0YQ=="
							}
						]
					}
				]
			}`,
			givePredecessors: []string{},
			wantErr:          "failed to decode and validate target proposal: failed to validate proposal: Key: 'TimelockProposal.BaseProposal.ChainMetadata' Error:Field validation for 'ChainMetadata' failed on the 'min' tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			give := strings.NewReader(tt.give)
			givePredecessors := []io.Reader{}
			for _, p := range tt.givePredecessors {
				givePredecessors = append(givePredecessors, strings.NewReader(p))
			}

			got, err := NewTimelockProposal(give, WithPredecessors(givePredecessors))

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
		giveWriter func() io.Writer // Use this to overwrite the default writer
		want       string
		wantErr    string
	}{
		{
			name: "success: writes a proposal to an io.Writer",
			want: `{
				"version": "v1",
				"kind": "TimelockProposal",
				"validUntil": 2004259681,
				"chainMetadata": {
					"16015286601757825753": {
						"mcmAddress": "0x0000000000000000000000000000000000000000",
						"startingOpCount": 0,
						"additionalFields": null
					}
				},
				"description": "Test proposal",
				"overridePreviousRoot": false,
				"action": "schedule",
				"delay": "1h0m0s",
				"signatures": null,
				"timelockAddresses": {
					"16015286601757825753": ""
				},
				"operations": [
					{
						"chainSelector": 16015286601757825753,
						"transactions": [
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
			wantErr: "write error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			builder := NewTimelockProposalBuilder()
			builder.SetVersion("v1").
				SetValidUntil(2004259681).
				SetDescription("Test proposal").
				SetChainMetadata(map[types.ChainSelector]types.ChainMetadata{
					chaintest.Chain2Selector: {
						StartingOpCount: 0,
						MCMAddress:      "0x0000000000000000000000000000000000000000",
					},
				}).
				SetOverridePreviousRoot(false).
				SetAction(types.TimelockActionSchedule).
				SetDelay(types.MustParseDuration("1h")).
				AddTimelockAddress(chaintest.Chain2Selector, "").
				SetOperations([]types.BatchOperation{{
					ChainSelector: chaintest.Chain2Selector,
					Transactions: []types.Transaction{
						{
							To:               "0x0000000000000000000000000000000000000000",
							AdditionalFields: []byte(`{"value": 0}`),
							Data:             []byte("data"),
						},
					},
				}})
			give, err := builder.Build()
			require.NoError(t, err)
			var w io.Writer
			b := new(strings.Builder)

			if tt.giveWriter != nil {
				w = tt.giveWriter()
			} else {
				w = b
			}

			err = WriteTimelockProposal(w, give)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.JSONEq(t, tt.want, b.String())
			}
		})
	}
}

func Test_TimelockProposal_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		giveFunc func(*TimelockProposal)
		wantErrs []string
	}{
		{
			name: "valid with schedule action and delay",
			giveFunc: func(*TimelockProposal) {
				// NOOP: All fields are valid
			},
		},
		{
			name: "valid with non scheduled action and no delay",
			giveFunc: func(p *TimelockProposal) {
				p.Action = types.TimelockActionCancel
				p.Delay = types.Duration{}
			},
		},
		{
			name: "required fields validation",
			giveFunc: func(p *TimelockProposal) {
				// Overwrite the timelock proposal with an empty one
				*p = TimelockProposal{
					BaseProposal: BaseProposal{},
				}
			},
			wantErrs: []string{
				"Key: 'TimelockProposal.BaseProposal.Version' Error:Field validation for 'Version' failed on the 'required' tag",
				"Key: 'TimelockProposal.BaseProposal.Kind' Error:Field validation for 'Kind' failed on the 'required' tag",
				"Key: 'TimelockProposal.BaseProposal.ValidUntil' Error:Field validation for 'ValidUntil' failed on the 'required' tag",
				"Key: 'TimelockProposal.BaseProposal.ChainMetadata' Error:Field validation for 'ChainMetadata' failed on the 'required' tag",
				"Key: 'TimelockProposal.Operations' Error:Field validation for 'Operations' failed on the 'required' tag",
				"Key: 'TimelockProposal.TimelockAddresses' Error:Field validation for 'TimelockAddresses' failed on the 'required' tag",
				"Key: 'TimelockProposal.Action' Error:Field validation for 'Action' failed on the 'required' tag",
			},
		},
		{
			name: "min validation",
			giveFunc: func(p *TimelockProposal) {
				p.ChainMetadata = map[types.ChainSelector]types.ChainMetadata{}
				p.TimelockAddresses = map[types.ChainSelector]string{}
				p.Operations = []types.BatchOperation{}
			},
			wantErrs: []string{
				"Key: 'TimelockProposal.BaseProposal.ChainMetadata' Error:Field validation for 'ChainMetadata' failed on the 'min' tag",
				"Key: 'TimelockProposal.Operations' Error:Field validation for 'Operations' failed on the 'min' tag",
				"Key: 'TimelockProposal.TimelockAddresses' Error:Field validation for 'TimelockAddresses' failed on the 'min' tag",
			},
		},
		{
			name: "oneof validation",
			giveFunc: func(p *TimelockProposal) {
				p.Version = "invalidversion"
				p.Kind = "invalidking"
				p.Action = "invalidaction"
			},
			wantErrs: []string{
				"Key: 'TimelockProposal.BaseProposal.Version' Error:Field validation for 'Version' failed on the 'oneof' tag",
				"Key: 'TimelockProposal.BaseProposal.Kind' Error:Field validation for 'Kind' failed on the 'oneof' tag",
				"Key: 'TimelockProposal.Action' Error:Field validation for 'Action' failed on the 'oneof' tag",
			},
		},
		{
			name: "invalid kind: must be TimelockProposal",
			giveFunc: func(p *TimelockProposal) {
				p.Kind = types.KindProposal
			},
			wantErrs: []string{
				"invalid proposal kind: Proposal, value accepted is TimelockProposal",
			},
		},
		{
			name: "all chain selectors in transactions must be present in chain metadata",
			giveFunc: func(p *TimelockProposal) {
				p.Operations[0].ChainSelector = chaintest.Chain2Selector
			},
			wantErrs: []string{
				"missing metadata for chain 16015286601757825753",
			},
		},
		{
			name: "invalid chain family specific operation data (additional fields)",
			giveFunc: func(p *TimelockProposal) {
				p.Operations[0].Transactions[0].AdditionalFields = json.RawMessage([]byte(`{"value": -100}`))
			},
			wantErrs: []string{
				"invalid EVM value: -100",
			},
		},
		{
			name: "proposal has expired (validUntil)",
			giveFunc: func(p *TimelockProposal) {
				p.ValidUntil = 1
			},
			wantErrs: []string{
				"invalid valid until: 1",
			},
		},
		{
			name: "must have at least 1 transaction in a batch operation",
			giveFunc: func(p *TimelockProposal) {
				p.Operations = []types.BatchOperation{
					{
						ChainSelector: chaintest.Chain1Selector,
						Transactions:  []types.Transaction{},
					},
				}
			},
			wantErrs: []string{
				"Key: 'TimelockProposal.Operations[0].Transactions' Error:Field validation for 'Transactions' failed on the 'min' tag",
			},
		},
		{
			name: "operations dive: required fields validation",
			giveFunc: func(p *TimelockProposal) {
				p.Operations[0] = types.BatchOperation{}
			},
			wantErrs: []string{
				"Key: 'TimelockProposal.Operations[0].ChainSelector' Error:Field validation for 'ChainSelector' failed on the 'required' tag",
				"Key: 'TimelockProposal.Operations[0].Transactions' Error:Field validation for 'Transactions' failed on the 'required' tag",
			},
		},
		{
			name: "transactions dive: required fields validation",
			giveFunc: func(p *TimelockProposal) {
				p.Operations[0].Transactions[0] = types.Transaction{}
			},
			wantErrs: []string{
				"Key: 'TimelockProposal.Operations[0].Transactions[0].To' Error:Field validation for 'To' failed on the 'required' tag",
				"Key: 'TimelockProposal.Operations[0].Transactions[0].Data' Error:Field validation for 'Data' failed on the 'required' tag",
				"Key: 'TimelockProposal.Operations[0].Transactions[0].AdditionalFields' Error:Field validation for 'AdditionalFields' failed on the 'required' tag",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// A valid timelock proposal to be passed to the giveFunc for the test case to modify
			proposal := TimelockProposal{
				BaseProposal: BaseProposal{
					Version:    "v1",
					Kind:       types.KindTimelockProposal,
					ValidUntil: 2004259681,
					Signatures: []types.Signature{},
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						chaintest.Chain1Selector: {
							StartingOpCount: 1,
							MCMAddress:      TestAddress,
						},
					},
				},
				Action: types.TimelockActionSchedule,
				Delay:  types.MustParseDuration("1h"),
				TimelockAddresses: map[types.ChainSelector]string{
					chaintest.Chain1Selector: "0x01",
				},
				Operations: []types.BatchOperation{
					{
						ChainSelector: chaintest.Chain1Selector,
						Transactions: []types.Transaction{
							{
								To:               TestAddress,
								AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
								Data:             common.Hex2Bytes("0x"),
							},
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

func Test_TimelockProposal_Convert(t *testing.T) {
	t.Parallel()

	var (
		ctx = context.Background()

		validChainMetadata = map[types.ChainSelector]types.ChainMetadata{
			chaintest.Chain1Selector: {
				StartingOpCount: 1,
				MCMAddress:      "0x123",
			},
			chaintest.Chain2Selector: {
				StartingOpCount: 1,
				MCMAddress:      "0x456",
			},
		}

		validTimelockAddresses = map[types.ChainSelector]string{
			chaintest.Chain1Selector: "0x123",
			chaintest.Chain2Selector: "0x456",
		}

		validTx1 = types.Transaction{
			To:               "0x123",
			AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
			Data:             common.Hex2Bytes("0x"),
			OperationMetadata: types.OperationMetadata{
				ContractType: "Sample contract",
				Tags:         []string{"tag1", "tag2"},
			},
		}

		validTx2 = types.Transaction{
			To:               "0x123",
			AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
			Data:             common.Hex2Bytes("0x1"),
			OperationMetadata: types.OperationMetadata{
				ContractType: "Sample contract",
				Tags:         []string{"tag1", "tag2"},
			},
		}

		validTx3 = types.Transaction{
			To:               "0x123",
			AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
			Data:             common.Hex2Bytes("0x2"),
			OperationMetadata: types.OperationMetadata{
				ContractType: "Sample contract",
				Tags:         []string{"tag1", "tag2"},
			},
		}

		validBatchOps = []types.BatchOperation{
			{
				ChainSelector: chaintest.Chain1Selector,
				Transactions: []types.Transaction{
					validTx1,
				},
			},
			{
				ChainSelector: chaintest.Chain2Selector,
				Transactions: []types.Transaction{
					validTx2,
				},
			},
			{
				ChainSelector: chaintest.Chain2Selector,
				Transactions: []types.Transaction{
					validTx3,
				},
			},
		}

		proposal = TimelockProposal{
			BaseProposal: BaseProposal{
				Version:              "v1",
				Kind:                 types.KindTimelockProposal,
				Description:          "description",
				ValidUntil:           2004259681,
				OverridePreviousRoot: false,
				Signatures:           []types.Signature{},
				ChainMetadata:        validChainMetadata,
			},
			Action:            types.TimelockActionSchedule,
			Delay:             types.MustParseDuration("1h"),
			TimelockAddresses: validTimelockAddresses,
			Operations:        validBatchOps,
		}

		converters = map[types.ChainSelector]sdk.TimelockConverter{
			chaintest.Chain1Selector: &evmsdk.TimelockConverter{},
			chaintest.Chain2Selector: &evmsdk.TimelockConverter{},
		}
	)

	mcmsProposal, predecessors, err := proposal.Convert(ctx, converters)
	require.NoError(t, err)

	require.Equal(t, "v1", mcmsProposal.Version)
	require.Equal(t, uint32(2004259681), mcmsProposal.ValidUntil)
	require.Equal(t, []types.Signature{}, mcmsProposal.Signatures)
	require.False(t, mcmsProposal.OverridePreviousRoot)
	require.Equal(t, validChainMetadata, mcmsProposal.ChainMetadata)
	require.Equal(t, "description", mcmsProposal.Description)
	require.Len(t, mcmsProposal.Operations, 3)

	require.Len(t, predecessors, 3)
	require.Equal(t, predecessors[0], ZERO_HASH)
	require.Equal(t, predecessors[1], ZERO_HASH)
	require.NotEqual(t, predecessors[2], ZERO_HASH)
}

func TestProposal_WithSaltOverride(t *testing.T) {
	t.Parallel()
	builder := NewTimelockProposalBuilder()
	builder.SetVersion("v1").
		SetAction(types.TimelockActionSchedule).
		SetValidUntil(2552083725).
		AddTimelockAddress(chaintest.Chain1Selector, "0x01").
		AddChainMetadata(chaintest.Chain1Selector, types.ChainMetadata{}).
		AddOperation(types.BatchOperation{
			ChainSelector: chaintest.Chain1Selector,
			Transactions: []types.Transaction{
				{
					To:               TestAddress,
					AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
					Data:             common.Hex2Bytes("0x1"),
				},
			},
		})
	proposal, err := builder.Build()
	require.NoError(t, err)
	salt := common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	proposal.SaltOverride = &salt
	assert.Equal(t, salt, *proposal.SaltOverride)
	saltBytes := proposal.Salt()
	assert.Equal(t, salt, common.BytesToHash(saltBytes[:]))
}

func Test_TimelockProposal_Decode(t *testing.T) {
	t.Parallel()

	// Get ABI
	timelockAbi, err := bindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err)
	exampleRole := crypto.Keccak256Hash([]byte("EXAMPLE_ROLE"))

	// Grant role data
	grantRoleData, err := timelockAbi.Pack("grantRole", [32]byte(exampleRole), common.HexToAddress("0x123"))
	require.NoError(t, err)

	tests := []struct {
		name    string
		setup   func(t *testing.T) (map[types.ChainSelector]sdk.Decoder, map[string]string)
		give    []types.BatchOperation
		want    [][]sdk.DecodedOperation
		wantErr string
	}{
		{
			name: "success: decodes a batch operation",
			setup: func(t *testing.T) (map[types.ChainSelector]sdk.Decoder, map[string]string) {
				t.Helper()
				decoders := map[types.ChainSelector]sdk.Decoder{
					chaintest.Chain1Selector: evm.NewDecoder(),
				}

				return decoders, map[string]string{"RBACTimelock": bindings.RBACTimelockABI}
			},
			give: []types.BatchOperation{
				{
					ChainSelector: chaintest.Chain1Selector,
					Transactions: []types.Transaction{
						evm.NewTransaction(
							common.HexToAddress("0xTestTarget"),
							grantRoleData,
							big.NewInt(0),
							"RBACTimelock",
							[]string{"grantRole"},
						),
					},
				},
			},
			want: [][]sdk.DecodedOperation{
				{
					&evm.DecodedOperation{
						FunctionName: "grantRole",
						InputKeys:    []string{"role", "account"},
						InputArgs:    []any{[32]byte(exampleRole.Bytes()), common.HexToAddress("0x0000000000000000000000000000000000000123")},
					},
				},
			},
		},
		{
			name: "failure: missing chain decoder",
			setup: func(t *testing.T) (map[types.ChainSelector]sdk.Decoder, map[string]string) {
				t.Helper()
				return map[types.ChainSelector]sdk.Decoder{}, map[string]string{}
			},
			give: []types.BatchOperation{
				{
					ChainSelector: chaintest.Chain1Selector,
					Transactions: []types.Transaction{
						evm.NewTransaction(
							common.HexToAddress("0xTestTarget"),
							grantRoleData,
							big.NewInt(0),
							"RBACTimelock",
							[]string{"grantRole"},
						),
					},
				},
			},
			want:    nil,
			wantErr: "no decoder found for chain selector 3379446385462418246",
		},
		{
			name: "failure: missing contract ABI",
			setup: func(t *testing.T) (map[types.ChainSelector]sdk.Decoder, map[string]string) {
				t.Helper()
				decoders := map[types.ChainSelector]sdk.Decoder{
					chaintest.Chain1Selector: evm.NewDecoder(),
				}

				return decoders, map[string]string{}
			},
			give: []types.BatchOperation{
				{
					ChainSelector: chaintest.Chain1Selector,
					Transactions: []types.Transaction{
						evm.NewTransaction(
							common.HexToAddress("0xTestTarget"),
							grantRoleData,
							big.NewInt(0),
							"RBACTimelock",
							[]string{"grantRole"},
						),
					},
				},
			},
			want:    nil,
			wantErr: "no contract interfaces found for contract type RBACTimelock",
		},
		{
			name: "failure: unable to decode operation",
			setup: func(t *testing.T) (map[types.ChainSelector]sdk.Decoder, map[string]string) {
				t.Helper()
				mockDecoder := mocks.NewDecoder(t)
				mockDecoder.EXPECT().Decode(mock.Anything, mock.Anything).Return(nil, errors.New("decode error"))
				decoders := map[types.ChainSelector]sdk.Decoder{
					chaintest.Chain1Selector: mockDecoder,
				}

				return decoders, map[string]string{"RBACTimelock": bindings.RBACTimelockABI}
			},
			give: []types.BatchOperation{
				{
					ChainSelector: chaintest.Chain1Selector,
					Transactions: []types.Transaction{
						evm.NewTransaction(
							common.HexToAddress("0xTestTarget"),
							grantRoleData,
							big.NewInt(0),
							"RBACTimelock",
							[]string{"grantRole"},
						),
					},
				},
			},
			want:    nil,
			wantErr: "unable to decode operation: decode error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			proposal := TimelockProposal{
				BaseProposal: BaseProposal{},
				Operations:   tt.give,
			}

			decoders, contractInterfaces := tt.setup(t)
			got, err := proposal.Decode(decoders, contractInterfaces)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestProposal_WithoutSaltOverride(t *testing.T) {
	t.Parallel()
	builder := NewTimelockProposalBuilder()
	builder.SetVersion("v1").
		SetAction(types.TimelockActionSchedule).
		SetValidUntil(2552083725).
		AddChainMetadata(chaintest.Chain1Selector, types.ChainMetadata{}).
		AddTimelockAddress(chaintest.Chain1Selector, "0x01").
		AddOperation(types.BatchOperation{
			ChainSelector: chaintest.Chain1Selector,
			Transactions: []types.Transaction{
				{
					To:               TestAddress,
					AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
					Data:             common.Hex2Bytes("0x1"),
				},
			},
		})
	proposal, err := builder.Build()
	require.NoError(t, err)
	assert.Nil(t, proposal.SaltOverride)
	saltBytes := proposal.Salt()
	assert.NotNil(t, saltBytes)
	assert.NotEqual(t, common.Hash{}, saltBytes)
}
