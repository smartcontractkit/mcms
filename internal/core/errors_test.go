package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnknownChainSelectorFamilyError(t *testing.T) {
	t.Parallel()

	// Input values
	selector := uint64(12345)
	family := "UnknownFamily"

	// Create the error
	err := NewUnknownChainSelectorFamilyError(selector, family)

	// Assert that the error is correctly initialized
	assert.NotNil(t, err)
	assert.Equal(t, selector, err.ChainSelector)
	assert.Equal(t, family, err.ChainFamily)

	// Assert that the error message is formatted correctly
	expectedErrorMessage := "unknown chain selector family: 12345 with family UnknownFamily. Supported families are [evm solana]"
	assert.EqualError(t, err, expectedErrorMessage)
}
