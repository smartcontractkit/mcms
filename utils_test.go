package mcms

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadProposal(t *testing.T) {
	t.Parallel()
	t.Run("should return error for invalid JSON", func(t *testing.T) {
		input := `{invalid json}`
		reader := strings.NewReader(input)

		proposal, err := LoadProposal(reader)

		assert.Error(t, err)
		assert.Nil(t, proposal)
	})

	t.Run("should return error for unknown proposal type", func(t *testing.T) {
		input := `{"kind": "unknown_type"}`
		reader := strings.NewReader(input)

		proposal, err := LoadProposal(reader)

		assert.Error(t, err)
		assert.Nil(t, proposal)
		assert.Equal(t, "unknown proposal type", err.Error())
	})
}
