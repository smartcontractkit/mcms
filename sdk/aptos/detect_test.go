package aptos

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/mcms/types"
)

func TestMCMSTypeFromOperations(t *testing.T) {
	t.Parallel()

	chainA := types.ChainSelector(100)
	chainB := types.ChainSelector(200)

	mustMarshal := func(af AdditionalFields) json.RawMessage {
		b, err := json.Marshal(af)
		if err != nil {
			t.Fatal(err)
		}

		return b
	}

	tests := []struct {
		name     string
		ops      []types.BatchOperation
		cs       types.ChainSelector
		expected MCMSType
	}{
		{
			name:     "empty operations",
			ops:      nil,
			cs:       chainA,
			expected: MCMSTypeRegular,
		},
		{
			name: "standard mcms",
			ops: []types.BatchOperation{
				{
					ChainSelector: chainA,
					Transactions: []types.Transaction{
						{AdditionalFields: mustMarshal(AdditionalFields{PackageName: "mcms", ModuleName: "mcms", Function: "accept_ownership"})},
					},
				},
			},
			cs:       chainA,
			expected: MCMSTypeRegular,
		},
		{
			name: "curse_mcms detected",
			ops: []types.BatchOperation{
				{
					ChainSelector: chainA,
					Transactions: []types.Transaction{
						{AdditionalFields: mustMarshal(AdditionalFields{PackageName: "curse_mcms", ModuleName: "curse_mcms_account", Function: "accept_ownership"})},
					},
				},
			},
			cs:       chainA,
			expected: MCMSTypeCurse,
		},
		{
			name: "different chain selector not matched",
			ops: []types.BatchOperation{
				{
					ChainSelector: chainB,
					Transactions: []types.Transaction{
						{AdditionalFields: mustMarshal(AdditionalFields{PackageName: "curse_mcms", ModuleName: "curse_mcms", Function: "execute"})},
					},
				},
			},
			cs:       chainA,
			expected: MCMSTypeRegular,
		},
		{
			name: "mixed transactions — one curse_mcms is enough",
			ops: []types.BatchOperation{
				{
					ChainSelector: chainA,
					Transactions: []types.Transaction{
						{AdditionalFields: mustMarshal(AdditionalFields{PackageName: "mcms", ModuleName: "mcms_deployer", Function: "stage_code_chunk"})},
						{AdditionalFields: mustMarshal(AdditionalFields{PackageName: "curse_mcms", ModuleName: "curse_mcms", Function: "timelock_update_min_delay"})},
					},
				},
			},
			cs:       chainA,
			expected: MCMSTypeCurse,
		},
		{
			name: "invalid additional fields ignored",
			ops: []types.BatchOperation{
				{
					ChainSelector: chainA,
					Transactions: []types.Transaction{
						{AdditionalFields: json.RawMessage(`{invalid}`)},
					},
				},
			},
			cs:       chainA,
			expected: MCMSTypeRegular,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := MCMSTypeFromOperations(tt.ops, tt.cs)
			assert.Equal(t, tt.expected, got)
		})
	}
}
