package mcms_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-playground/validator/v10"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/internal/utils/safecast"
	"github.com/smartcontractkit/mcms/types"
)

var SepoliaSelector = chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector

func TestProposalBuilder(t *testing.T) {
	t.Parallel()

	// Define a fixed validUntil timestamp for consistency
	fixedValidUntil := int64(1893456000) // January 1, 2030
	fixedValidUntilCasted, err := safecast.Int64ToUint32(fixedValidUntil)
	require.NoError(t, err)
	pastValidUntilCast, err := safecast.Int64ToUint32(time.Now().Add(-24 * time.Hour).Unix())
	require.NoError(t, err)
	tests := []struct {
		name     string
		setup    func(*mcms.ProposalBuilder)
		want     *mcms.Proposal
		wantErrs []string
	}{
		{
			name: "valid Proposal",
			setup: func(b *mcms.ProposalBuilder) {
				b.SetVersion("1.0").
					SetValidUntil(fixedValidUntilCasted).
					SetDescription("Valid Proposal").
					SetOverridePreviousRoot(false).
					AddChainMetadata(types.ChainSelector(SepoliaSelector), types.ChainMetadata{StartingOpCount: 0}).
					AddTransaction(types.ChainOperation{
						ChainSelector: types.ChainSelector(SepoliaSelector),
						Operation:     types.Operation{Data: []byte{0x01}},
					})
			},
			want: &mcms.Proposal{
				BaseProposal: mcms.BaseProposal{
					Version:              "1.0",
					ValidUntil:           fixedValidUntilCasted,
					Description:          "Valid Proposal",
					OverridePreviousRoot: false,
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						types.ChainSelector(SepoliaSelector): {StartingOpCount: 0},
					},
				},
				Transactions: []types.ChainOperation{
					{
						ChainSelector: types.ChainSelector(SepoliaSelector),
						Operation:     types.Operation{Data: []byte{0x01}},
					},
				},
			},
			wantErrs: nil,
		},
		{
			name: "valid Proposal using SetTransactions",
			setup: func(b *mcms.ProposalBuilder) {
				b.SetVersion("1.0").
					SetValidUntil(fixedValidUntilCasted).
					SetDescription("Valid Proposal").
					SetOverridePreviousRoot(false).
					AddChainMetadata(types.ChainSelector(SepoliaSelector), types.ChainMetadata{StartingOpCount: 0}).
					SetTransactions([]types.ChainOperation{
						{
							ChainSelector: types.ChainSelector(SepoliaSelector),
							Operation:     types.Operation{Data: []byte{0x01}},
						},
						{
							ChainSelector: types.ChainSelector(SepoliaSelector),
							Operation:     types.Operation{Data: []byte{0x02}},
						},
					})
			},
			want: &mcms.Proposal{
				BaseProposal: mcms.BaseProposal{
					Version:              "1.0",
					ValidUntil:           fixedValidUntilCasted,
					Description:          "Valid Proposal",
					OverridePreviousRoot: false,
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						types.ChainSelector(SepoliaSelector): {StartingOpCount: 0},
					},
				},
				Transactions: []types.ChainOperation{
					{
						ChainSelector: types.ChainSelector(SepoliaSelector),
						Operation:     types.Operation{Data: []byte{0x01}},
					},
					{
						ChainSelector: types.ChainSelector(SepoliaSelector),
						Operation:     types.Operation{Data: []byte{0x02}},
					},
				},
			},
			wantErrs: nil,
		},
		{
			name: "valid Proposal with signature and set chain metadata",
			setup: func(b *mcms.ProposalBuilder) {
				b.SetVersion("1.0").
					SetValidUntil(fixedValidUntilCasted).
					SetDescription("Valid Proposal").
					SetOverridePreviousRoot(false).
					SetChainMetadata(map[types.ChainSelector]types.ChainMetadata{
						types.ChainSelector(SepoliaSelector): {StartingOpCount: 0},
					}).
					SetTransactions([]types.ChainOperation{
						{
							ChainSelector: types.ChainSelector(SepoliaSelector),
							Operation:     types.Operation{Data: []byte{0x01}},
						},
						{
							ChainSelector: types.ChainSelector(SepoliaSelector),
							Operation:     types.Operation{Data: []byte{0x02}},
						},
					}).
					AddSignature(types.Signature{
						R: common.Hash{0x01},
						S: common.Hash{0x02},
						V: 28,
					})
			},
			want: &mcms.Proposal{
				BaseProposal: mcms.BaseProposal{
					Version:              "1.0",
					ValidUntil:           fixedValidUntilCasted,
					Description:          "Valid Proposal",
					OverridePreviousRoot: false,
					Signatures: []types.Signature{{
						R: common.Hash{0x01},
						S: common.Hash{0x02},
						V: 28,
					}},
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						types.ChainSelector(SepoliaSelector): {StartingOpCount: 0},
					},
				},
				Transactions: []types.ChainOperation{
					{
						ChainSelector: types.ChainSelector(SepoliaSelector),
						Operation:     types.Operation{Data: []byte{0x01}},
					},
					{
						ChainSelector: types.ChainSelector(SepoliaSelector),
						Operation:     types.Operation{Data: []byte{0x02}},
					},
				},
			},
			wantErrs: nil,
		},
		{
			name: "Missing Version",
			setup: func(b *mcms.ProposalBuilder) {
				b.SetValidUntil(fixedValidUntilCasted).
					SetDescription("Missing Version").
					SetOverridePreviousRoot(false).
					AddChainMetadata(types.ChainSelector(SepoliaSelector), types.ChainMetadata{StartingOpCount: 0}).
					AddTransaction(types.ChainOperation{
						ChainSelector: types.ChainSelector(SepoliaSelector),
						Operation:     types.Operation{Data: []byte{0x01}},
					})
			},
			want: nil,
			wantErrs: []string{
				"Key: 'Proposal.BaseProposal.Version' Error:Field validation for 'Version' failed on the 'required' tag",
			},
		},
		{
			name: "ValidUntil in Past",
			setup: func(b *mcms.ProposalBuilder) {
				b.SetVersion("1.0").
					SetValidUntil(pastValidUntilCast).
					SetDescription("ValidUntil in Past").
					SetOverridePreviousRoot(false).
					AddChainMetadata(types.ChainSelector(SepoliaSelector), types.ChainMetadata{StartingOpCount: 0}).
					AddTransaction(types.ChainOperation{
						ChainSelector: types.ChainSelector(SepoliaSelector),
						Operation:     types.Operation{Data: []byte{0x01}},
					})
			},
			want: nil,
			wantErrs: []string{
				fmt.Sprintf("invalid valid until: %d", pastValidUntilCast),
			},
		},
		{
			name: "No Transactions",
			setup: func(b *mcms.ProposalBuilder) {
				b.SetVersion("1.0").
					SetValidUntil(fixedValidUntilCasted).
					SetDescription("No Transactions").
					SetOverridePreviousRoot(false).
					AddChainMetadata(types.ChainSelector(SepoliaSelector), types.ChainMetadata{StartingOpCount: 0})
				// No transactions added
			},
			want: nil,
			wantErrs: []string{
				"Key: 'Proposal.Transactions' Error:Field validation for 'Transactions' failed on the 'min' tag",
			},
		},
		{
			name: "Missing ChainMetadata",
			setup: func(b *mcms.ProposalBuilder) {
				b.SetVersion("1.0").
					SetValidUntil(fixedValidUntilCasted).
					SetDescription("Missing ChainMetadata").
					SetOverridePreviousRoot(false).
					// ChainMetadata is not added
					AddTransaction(types.ChainOperation{
						ChainSelector: types.ChainSelector(SepoliaSelector),
						Operation:     types.Operation{Data: []byte{0x01}},
					})
			},
			want: nil,
			wantErrs: []string{
				"Key: 'Proposal.BaseProposal.ChainMetadata' Error:Field validation for 'ChainMetadata' failed on the 'min' tag",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			builder := mcms.NewProposalBuilder()
			tt.setup(builder)

			proposal, err := builder.Build()
			if tt.wantErrs != nil {
				require.Error(t, err)
				assert.Nil(t, proposal)

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
				assert.NotNil(t, proposal)

				// Compare the built proposal to the expected proposal
				assert.Equal(t, tt.want, proposal)
			}
		})
	}
}
