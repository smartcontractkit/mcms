package mcms

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadProposal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "should return error for invalid JSON",
			input:   `{invalid json}`,
			wantErr: "invalid character 'i' looking for beginning of object",
		},
		{
			name:    "should return error for unknown proposal type",
			input:   `{"kind": "unknown_type"}`,
			wantErr: "unknown proposal type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reader := strings.NewReader(tt.input)

			proposal, err := LoadProposal(reader)

			if tt.wantErr == "" {
				require.NoError(t, err)
				assert.NotNil(t, proposal)
			} else {
				require.Error(t, err)
				assert.Nil(t, proposal)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}
