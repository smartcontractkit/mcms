package mcms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_EncoderNotFoundError_Error(t *testing.T) {
	t.Parallel()

	err := NewEncoderNotFoundError(1)

	assert.EqualError(t, err, "encoder not provided for chain selector 1")
}

func Test_ChainMetadataNotFoundError_Error(t *testing.T) {
	t.Parallel()

	err := NewChainMetadataNotFoundError(1)

	assert.EqualError(t, err, "missing metadata for chain 1")
}

func Test_InconsistentConfigsError_Error(t *testing.T) {
	t.Parallel()

	err := NewInconsistentConfigsError(1, 2)

	assert.EqualError(t, err, "inconsistent configs for chains 1 and 2")
}

func Test_QuorumNotReachedError_Error(t *testing.T) {
	t.Parallel()

	err := NewQuorumNotReachedError(1)

	assert.EqualError(t, err, "quorum not reached for chain 1")
}
