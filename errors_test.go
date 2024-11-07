package mcms

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestErrorMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err      error
		expected string
	}{
		{NewEncoderNotFoundError(1), "encoder not provided for chain selector 1"},
		{NewChainMetadataNotFoundError(1), "missing metadata for chain 1"},
		{NewInconsistentConfigsError(1, 2), "inconsistent configs for chains 1 and 2"},
		{NewQuorumNotReachedError(1), "quorum not reached for chain 1"},
		{NewInvalidDelayError("1h"), "invalid delay: 1h"},
		{NewInvalidValidUntilError(1), "invalid valid until: 1"},
		{NewInvalidSignatureError(common.HexToAddress("0x1")), "invalid signature: received signature for address 0x0000000000000000000000000000000000000001 is not a valid signer in the MCMS proposal"},
		{NewQuorumNotReachedError(1), "quorum not reached for chain 1"},
	}

	for _, test := range tests {
		got := test.err.Error()
		if got != test.expected {
			assert.Equal(t, test.expected, test.err.Error())
		}
	}
}
