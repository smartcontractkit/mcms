package mcms

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/mcms/types"
)

func TestErrorMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err  error
		want string
	}{
		{NewEncoderNotFoundError(1), "encoder not provided for chain selector 1"},
		{NewChainMetadataNotFoundError(1), "missing metadata for chain 1"},
		{NewInconsistentConfigsError(1, 2), "inconsistent configs for chains 1 and 2"},
		{NewQuorumNotReachedError(1), "quorum not reached for chain 1"},
		{NewInvalidValidUntilError(1), "invalid valid until: 1"},
		{NewInvalidSignatureError(common.HexToAddress("0x1")), "invalid signature: received signature for address 0x0000000000000000000000000000000000000001 is not a valid signer in the MCMS proposal"},
		{NewQuorumNotReachedError(1), "quorum not reached for chain 1"},
		{&DuplicateSignersError{signer: "0x1234567890123456789012345678901234567890"}, "duplicate signer detected: 0x1234567890123456789012345678901234567890"},
	}

	for _, tt := range tests {
		got := tt.err.Error()
		if got != tt.want {
			assert.Equal(t, tt.want, tt.err.Error())
		}
	}
}

func TestInvalidSignatureAtIndexError(t *testing.T) {
	t.Parallel()

	// Test signature for testing
	sig := types.Signature{
		R: common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
		S: common.HexToHash("0xfedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321"),
		V: 27,
	}

	t.Run("with recovery error", func(t *testing.T) {
		t.Parallel()

		recoveryErr := errors.New("invalid signature format")
		err := NewInvalidSignatureAtIndexError(0, sig, common.Address{}, recoveryErr)

		expected := "signature at index 0 is invalid: failed to recover address from signature (r=0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef, s=0xfedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321, v=27): invalid signature format"
		assert.Equal(t, expected, err.Error())
		assert.Equal(t, 0, err.Index)
		assert.Equal(t, sig, err.Signature)
		assert.Equal(t, common.Address{}, err.RecoveredAddress)
		assert.Equal(t, recoveryErr, err.RecoveryError)
	})

	t.Run("with invalid signer", func(t *testing.T) {
		t.Parallel()

		recoveredAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")
		err := NewInvalidSignatureAtIndexError(2, sig, recoveredAddr, nil)

		expected := "signature at index 2 is invalid: recovered address 0x1234567890123456789012345678901234567890 is not a valid signer (signature: r=0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef, s=0xfedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321, v=27)"
		assert.Equal(t, expected, err.Error())
		assert.Equal(t, 2, err.Index)
		assert.Equal(t, sig, err.Signature)
		assert.Equal(t, recoveredAddr, err.RecoveredAddress)
		assert.NoError(t, err.RecoveryError)
	})
}
