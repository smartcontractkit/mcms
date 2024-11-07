package mcms_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/internal/utils/safecast"
	"github.com/smartcontractkit/mcms/types"
)

func TestTimelockProposalBuilder(t *testing.T) {
	t.Parallel()

	// Define a fixed validUntil timestamp for consistency
	fixedValidUntil := int64(1893456000) // January 1, 2030
	fixedValidUntilCasted, err := safecast.Int64ToUint32(fixedValidUntil)
	require.NoError(t, err)
	pastValidUntil := time.Now().Add(-24 * time.Hour).Unix()
	postValidUntilCasted, err := safecast.Int64ToUint32(pastValidUntil)
	require.NoError(t, err)
	futureValidUntil := time.Now().Add(24 * time.Hour).Unix()
	futureValidUntilCasted, err := safecast.Int64ToUint32(futureValidUntil)
	require.NoError(t, err)

	tests := []struct {
		name     string
		setup    func(*mcms.TimelockProposalBuilder)
		want     *mcms.TimelockProposal
		wantErrs []string
	}{
		{
			name: "valid timelock proposal",
			setup: func(b *mcms.TimelockProposalBuilder) {
				b.SetVersion("v1").
					SetValidUntil(fixedValidUntilCasted).
					SetDescription("Valid Timelock Proposal").
					SetOverridePreviousRoot(false).
					SetOperation(types.TimelockActionSchedule).
					SetDelay("24h").
					AddChainMetadata(types.ChainSelector(SepoliaSelector), types.ChainMetadata{StartingOpCount: 0}).
					SetTimelockAddress(types.ChainSelector(SepoliaSelector), "0xTimelockAddress").
					AddTransaction(types.BatchChainOperation{
						ChainSelector: types.ChainSelector(SepoliaSelector),
						Batch: []types.Operation{
							{
								Data:             []byte{0x01},
								To:               "0xContractAddress",
								AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
							},
						},
					})
			},
			want: &mcms.TimelockProposal{
				BaseProposal: mcms.BaseProposal{
					Version:              "v1",
					Kind:                 string(types.KindTimelockProposal),
					ValidUntil:           fixedValidUntilCasted,
					Description:          "Valid Timelock Proposal",
					OverridePreviousRoot: false,
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						types.ChainSelector(SepoliaSelector): {StartingOpCount: 0},
					},
				},
				Operation: types.TimelockActionSchedule,
				Delay:     "24h",
				TimelockAddresses: map[types.ChainSelector]string{
					types.ChainSelector(SepoliaSelector): "0xTimelockAddress",
				},
				Transactions: []types.BatchChainOperation{
					{
						ChainSelector: types.ChainSelector(SepoliaSelector),
						Batch: []types.Operation{
							{
								Data:             []byte{0x01},
								To:               "0xContractAddress",
								AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
							},
						},
					},
				},
			},
			wantErrs: nil,
		},
		{
			name: "valid timelock proposal with SetTransactions",
			setup: func(b *mcms.TimelockProposalBuilder) {
				b.SetVersion("v1").
					SetValidUntil(fixedValidUntilCasted).
					SetDescription("Valid Timelock Proposal").
					SetOverridePreviousRoot(false).
					SetOperation(types.TimelockActionSchedule).
					SetDelay("24h").
					AddChainMetadata(types.ChainSelector(SepoliaSelector), types.ChainMetadata{StartingOpCount: 0}).
					SetTimelockAddress(types.ChainSelector(SepoliaSelector), "0xTimelockAddress").
					SetTransactions([]types.BatchChainOperation{{
						ChainSelector: types.ChainSelector(SepoliaSelector),
						Batch: []types.Operation{
							{
								Data:             []byte{0x01},
								To:               "0xContractAddress",
								AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
							},
						},
					},
					})
			},
			want: &mcms.TimelockProposal{
				BaseProposal: mcms.BaseProposal{
					Version:              "v1",
					Kind:                 string(types.KindTimelockProposal),
					ValidUntil:           fixedValidUntilCasted,
					Description:          "Valid Timelock Proposal",
					OverridePreviousRoot: false,
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						types.ChainSelector(SepoliaSelector): {StartingOpCount: 0},
					},
				},
				Operation: types.TimelockActionSchedule,
				Delay:     "24h",
				TimelockAddresses: map[types.ChainSelector]string{
					types.ChainSelector(SepoliaSelector): "0xTimelockAddress",
				},
				Transactions: []types.BatchChainOperation{
					{
						ChainSelector: types.ChainSelector(SepoliaSelector),
						Batch: []types.Operation{
							{
								Data:             []byte{0x01},
								To:               "0xContractAddress",
								AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
							},
						},
					},
				},
			},
			wantErrs: nil,
		},
		{
			name: "missing delay on schedule operation",
			setup: func(b *mcms.TimelockProposalBuilder) {
				b.SetVersion("v1").
					SetValidUntil(futureValidUntilCasted).
					SetDescription("Missing Delay").
					SetOverridePreviousRoot(false).
					SetOperation(types.TimelockActionSchedule).
					// Missing SetDelay
					AddChainMetadata(types.ChainSelector(SepoliaSelector), types.ChainMetadata{StartingOpCount: 0}).
					SetTimelockAddress(types.ChainSelector(SepoliaSelector), "0xTimelockAddress").
					AddTransaction(types.BatchChainOperation{
						ChainSelector: types.ChainSelector(SepoliaSelector),
						Batch: []types.Operation{
							{
								Data:             []byte{0x01},
								To:               "0xContractAddress",
								AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
							},
						},
					})
			},
			want: nil,
			wantErrs: []string{
				"invalid delay: ",
			},
		},
		{
			name: "invalid delay format",
			setup: func(b *mcms.TimelockProposalBuilder) {
				b.SetVersion("v1").
					SetValidUntil(fixedValidUntilCasted).
					SetDescription("Invalid Delay Format").
					SetOverridePreviousRoot(false).
					SetOperation(types.TimelockActionSchedule).
					SetDelay("invalid_duration").
					AddChainMetadata(types.ChainSelector(SepoliaSelector), types.ChainMetadata{StartingOpCount: 0}).
					SetTimelockAddress(types.ChainSelector(SepoliaSelector), "0xTimelockAddress").
					AddTransaction(types.BatchChainOperation{
						ChainSelector: types.ChainSelector(SepoliaSelector),
						Batch: []types.Operation{
							{
								Data:             []byte{0x01},
								To:               "0xContractAddress",
								AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
							},
						},
					})
			},
			want: nil,
			wantErrs: []string{
				"invalid delay: invalid_duration",
			},
		},
		{
			name: "missing timelock address",
			setup: func(b *mcms.TimelockProposalBuilder) {
				b.SetVersion("v1").
					SetValidUntil(futureValidUntilCasted).
					SetDescription("Missing Timelock Address").
					SetOverridePreviousRoot(false).
					SetOperation(types.TimelockActionSchedule).
					SetDelay("24h").
					AddChainMetadata(types.ChainSelector(SepoliaSelector), types.ChainMetadata{StartingOpCount: 0}).
					// Missing SetTimelockAddress
					AddTransaction(types.BatchChainOperation{
						ChainSelector: types.ChainSelector(SepoliaSelector),
						Batch: []types.Operation{
							{
								Data:             []byte{0x01},
								To:               "0xContractAddress",
								AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
							},
						},
					})
			},
			want: nil,
			wantErrs: []string{
				"Key: 'TimelockProposal.TimelockAddresses' Error:Field validation for 'TimelockAddresses' failed on the 'min' tag",
			},
		},
		{
			name: "missing transactions",
			setup: func(b *mcms.TimelockProposalBuilder) {
				b.SetVersion("v1").
					SetValidUntil(futureValidUntilCasted).
					SetDescription("Missing Transactions").
					SetOverridePreviousRoot(false).
					SetOperation(types.TimelockActionSchedule).
					SetDelay("24h").
					AddChainMetadata(types.ChainSelector(SepoliaSelector), types.ChainMetadata{StartingOpCount: 0}).
					SetTimelockAddress(types.ChainSelector(SepoliaSelector), "0xTimelockAddress")
				// No transactions added
			},
			want: nil,
			wantErrs: []string{
				"Key: 'TimelockProposal.Transactions' Error:Field validation for 'Transactions' failed on the 'min' tag",
			},
		},
		{
			name: "validUntil in past",
			setup: func(b *mcms.TimelockProposalBuilder) {
				b.SetVersion("v1").
					SetValidUntil(postValidUntilCasted).
					SetDescription("ValidUntil in Past").
					SetOverridePreviousRoot(false).
					SetOperation(types.TimelockActionSchedule).
					SetDelay("24h").
					AddChainMetadata(types.ChainSelector(SepoliaSelector), types.ChainMetadata{StartingOpCount: 0}).
					SetTimelockAddress(types.ChainSelector(SepoliaSelector), "0xTimelockAddress").
					AddTransaction(types.BatchChainOperation{
						ChainSelector: types.ChainSelector(SepoliaSelector),
						Batch: []types.Operation{
							{
								Data:             []byte{0x01},
								To:               "0xContractAddress",
								AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
							},
						},
					})
			},
			want: nil,
			wantErrs: []string{
				fmt.Sprintf("invalid valid until: %d", pastValidUntil),
			},
		},
		{
			name: "invalid operation",
			setup: func(b *mcms.TimelockProposalBuilder) {
				b.SetVersion("v1").
					SetValidUntil(fixedValidUntilCasted).
					SetDescription("Invalid Operation").
					SetOverridePreviousRoot(false).
					// Set an invalid operation (not schedule, cancel, or bypass)
					SetOperation("invalid_operation").
					AddChainMetadata(types.ChainSelector(SepoliaSelector), types.ChainMetadata{StartingOpCount: 0}).
					SetTimelockAddress(types.ChainSelector(SepoliaSelector), "0xTimelockAddress").
					AddTransaction(types.BatchChainOperation{
						ChainSelector: types.ChainSelector(SepoliaSelector),
						Batch: []types.Operation{
							{
								Data: []byte{0x01},
								To:   "0xContractAddress",
							},
						},
					})
			},
			want: nil,
			wantErrs: []string{
				"Key: 'TimelockProposal.Operation' Error:Field validation for 'Operation' failed on the 'oneof' tag",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			builder := mcms.NewTimelockProposalBuilder()
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
