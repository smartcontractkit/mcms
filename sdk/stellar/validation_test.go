package stellar

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
)

func TestValidateChainMetadata(t *testing.T) {
	t.Parallel()

	validAddr := "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"

	tests := []struct {
		name     string
		metadata types.ChainMetadata
		wantErr  string
	}{
		{
			name: "valid hex contract id",
			metadata: types.ChainMetadata{
				MCMAddress:      validAddr,
				StartingOpCount: 0,
			},
		},
		{
			name: "valid strkey",
			metadata: types.ChainMetadata{
				MCMAddress:      "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA",
				StartingOpCount: 0,
			},
		},
		{
			name: "empty mcm address",
			metadata: types.ChainMetadata{
				MCMAddress: "",
			},
			wantErr: "mcm address is required",
		},
		{
			name: "invalid address",
			metadata: types.ChainMetadata{
				MCMAddress: "not-an-address",
			},
			wantErr: "mcmAddress:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateChainMetadata(tt.metadata)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateTimelockChainMetadata(t *testing.T) {
	t.Parallel()

	const (
		validMCM     = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"
		validAccount = "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H"
	)

	meta := func(raw string) types.ChainMetadata {
		return types.ChainMetadata{
			StartingOpCount:  1,
			MCMAddress:       validMCM,
			AdditionalFields: json.RawMessage(raw),
		}
	}

	fullSchedule := fmt.Sprintf(`{
		"timelockExecutor": %q,
		"timelockProposer": %q
	}`, validAccount, validAccount)

	tests := []struct {
		name    string
		meta    types.ChainMetadata
		action  types.TimelockAction
		wantErr string
	}{
		{
			name:    "schedule valid",
			meta:    meta(fullSchedule),
			action:  types.TimelockActionSchedule,
			wantErr: "",
		},
		{
			name:    "missing additionalFields",
			meta:    types.ChainMetadata{StartingOpCount: 1, MCMAddress: validMCM},
			action:  types.TimelockActionSchedule,
			wantErr: "additionalFields is required",
		},
		{
			name:    "invalid additionalFields json",
			meta:    meta(`{`),
			action:  types.TimelockActionSchedule,
			wantErr: "additionalFields:",
		},
		{
			name:    "schedule missing executor",
			meta:    meta(fmt.Sprintf(`{"timelockProposer": %q}`, validAccount)),
			action:  types.TimelockActionSchedule,
			wantErr: "timelockExecutor is required",
		},
		{
			name:    "schedule missing proposer",
			meta:    meta(fmt.Sprintf(`{"timelockExecutor": %q}`, validAccount)),
			action:  types.TimelockActionSchedule,
			wantErr: "timelockProposer is required",
		},
		{
			name: "cancel valid",
			meta: meta(fmt.Sprintf(`{
				"timelockExecutor": %q,
				"timelockCanceller": %q
			}`, validAccount, validAccount)),
			action: types.TimelockActionCancel,
		},
		{
			name:    "cancel missing canceller",
			meta:    meta(fmt.Sprintf(`{"timelockExecutor": %q}`, validAccount)),
			action:  types.TimelockActionCancel,
			wantErr: "timelockCanceller is required",
		},
		{
			name: "bypass valid",
			meta: meta(fmt.Sprintf(`{
				"timelockExecutor": %q,
				"timelockBypasser": %q
			}`, validAccount, validAccount)),
			action: types.TimelockActionBypass,
		},
		{
			name:    "bypass missing bypasser",
			meta:    meta(fmt.Sprintf(`{"timelockExecutor": %q}`, validAccount)),
			action:  types.TimelockActionBypass,
			wantErr: "timelockBypasser is required",
		},
		{
			name:    "invalid proposer strkey",
			meta:    meta(fmt.Sprintf(`{"timelockExecutor": %q, "timelockProposer": "not-an-address"}`, validAccount)),
			action:  types.TimelockActionSchedule,
			wantErr: "timelockProposer:",
		},
		{
			name: "optional admin invalid",
			meta: meta(fmt.Sprintf(`{
				"timelockExecutor": %q,
				"timelockProposer": %q,
				"timelockAdmin": "bad"
			}`, validAccount, validAccount)),
			action:  types.TimelockActionSchedule,
			wantErr: "timelockAdmin:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateTimelockChainMetadata(tt.meta, tt.action)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateAdditionalFields(t *testing.T) {
	t.Parallel()

	valid64 := `{"value":"0x` + strings.Repeat("00", 32) + `"}`

	tests := []struct {
		name   string
		raw    json.RawMessage
		wantOK bool
	}{
		{name: "nil", raw: nil, wantOK: true},
		{name: "empty object", raw: json.RawMessage(`{}`), wantOK: true},
		{name: "valid value word", raw: json.RawMessage(valid64), wantOK: true},
		{name: "invalid json", raw: json.RawMessage(`{`), wantOK: false},
		{name: "value wrong length", raw: json.RawMessage(`{"value":"0x01"}`), wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateAdditionalFields(tt.raw)
			if tt.wantOK {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
