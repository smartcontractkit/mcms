package mcms

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/types"
)

func TestTimelockProposal_Merge(t *testing.T) {
	t.Parallel()

	sig1, err := types.NewSignatureFromBytes(common.Hex2Bytes("0000000000000000000000000000000000000000000000001111111111111111000000000000000000000000000000000000000000000000aaaaaaaaaaaaaaaa1b"))
	require.NoError(t, err)
	sig2, err := types.NewSignatureFromBytes(common.Hex2Bytes("0000000000000000000000000000000000000000000000002222222222222222000000000000000000000000000000000000000000000000bbbbbbbbbbbbbbbb1b"))
	require.NoError(t, err)
	sig3, err := types.NewSignatureFromBytes(common.Hex2Bytes("0000000000000000000000000000000000000000000000003333333333333333000000000000000000000000000000000000000000000000cccccccccccccccc1b"))
	require.NoError(t, err)

	baseProposalBuilder := func() *TimelockProposalBuilder {
		return NewTimelockProposalBuilder().
			SetDescription("proposal 1").
			SetVersion("v1").
			SetAction(types.TimelockActionSchedule).
			SetValidUntil(2051222400). // 2035-01-01 00:00:00 UTC
			SetDelay(types.NewDuration(1*time.Minute)).
			AddSignature(sig1).
			AddTimelockAddress(chaintest.Chain1Selector, "0xchain1TimelockAddress").
			AddChainMetadata(chaintest.Chain1Selector, types.ChainMetadata{
				StartingOpCount:  1,
				MCMAddress:       "0xchain1McmAddress",
				AdditionalFields: json.RawMessage(`{"chain1":"value1"}`),
			}).
			AddOperation(types.BatchOperation{
				ChainSelector: chaintest.Chain1Selector,
				Transactions: []types.Transaction{
					{
						To:               "0xchain1ToAddress1",
						Data:             common.Hex2Bytes("0x0001"),
						AdditionalFields: json.RawMessage(`{"value":0}`),
					},
				},
			})
	}

	tests := []struct {
		name      string
		proposal1 *TimelockProposal
		proposal2 *TimelockProposal
		wantErr   string
		assert    func(t *testing.T, merged *TimelockProposal)
	}{
		{
			name:      "success: merge with non-overlapping chains and addresses",
			proposal1: mustBuild(t, baseProposalBuilder()),
			proposal2: mustBuild(t, NewTimelockProposalBuilder().
				SetDescription("proposal 2").
				SetVersion("v1").
				SetAction(types.TimelockActionSchedule).
				SetValidUntil(2053987200). // 2035-02-02 00:00:00 UTC
				SetDelay(types.NewDuration(2*time.Minute)).
				AddSignature(sig2).
				AddTimelockAddress(chaintest.Chain2Selector, "0xchain2TimelockAddress").
				AddChainMetadata(chaintest.Chain2Selector, types.ChainMetadata{
					StartingOpCount:  2,
					MCMAddress:       "0xchain2McmAddress",
					AdditionalFields: json.RawMessage(`{"chain2":"value2"}`),
				}).
				AddOperation(types.BatchOperation{
					ChainSelector: chaintest.Chain2Selector,
					Transactions: []types.Transaction{
						{
							To:               "0xchain2ToAddress1",
							Data:             common.Hex2Bytes("0x2001"),
							AdditionalFields: json.RawMessage(`{"value": 0}`),
						},
						{
							To:               "0xchain2ToAddress2",
							Data:             common.Hex2Bytes("0x2002"),
							AdditionalFields: json.RawMessage{},
						},
					},
				}),
			),
			assert: func(t *testing.T, merged *TimelockProposal) {
				t.Helper()
				want := mustBuild(t, baseProposalBuilder().
					SetDescription("proposal 1\nproposal 2").
					SetValidUntil(2051222400). // 2035-01-01 00:00:00 UTC
					SetDelay(types.NewDuration(2*time.Minute)).
					SetSignatures([]types.Signature(nil)).
					AddTimelockAddress(chaintest.Chain2Selector, "0xchain2TimelockAddress").
					AddChainMetadata(chaintest.Chain2Selector, types.ChainMetadata{
						StartingOpCount:  2,
						MCMAddress:       "0xchain2McmAddress",
						AdditionalFields: json.RawMessage(`{"chain2":"value2"}`),
					}).
					AddOperation(types.BatchOperation{
						ChainSelector: chaintest.Chain2Selector,
						Transactions: []types.Transaction{
							{
								To:               "0xchain2ToAddress1",
								Data:             common.Hex2Bytes("0x2001"),
								AdditionalFields: json.RawMessage(`{"value": 0}`),
							},
							{
								To:               "0xchain2ToAddress2",
								Data:             common.Hex2Bytes("0x2002"),
								AdditionalFields: json.RawMessage{},
							},
						},
					}),
				)

				require.Equal(t, want, merged)
			},
		},
		{
			name:      "success: merge with overlapping chains",
			proposal1: mustBuild(t, baseProposalBuilder()),
			proposal2: func() *TimelockProposal {
				return mustBuild(t, baseProposalBuilder().
					SetDescription("proposal 2").
					SetValidUntil(2053987200). // 2035-02-02 00:00:00 UTC
					SetDelay(types.NewDuration(30*time.Second)).
					AddSignature(sig2).
					AddSignature(sig3).
					// chain 1
					AddTimelockAddress(chaintest.Chain1Selector, "0xchain1TimelockAddress").
					AddChainMetadata(chaintest.Chain1Selector, types.ChainMetadata{
						StartingOpCount:  11,
						MCMAddress:       "0xchain1McmAddress",
						AdditionalFields: json.RawMessage(`{"chain1.otherKey": "otherValue"}`),
					}).
					SetOperations([]types.BatchOperation{{
						ChainSelector: chaintest.Chain1Selector,
						Transactions: []types.Transaction{
							{
								To:               "0xchain1ToAddress11",
								Data:             common.Hex2Bytes("0x1011"),
								AdditionalFields: json.RawMessage(`{"value":11}`),
							},
						},
					}}).
					// chain 2
					AddTimelockAddress(chaintest.Chain2Selector, "0xchain2TimelockAddress").
					AddChainMetadata(chaintest.Chain2Selector, types.ChainMetadata{
						StartingOpCount:  2,
						MCMAddress:       "0xchain2McmAddress",
						AdditionalFields: json.RawMessage(`{"chain2":"value2"}`),
					}).
					AddOperation(types.BatchOperation{
						ChainSelector: chaintest.Chain2Selector,
						Transactions: []types.Transaction{
							{
								To:               "0xchain2ToAddress1",
								Data:             common.Hex2Bytes("0x2001"),
								AdditionalFields: json.RawMessage(`{"value":2}`),
							},
						},
					}),
				)
			}(),
			assert: func(t *testing.T, merged *TimelockProposal) {
				t.Helper()
				want := mustBuild(t, baseProposalBuilder().
					SetDescription("proposal 1\nproposal 2").
					SetValidUntil(2051222400). // 2035-01-01 00:00:00 UTC
					SetDelay(types.NewDuration(1*time.Minute)).
					SetSignatures([]types.Signature(nil)).
					// chain 1
					AddTimelockAddress(chaintest.Chain1Selector, "0xchain1TimelockAddress").
					AddChainMetadata(chaintest.Chain1Selector, types.ChainMetadata{
						StartingOpCount:  1, // lowest opcount
						MCMAddress:       "0xchain1McmAddress",
						AdditionalFields: json.RawMessage(`{"chain1":"value1","chain1.otherKey":"otherValue"}`),
					}).
					AddOperation(types.BatchOperation{
						ChainSelector: chaintest.Chain1Selector,
						Transactions: []types.Transaction{
							{
								To:               "0xchain1ToAddress11",
								Data:             common.Hex2Bytes("0x1011"),
								AdditionalFields: json.RawMessage(`{"value":11}`),
							},
						},
					}).
					// chain 2
					AddTimelockAddress(chaintest.Chain2Selector, "0xchain2TimelockAddress").
					AddChainMetadata(chaintest.Chain2Selector, types.ChainMetadata{
						StartingOpCount:  2,
						MCMAddress:       "0xchain2McmAddress",
						AdditionalFields: json.RawMessage(`{"chain2":"value2"}`),
					}).
					AddOperation(types.BatchOperation{
						ChainSelector: chaintest.Chain2Selector,
						Transactions: []types.Transaction{
							{
								To:               "0xchain2ToAddress1",
								Data:             common.Hex2Bytes("0x2001"),
								AdditionalFields: json.RawMessage(`{"value":2}`),
							},
						},
					}),
				)

				require.Equal(t, want, merged)
			},
		},
		{
			name: "success: merge with empty optional fields",
			proposal1: mustBuild(t, NewTimelockProposalBuilder().
				SetVersion("v1").
				SetAction(types.TimelockActionSchedule).
				SetValidUntil(2051222400). // 2035-01-01 00:00:00 UTC
				SetDelay(types.NewDuration(1*time.Minute)).
				AddTimelockAddress(chaintest.Chain1Selector, "0xchain1TimelockAddress").
				AddChainMetadata(chaintest.Chain1Selector, types.ChainMetadata{
					StartingOpCount: 1,
					MCMAddress:      "0xchain1McmAddress",
				}).
				AddOperation(types.BatchOperation{
					ChainSelector: chaintest.Chain1Selector,
					Transactions: []types.Transaction{{
						To:               "0xchain1ToAddress1",
						Data:             common.Hex2Bytes("0x1001"),
						AdditionalFields: json.RawMessage(""),
					}},
				}),
			),
			proposal2: mustBuild(t, NewTimelockProposalBuilder().
				SetVersion("v1").
				SetAction(types.TimelockActionSchedule).
				SetValidUntil(2051222400). // 2035-01-01 00:00:00 UTC
				SetDelay(types.NewDuration(2*time.Minute)).
				AddTimelockAddress(chaintest.Chain1Selector, "0xchain1TimelockAddress").
				AddChainMetadata(chaintest.Chain1Selector, types.ChainMetadata{
					StartingOpCount: 1,
					MCMAddress:      "0xchain1McmAddress",
				}).
				AddOperation(types.BatchOperation{
					ChainSelector: chaintest.Chain1Selector,
					Transactions: []types.Transaction{{
						To:               "0xchain1ToAddress2",
						Data:             common.Hex2Bytes("0x1002"),
						AdditionalFields: json.RawMessage(""),
					}},
				}),
			),
			assert: func(t *testing.T, merged *TimelockProposal) {
				t.Helper()
				want := mustBuild(t, NewTimelockProposalBuilder().
					SetVersion("v1").
					SetAction(types.TimelockActionSchedule).
					SetValidUntil(2051222400). // 2035-01-01 00:00:00 UTC
					SetDelay(types.NewDuration(2*time.Minute)).
					AddTimelockAddress(chaintest.Chain1Selector, "0xchain1TimelockAddress").
					AddChainMetadata(chaintest.Chain1Selector, types.ChainMetadata{
						StartingOpCount: 1,
						MCMAddress:      "0xchain1McmAddress",
					}).
					AddOperation(types.BatchOperation{
						ChainSelector: chaintest.Chain1Selector,
						Transactions: []types.Transaction{{
							To:               "0xchain1ToAddress1",
							Data:             common.Hex2Bytes("0x1001"),
							AdditionalFields: json.RawMessage(""),
						}},
					}).
					AddOperation(types.BatchOperation{
						ChainSelector: chaintest.Chain1Selector,
						Transactions: []types.Transaction{{
							To:               "0xchain1ToAddress2",
							Data:             common.Hex2Bytes("0x1002"),
							AdditionalFields: json.RawMessage(""),
						}},
					}),
				)

				require.Equal(t, want, merged)
			},
		},
		{
			name:      "success: merge salt",
			proposal1: mustBuild(t, baseProposalBuilder().SetSalt(new(common.HexToHash("0x0123456789abcdef")))),
			proposal2: mustBuild(t, baseProposalBuilder().SetSalt(new(common.HexToHash("0x9876543210fedcba")))),
			assert: func(t *testing.T, merged *TimelockProposal) {
				t.Helper()
				want := new(common.HexToHash("0x9955115599551155"))
				require.Equal(t, want, merged.SaltOverride)
			},
		},
		{
			name: "success: merge Metadata",
			proposal1: func() *TimelockProposal {
				proposal := mustBuild(t, baseProposalBuilder())
				proposal.Metadata = map[string]any{"key1": 1, "key2": "two", "key3": []any{1, 2, 3}}

				return proposal
			}(),
			proposal2: func() *TimelockProposal {
				proposal := mustBuild(t, baseProposalBuilder())
				proposal.Metadata = map[string]any{"key1": "one", "key3": []any{4, 5}, "key4": 3.0}

				return proposal
			}(),
			assert: func(t *testing.T, merged *TimelockProposal) {
				t.Helper()
				want := map[string]any{"key1": "one", "key2": "two", "key3": []any{1, 2, 3, 4, 5}, "key4": 3.0}
				require.Equal(t, want, merged.Metadata)
			},
		},
		{
			name:      "failure: different versions",
			proposal1: mustBuild(t, baseProposalBuilder()),
			proposal2: func() *TimelockProposal {
				proposal := mustBuild(t, baseProposalBuilder())
				proposal.Version = "v2"

				return proposal
			}(),
			wantErr: "cannot merge proposals with different versions",
		},
		{
			name:      "failure: different kinds",
			proposal1: mustBuild(t, baseProposalBuilder()),
			proposal2: func() *TimelockProposal {
				proposal := mustBuild(t, baseProposalBuilder())
				proposal.Kind = types.KindProposal

				return proposal
			}(),
			wantErr: "cannot merge proposals with different kinds",
		},
		{
			name:      "failure: different actions",
			proposal1: mustBuild(t, baseProposalBuilder()),
			proposal2: func() *TimelockProposal {
				proposal := mustBuild(t, baseProposalBuilder())
				proposal.Action = types.TimelockActionCancel

				return proposal
			}(),
			wantErr: "cannot merge proposals with different actions",
		},
		{
			name:      "failure: different timelock addresses for same chain",
			proposal1: mustBuild(t, baseProposalBuilder()),
			proposal2: mustBuild(t, baseProposalBuilder().
				AddTimelockAddress(chaintest.Chain1Selector, "0xdifferentAddress")),
			wantErr: "cannot merge proposals with different timelock addresses",
		},
		{
			name:      "failure: different mcm addresses for same chain",
			proposal1: mustBuild(t, baseProposalBuilder()),
			proposal2: mustBuild(t, baseProposalBuilder().
				AddChainMetadata(chaintest.Chain1Selector, types.ChainMetadata{
					StartingOpCount: 2,
					MCMAddress:      "0xotherMcmAddress",
				}),
			),
			wantErr: "cannot merge ChainMetadata with different MCMAddress",
		},
		{
			name: "failure: invalid ChainMetadata.AdditionalFields format in first proposal",
			proposal1: func() *TimelockProposal {
				proposal := mustBuild(t, baseProposalBuilder())
				metadata := proposal.ChainMetadata[chaintest.Chain1Selector]
				metadata.AdditionalFields = json.RawMessage(`invalid json proposal 1`)
				proposal.ChainMetadata[chaintest.Chain1Selector] = metadata

				return proposal
			}(),
			proposal2: mustBuild(t, baseProposalBuilder()),
			wantErr:   "failed to unmarshal AdditionalFields of ChainMetadata (invalid json proposal 1)",
		},
		{
			name:      "failure: invalid ChainMetadata.AdditionalFields format in second proposal",
			proposal1: mustBuild(t, baseProposalBuilder()),
			proposal2: func() *TimelockProposal {
				proposal := mustBuild(t, baseProposalBuilder())
				metadata := proposal.ChainMetadata[chaintest.Chain1Selector]
				metadata.AdditionalFields = json.RawMessage(`invalid json proposal 2`)
				proposal.ChainMetadata[chaintest.Chain1Selector] = metadata

				return proposal
			}(),
			wantErr: "failed to unmarshal AdditionalFields of ChainMetadata (invalid json proposal 2)",
		},
		{
			name: "failure: conflicting values in ChainMetadata.AdditionalFields",
			proposal1: func() *TimelockProposal {
				proposal := mustBuild(t, baseProposalBuilder())
				metadata := proposal.ChainMetadata[chaintest.Chain1Selector]
				metadata.AdditionalFields = json.RawMessage(`{"key":"value1"}`)
				proposal.ChainMetadata[chaintest.Chain1Selector] = metadata

				return proposal
			}(),
			proposal2: func() *TimelockProposal {
				proposal := mustBuild(t, baseProposalBuilder())
				metadata := proposal.ChainMetadata[chaintest.Chain1Selector]
				metadata.AdditionalFields = json.RawMessage(`{"key":"value2"}`)
				proposal.ChainMetadata[chaintest.Chain1Selector] = metadata

				return proposal
			}(),
			wantErr: "cannot merge ChainMetadata with different value for key \"key\" in AdditionalFields: value1 vs value2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			merged, err := tt.proposal1.Merge(t.Context(), tt.proposal2)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				tt.assert(t, merged)
			}
		})
	}
}

// ----- helpers -----

type builder[T any] interface{ Build() (T, error) }

func mustBuild[T any, B builder[T]](t *testing.T, builder B) T {
	t.Helper()
	proposal, err := builder.Build()
	require.NoError(t, err)

	return proposal
}
