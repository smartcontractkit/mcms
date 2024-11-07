package sdkerrors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err      error
		expected string
	}{
		{NewInvalidChainIDError(1), "invalid chain ID: 1"},
		{NewTooManySignersError(1), "too many signers: 1 max number is 255"},
		{NewInvalidTimelockOperationError("invalid"), "invalid timelock operation: invalid"},
	}

	for _, test := range tests {
		got := test.err.Error()
		if got != test.expected {
			assert.Equal(t, test.expected, test.err.Error())
		}
	}
}
