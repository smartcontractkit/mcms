package config

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	t.Parallel()

	signers := []common.Address{common.HexToAddress("0x1"), common.HexToAddress("0x2")}
	groupSigners := []Config{
		{Quorum: 1, Signers: []common.Address{common.HexToAddress("0x3")}},
	}
	config, err := NewConfig(1, signers, groupSigners)

	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, uint8(1), config.Quorum)
	assert.Equal(t, signers, config.Signers)
	assert.Equal(t, groupSigners, config.GroupSigners)
}

func TestValidate_Success(t *testing.T) {
	t.Parallel()

	// Test case 1: Valid configuration
	config, err := NewConfig(2, []common.Address{common.HexToAddress("0x1"), common.HexToAddress("0x2")}, []Config{})
	assert.NotNil(t, config)
	require.NoError(t, err)
}

func TestValidate_InvalidQuorum(t *testing.T) {
	t.Parallel()

	// Test case 2: Quorum is 0
	config, err := NewConfig(0, []common.Address{common.HexToAddress("0x1"), common.HexToAddress("0x2")}, []Config{})
	assert.Nil(t, config)
	require.Error(t, err)
	assert.Equal(t, "invalid MCMS config: Quorum must be greater than 0", err.Error())
}

func TestValidate_InvalidSigners(t *testing.T) {
	t.Parallel()

	// Test case 3: No signers or groups
	config, err := NewConfig(2, []common.Address{}, []Config{})
	assert.Nil(t, config)
	require.Error(t, err)
	assert.Equal(t, "invalid MCMS config: Config must have at least one signer or group", err.Error())
}

func TestValidate_InvalidQuorumCount(t *testing.T) {
	t.Parallel()

	// Test case 4: Quorum is greater than the number of signers and groups
	config, err := NewConfig(3, []common.Address{common.HexToAddress("0x1"), common.HexToAddress("0x2")}, []Config{})
	assert.Nil(t, config)
	require.Error(t, err)
	assert.Equal(t, "invalid MCMS config: Quorum must be less than or equal to the number of signers and groups", err.Error())
}

func TestValidate_InvalidGroupSigner(t *testing.T) {
	t.Parallel()

	// Test case 5: Invalid group signer
	config, err := NewConfig(2, []common.Address{common.HexToAddress("0x1"), common.HexToAddress("0x2")}, []Config{
		{Quorum: 0, Signers: []common.Address{}},
	})
	assert.Nil(t, config)
	require.Error(t, err)
	assert.Equal(t, "invalid MCMS config: Quorum must be greater than 0", err.Error())
}

func TestConfigEquals_Success(t *testing.T) {
	t.Parallel()

	signers := []common.Address{common.HexToAddress("0x1"), common.HexToAddress("0x2")}
	groupSigners := []Config{
		{Quorum: 1, Signers: []common.Address{common.HexToAddress("0x3")}},
	}
	config1, err := NewConfig(2, signers, groupSigners)
	require.NoError(t, err)
	assert.NotNil(t, config1)

	config2, err := NewConfig(2, signers, groupSigners)
	require.NoError(t, err)
	assert.NotNil(t, config2)

	assert.True(t, config1.Equals(config2))
}
func TestConfigEquals_Failure_MismatchingQuorum(t *testing.T) {
	t.Parallel()

	signers := []common.Address{common.HexToAddress("0x1"), common.HexToAddress("0x2")}
	groupSigners := []Config{
		{Quorum: 1, Signers: []common.Address{common.HexToAddress("0x3")}},
	}
	config1, err := NewConfig(2, signers, groupSigners)
	require.NoError(t, err)
	assert.NotNil(t, config1)

	config2, err := NewConfig(1, signers, groupSigners)
	require.NoError(t, err)
	assert.NotNil(t, config2)

	assert.False(t, config1.Equals(config2))
}

func TestConfigEquals_Failure_MismatchingSigners(t *testing.T) {
	t.Parallel()

	signers1 := []common.Address{common.HexToAddress("0x1"), common.HexToAddress("0x2")}
	signers2 := []common.Address{common.HexToAddress("0x1")}
	groupSigners := []Config{
		{Quorum: 1, Signers: []common.Address{common.HexToAddress("0x3")}},
	}
	config1, err := NewConfig(2, signers1, groupSigners)
	assert.NotNil(t, config1)
	require.NoError(t, err)

	config2, err := NewConfig(2, signers2, groupSigners)
	assert.NotNil(t, config2)
	require.NoError(t, err)

	assert.False(t, config1.Equals(config2))
}

func TestConfigEquals_Failure_MismatchingGroupSigners(t *testing.T) {
	t.Parallel()

	signers := []common.Address{common.HexToAddress("0x1"), common.HexToAddress("0x2")}
	groupSigners1 := []Config{
		{Quorum: 1, Signers: []common.Address{common.HexToAddress("0x3")}},
	}
	groupSigners2 := []Config{
		{Quorum: 1, Signers: []common.Address{common.HexToAddress("0x4")}},
	}
	config1, err := NewConfig(2, signers, groupSigners1)
	assert.NotNil(t, config1)
	require.NoError(t, err)

	config2, err := NewConfig(2, signers, groupSigners2)
	assert.NotNil(t, config2)
	require.NoError(t, err)

	assert.False(t, config1.Equals(config2))
}
