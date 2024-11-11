package sdkerrors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err  error
		want string
	}{
		{NewInvalidChainIDError(1), "invalid chain ID: 1"},
		{NewTooManySignersError(1), "too many signers: 1 max number is 255"},
		{NewInvalidTimelockOperationError("invalid"), "invalid timelock operation: invalid"},
	}

	for _, tt := range tests {
		got := tt.err.Error()
		if got != tt.want {
			assert.Equal(t, tt.want, tt.err.Error())
		}
	}
}
