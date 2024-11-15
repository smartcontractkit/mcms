package mcms_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/types"
)

func TestTimelockProposalBuilder(t *testing.T) {
	t.Parallel()

	// Define a fixed validUntil timestamp for consistency
	validUntil := uint32(1893456000) // January 1, 2030

	tx1 := types.Transaction{
		Data:             []byte{0x01},
		To:               "0xContractAddress",
		AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
	}

	tests := []struct {
		name    string
		setup   func(*mcms.TimelockProposalBuilder)
		want    *mcms.TimelockProposal
		wantErr bool
	}{
		{
			name: "valid timelock proposal with setters",
			setup: func(b *mcms.TimelockProposalBuilder) {
				b.SetVersion("v1").
					SetValidUntil(validUntil).
					SetDescription("Valid Timelock Proposal").
					SetOverridePreviousRoot(false).
					SetAction(types.TimelockActionSchedule).
					SetDelay(types.MustParseDuration("24h")).
					SetChainMetadata(map[types.ChainSelector]types.ChainMetadata{
						chaintest.Chain2Selector: {StartingOpCount: 0},
					}).
					SetTimelockAddresses(map[types.ChainSelector]string{
						chaintest.Chain2Selector: "0xTimelockAddress",
					}).
					SetOperations([]types.BatchOperation{{
						ChainSelector: chaintest.Chain2Selector,
						Transactions:  []types.Transaction{tx1},
					},
					})
			},
			want: &mcms.TimelockProposal{
				BaseProposal: mcms.BaseProposal{
					Version:              "v1",
					Kind:                 types.KindTimelockProposal,
					ValidUntil:           validUntil,
					Description:          "Valid Timelock Proposal",
					OverridePreviousRoot: false,
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						chaintest.Chain2Selector: {StartingOpCount: 0},
					},
				},
				Action: types.TimelockActionSchedule,
				Delay:  types.MustParseDuration("24h"),
				TimelockAddresses: map[types.ChainSelector]string{
					chaintest.Chain2Selector: "0xTimelockAddress",
				},
				Operations: []types.BatchOperation{
					{
						ChainSelector: chaintest.Chain2Selector,
						Transactions:  []types.Transaction{tx1},
					},
				},
			},
		},
		{
			name: "valid timelock proposal with adders",
			setup: func(b *mcms.TimelockProposalBuilder) {
				b.SetVersion("v1").
					SetValidUntil(validUntil).
					SetDescription("Valid Timelock Proposal").
					SetOverridePreviousRoot(false).
					SetAction(types.TimelockActionSchedule).
					SetDelay(types.MustParseDuration("24h")).
					AddChainMetadata(chaintest.Chain2Selector, types.ChainMetadata{StartingOpCount: 0}).
					AddTimelockAddress(chaintest.Chain2Selector, "0xTimelockAddress").
					AddOperation(types.BatchOperation{
						ChainSelector: chaintest.Chain2Selector,
						Transactions:  []types.Transaction{tx1},
					})
			},
			want: &mcms.TimelockProposal{
				BaseProposal: mcms.BaseProposal{
					Version:              "v1",
					Kind:                 types.KindTimelockProposal,
					ValidUntil:           validUntil,
					Description:          "Valid Timelock Proposal",
					OverridePreviousRoot: false,
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						chaintest.Chain2Selector: {StartingOpCount: 0},
					},
				},
				Action: types.TimelockActionSchedule,
				Delay:  types.MustParseDuration("24h"),
				TimelockAddresses: map[types.ChainSelector]string{
					chaintest.Chain2Selector: "0xTimelockAddress",
				},
				Operations: []types.BatchOperation{
					{
						ChainSelector: chaintest.Chain2Selector,
						Transactions:  []types.Transaction{tx1},
					},
				},
			},
		},
		{
			name:    "validation fails: missing delay",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			builder := mcms.NewTimelockProposalBuilder()

			if tt.setup != nil {
				tt.setup(builder)
			}

			got, err := builder.Build()
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
