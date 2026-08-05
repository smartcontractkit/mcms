package mcms

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/sdk"
	evmsdk "github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

const (
	testMCMAddressPrimary = "0x0000000000000000000000000000000000000aaa"
	testMCMAddressSecond  = "0x0000000000000000000000000000000000000bbb"
)

// multiMCMChainMetadata returns Chain1 metadata with a primary and one additional MCM instance.
func multiMCMChainMetadata() types.ChainMetadata {
	return types.ChainMetadata{
		StartingOpCount: 5,
		MCMAddress:      testMCMAddressPrimary,
		AdditionalMCMs: []types.ChainMetadata{
			{StartingOpCount: 2, MCMAddress: testMCMAddressSecond},
		},
	}
}

func multiMCMOp(chainSelector types.ChainSelector, mcmAddress string) types.Operation {
	return types.Operation{
		ChainSelector: chainSelector,
		McmAddress:    mcmAddress,
		Transaction: types.Transaction{
			To:               TestAddress,
			AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
			Data:             common.Hex2Bytes("0x"),
		},
	}
}

func TestProposal_MultiMCM_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(p *Proposal)
		wantErr string
	}{
		{
			name: "success: v2 proposal with two instances and attributed ops",
			setup: func(_ *Proposal) {
				// NOP: valid proposal built by default
			},
		},
		{
			name: "failure: additional MCMs require v2",
			setup: func(p *Proposal) {
				p.Version = "v1"
				p.Operations = []types.Operation{multiMCMOp(chaintest.Chain1Selector, "")}
			},
			wantErr: "additional MCM instances require proposal version v2",
		},
		{
			name: "failure: operation mcmAddress requires v2",
			setup: func(p *Proposal) {
				p.Version = "v1"
				p.ChainMetadata = map[types.ChainSelector]types.ChainMetadata{
					chaintest.Chain1Selector: {StartingOpCount: 5, MCMAddress: testMCMAddressPrimary},
				}
				p.Operations = []types.Operation{multiMCMOp(chaintest.Chain1Selector, testMCMAddressPrimary)}
			},
			wantErr: "operation mcmAddress requires proposal version v2",
		},
		{
			name: "failure: unknown operation mcmAddress",
			setup: func(p *Proposal) {
				p.Operations = []types.Operation{multiMCMOp(chaintest.Chain1Selector, "0xunknown")}
			},
			wantErr: "does not match the chain's primary MCM or any additional MCM instance",
		},
		{
			name: "failure: duplicate instance address",
			setup: func(p *Proposal) {
				p.ChainMetadata = map[types.ChainSelector]types.ChainMetadata{
					chaintest.Chain1Selector: {
						StartingOpCount: 5,
						MCMAddress:      testMCMAddressPrimary,
						AdditionalMCMs: []types.ChainMetadata{
							{StartingOpCount: 2, MCMAddress: testMCMAddressPrimary},
						},
					},
				}
			},
			wantErr: "duplicate MCMAddress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			builder := NewProposalBuilder()
			builder.SetVersion("v2").
				SetValidUntil(2552083725).
				AddChainMetadata(chaintest.Chain1Selector, multiMCMChainMetadata()).
				AddOperation(multiMCMOp(chaintest.Chain1Selector, "")).
				AddOperation(multiMCMOp(chaintest.Chain1Selector, testMCMAddressSecond))
			give, err := builder.Build()
			require.NoError(t, err)

			tt.setup(give)

			err = give.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestProposal_MultiMCM_TransactionNonces(t *testing.T) {
	t.Parallel()

	builder := NewProposalBuilder()
	builder.SetVersion("v2").
		SetValidUntil(2552083725).
		AddChainMetadata(chaintest.Chain1Selector, multiMCMChainMetadata()).
		SetOperations([]types.Operation{
			multiMCMOp(chaintest.Chain1Selector, ""), // primary, nonce 5
			multiMCMOp(chaintest.Chain1Selector, testMCMAddressSecond), // second, nonce 2
			multiMCMOp(chaintest.Chain1Selector, ""), // primary, nonce 6
			multiMCMOp(chaintest.Chain1Selector, testMCMAddressSecond), // second, nonce 3
		})
	proposal, err := builder.Build()
	require.NoError(t, err)

	nonces, err := proposal.TransactionNonces()
	require.NoError(t, err)
	assert.Equal(t, []uint64{5, 2, 6, 3}, nonces)
}

func TestProposal_MultiMCM_MerkleTree(t *testing.T) {
	t.Parallel()

	buildProposal := func(ops []types.Operation) *Proposal {
		builder := NewProposalBuilder()
		builder.SetVersion("v2").
			SetValidUntil(2552083725).
			AddChainMetadata(chaintest.Chain1Selector, multiMCMChainMetadata()).
			SetOperations(ops)
		p, err := builder.Build()
		require.NoError(t, err)
		return p
	}

	opsPrimaryThenSecond := []types.Operation{
		multiMCMOp(chaintest.Chain1Selector, ""),
		multiMCMOp(chaintest.Chain1Selector, testMCMAddressSecond),
	}

	t.Run("success: builds tree covering both instances", func(t *testing.T) {
		t.Parallel()
		tree, err := buildProposal(opsPrimaryThenSecond).MerkleTree()
		require.NoError(t, err)
		require.NotEqual(t, common.Hash{}, tree.Root)
	})

	t.Run("root is deterministic regardless of additionalMCMs ordering", func(t *testing.T) {
		t.Parallel()
		p1 := buildProposal(opsPrimaryThenSecond)

		p2 := buildProposal(opsPrimaryThenSecond)
		md := p2.ChainMetadata[chaintest.Chain1Selector]
		md.AdditionalMCMs = append(md.AdditionalMCMs, types.ChainMetadata{
			StartingOpCount: 9, MCMAddress: "0x0000000000000000000000000000000000000ccc",
		})
		p2.ChainMetadata[chaintest.Chain1Selector] = md

		tree1, err := p1.MerkleTree()
		require.NoError(t, err)
		tree2, err := p2.MerkleTree()
		require.NoError(t, err)
		assert.NotEqual(t, tree1.Root, tree2.Root, "adding an instance must change the root")

		// Rebuilding p1's tree yields the same root
		tree1Again, err := buildProposal(opsPrimaryThenSecond).MerkleTree()
		require.NoError(t, err)
		assert.Equal(t, tree1.Root, tree1Again.Root)
	})

	t.Run("op attribution changes the root", func(t *testing.T) {
		t.Parallel()
		treePrimary, err := buildProposal([]types.Operation{
			multiMCMOp(chaintest.Chain1Selector, ""),
			multiMCMOp(chaintest.Chain1Selector, ""),
		}).MerkleTree()
		require.NoError(t, err)

		treeSecond, err := buildProposal([]types.Operation{
			multiMCMOp(chaintest.Chain1Selector, testMCMAddressSecond),
			multiMCMOp(chaintest.Chain1Selector, testMCMAddressSecond),
		}).MerkleTree()
		require.NoError(t, err)

		assert.NotEqual(t, treePrimary.Root, treeSecond.Root)
	})

	t.Run("single-MCM v1 regression: root matches pre-multi-MCM algorithm", func(t *testing.T) {
		t.Parallel()
		// Same shape as TestProposal_MerkleTree's success case, whose root was
		// generated before multi-MCM support existed.
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
		p, err := builder.Build()
		require.NoError(t, err)

		tree, err := p.MerkleTree()
		require.NoError(t, err)
		assert.Equal(t,
			common.HexToHash("0x4fdb98431759bbcab33cbd1b4034fea43ef360f11b1de4ca10fc20f8916bda19"),
			tree.Root)
	})
}

func TestProposal_MultiMCM_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	proposalJSON := `{
		"version": "v2",
		"kind": "Proposal",
		"validUntil": 2552083725,
		"chainMetadata": {
			"3379446385462418246": {
				"startingOpCount": 5,
				"mcmAddress": "0x0000000000000000000000000000000000000aaa",
				"additionalMCMs": [
					{"startingOpCount": 2, "mcmAddress": "0x0000000000000000000000000000000000000bbb"}
				]
			}
		},
		"operations": [
			{
				"chainSelector": 3379446385462418246,
				"mcmAddress": "0x0000000000000000000000000000000000000bbb",
				"transaction": {
					"to": "0xsomeaddress",
					"data": "EjM=",
					"additionalFields": {"value": 0}
				}
			}
		]
	}`

	proposal, err := NewProposal(jsonReader(proposalJSON))
	require.NoError(t, err)

	md := proposal.ChainMetadata[chaintest.Chain1Selector]
	require.Len(t, md.AdditionalMCMs, 1)
	assert.Equal(t, testMCMAddressSecond, md.AdditionalMCMs[0].MCMAddress)
	assert.Equal(t, testMCMAddressSecond, proposal.Operations[0].McmAddress)
}

func TestTimelockProposal_MultiMCM_Convert(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	chainMetadata := map[types.ChainSelector]types.ChainMetadata{
		chaintest.Chain1Selector: multiMCMChainMetadata(),
	}
	timelockAddresses := map[types.ChainSelector]string{
		chaintest.Chain1Selector: "0xtimelock",
	}
	tx := func(data string) types.Transaction {
		return types.Transaction{
			To:               "0x123",
			AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
			Data:             common.Hex2Bytes(data),
		}
	}

	proposal := TimelockProposal{
		BaseProposal: BaseProposal{
			Version:       "v2",
			Kind:          types.KindTimelockProposal,
			ValidUntil:    2552083725,
			ChainMetadata: chainMetadata,
		},
		Action:            types.TimelockActionSchedule,
		Delay:             types.MustParseDuration("1h"),
		TimelockAddresses: timelockAddresses,
		Operations: []types.BatchOperation{
			{ChainSelector: chaintest.Chain1Selector, Transactions: []types.Transaction{tx("0x1")}},
			{ChainSelector: chaintest.Chain1Selector, McmAddress: testMCMAddressSecond, Transactions: []types.Transaction{tx("0x2")}},
			{ChainSelector: chaintest.Chain1Selector, Transactions: []types.Transaction{tx("0x3")}},
		},
	}

	converters := map[types.ChainSelector]sdk.TimelockConverter{
		chaintest.Chain1Selector: &evmsdk.TimelockConverter{},
	}

	mcmsProposal, predecessors, err := proposal.Convert(ctx, converters)
	require.NoError(t, err)

	// One converted op per batch op for the EVM schedule action
	require.Len(t, mcmsProposal.Operations, 3)
	require.Len(t, predecessors, 3)

	// Converted ops preserve the batch's MCM attribution
	assert.Empty(t, mcmsProposal.Operations[0].McmAddress)
	assert.Equal(t, testMCMAddressSecond, mcmsProposal.Operations[1].McmAddress)
	assert.Empty(t, mcmsProposal.Operations[2].McmAddress)

	// Predecessors chain per instance: the second instance's first op has a zero
	// predecessor even though the primary instance already scheduled an op on this chain.
	assert.Equal(t, ZeroHash, predecessors[0])
	assert.Equal(t, ZeroHash, predecessors[1], "second instance's first op must not chain to the primary instance")
	assert.NotEqual(t, ZeroHash, predecessors[2], "primary instance's second op chains to its first")

	// The converted MCMS proposal must validate and sequence nonces per instance
	require.NoError(t, mcmsProposal.Validate())
	nonces, err := mcmsProposal.TransactionNonces()
	require.NoError(t, err)
	assert.Equal(t, []uint64{5, 2, 6}, nonces)
}

func TestTimelockProposal_TimelockAddressForOp(t *testing.T) {
	t.Parallel()

	proposal := TimelockProposal{
		BaseProposal: BaseProposal{
			ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
				chaintest.Chain1Selector: multiMCMChainMetadata(),
			},
		},
		TimelockAddresses: map[types.ChainSelector]string{
			chaintest.Chain1Selector: "0xtimelock",
		},
	}

	// Primary op (no attribution): chain timelock address
	assert.Equal(t, "0xtimelock", proposal.TimelockAddressForOp(types.BatchOperation{
		ChainSelector: chaintest.Chain1Selector,
	}))

	// Explicit primary attribution: still the chain timelock address
	assert.Equal(t, "0xtimelock", proposal.TimelockAddressForOp(types.BatchOperation{
		ChainSelector: chaintest.Chain1Selector,
		McmAddress:    testMCMAddressPrimary,
	}))

	// Additional instance: the instance itself is the timelock (Canton model)
	assert.Equal(t, testMCMAddressSecond, proposal.TimelockAddressForOp(types.BatchOperation{
		ChainSelector: chaintest.Chain1Selector,
		McmAddress:    testMCMAddressSecond,
	}))
}

func TestTimelockProposal_Merge_MultiMCM(t *testing.T) {
	t.Parallel()

	newProposal := func(md types.ChainMetadata) *TimelockProposal {
		return &TimelockProposal{
			BaseProposal: BaseProposal{
				Version:    "v2",
				Kind:       types.KindTimelockProposal,
				ValidUntil: 2552083725,
				ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
					chaintest.Chain1Selector: md,
				},
			},
			Action: types.TimelockActionSchedule,
			TimelockAddresses: map[types.ChainSelector]string{
				chaintest.Chain1Selector: "0xtimelock",
			},
		}
	}

	t.Run("union when primaries cross-reference additionals", func(t *testing.T) {
		t.Parallel()
		a := newProposal(types.ChainMetadata{
			StartingOpCount: 5,
			MCMAddress:      testMCMAddressPrimary,
			AdditionalMCMs:  []types.ChainMetadata{{StartingOpCount: 2, MCMAddress: testMCMAddressSecond}},
		})
		b := newProposal(types.ChainMetadata{
			StartingOpCount: 3,
			MCMAddress:      testMCMAddressSecond,
			AdditionalMCMs:  []types.ChainMetadata{{StartingOpCount: 7, MCMAddress: testMCMAddressPrimary}},
		})

		merged, err := a.Merge(context.Background(), b)
		require.NoError(t, err)

		md := merged.ChainMetadata[chaintest.Chain1Selector]
		// Deterministic primary: lexicographically smallest address
		assert.Equal(t, testMCMAddressPrimary, md.MCMAddress)
		assert.Equal(t, uint64(5), md.StartingOpCount) // min(5, 7) for the primary instance
		require.Len(t, md.AdditionalMCMs, 1)
		assert.Equal(t, testMCMAddressSecond, md.AdditionalMCMs[0].MCMAddress)
		assert.Equal(t, uint64(2), md.AdditionalMCMs[0].StartingOpCount) // min(2, 3)
	})

	t.Run("error on truly disjoint instance sets", func(t *testing.T) {
		t.Parallel()
		a := newProposal(types.ChainMetadata{MCMAddress: testMCMAddressPrimary})
		b := newProposal(types.ChainMetadata{MCMAddress: testMCMAddressSecond})

		_, err := a.Merge(context.Background(), b)
		require.ErrorContains(t, err, "cannot merge ChainMetadata with different MCMAddress")
	})
}

// jsonReader is a small helper to keep test JSON readable.
func jsonReader(s string) *strings.Reader {
	return strings.NewReader(s)
}
