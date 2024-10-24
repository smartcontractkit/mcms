package evm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/mcms/internal/core/config"
	"github.com/smartcontractkit/mcms/internal/evm/bindings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigFromRaw(t *testing.T) {
	t.Parallel()

	rawConfig := bindings.ManyChainMultiSigConfig{
		GroupQuorums: [32]uint8{1, 1},
		GroupParents: [32]uint8{0, 0},
		Signers: []bindings.ManyChainMultiSigSigner{
			{Addr: common.HexToAddress("0x1"), Group: 0},
			{Addr: common.HexToAddress("0x2"), Group: 1},
		},
	}
	configurator := EVMConfigurator{}
	config, err := configurator.ToConfig(rawConfig)

	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, uint8(1), config.Quorum)
	assert.Equal(t, []common.Address{common.HexToAddress("0x1")}, config.Signers)
	assert.Equal(t, uint8(1), config.GroupSigners[0].Quorum)
	assert.Equal(t, []common.Address{common.HexToAddress("0x2")}, config.GroupSigners[0].Signers)
}

func TestToRawConfig(t *testing.T) {
	t.Parallel()

	signers := []common.Address{common.HexToAddress("0x1"), common.HexToAddress("0x2")}
	groupSigners := []config.Config{
		{Quorum: 1, Signers: []common.Address{common.HexToAddress("0x3")}},
	}
	config, err := config.NewConfig(1, signers, groupSigners)
	assert.NotNil(t, config)
	require.NoError(t, err)

	configurator := EVMConfigurator{}
	rawConfig, err := configurator.SetConfigInputs(*config)
	require.NoError(t, err)

	assert.Equal(t, [32]uint8{1, 1}, rawConfig.GroupQuorums)
	assert.Equal(t, [32]uint8{0, 0}, rawConfig.GroupParents)
	assert.Equal(t, common.HexToAddress("0x1"), rawConfig.Signers[0].Addr)
	assert.Equal(t, common.HexToAddress("0x2"), rawConfig.Signers[1].Addr)
	assert.Equal(t, common.HexToAddress("0x3"), rawConfig.Signers[2].Addr)
}

// Test case 0: Valid configuration with no signers or groups
// Configuration:
// Quorum: 0
// Signers: []
// Group signers: []
func TestExtractSetConfigInputs_EmptyConfig(t *testing.T) {
	t.Parallel()

	config, err := config.NewConfig(0, []common.Address{}, []config.Config{})
	assert.Nil(t, config)
	require.Error(t, err)
	assert.Equal(t, "invalid MCMS config: Quorum must be greater than 0", err.Error())
}

// Test case 1: Valid configuration with some root signers and some groups
// Configuration:
// Quorum: 2
// Signers: [0x1, 0x2]
//
//	Group signers: [{
//		Quorum: 1
//		Signers: [0x3]
//		Group signers: []
//	}]
func TestExtractSetConfigInputs(t *testing.T) {
	t.Parallel()

	signers := []common.Address{common.HexToAddress("0x1"), common.HexToAddress("0x2")}
	groupSigners := []config.Config{
		{Quorum: 1, Signers: []common.Address{common.HexToAddress("0x3")}},
	}
	config, err := config.NewConfig(2, signers, groupSigners)
	assert.NotNil(t, config)
	require.NoError(t, err)

	groupQuorums, groupParents, signerAddresses, signerGroups, err := ExtractSetConfigInputs(config)
	require.NoError(t, err)
	assert.Equal(t, [32]uint8{2, 1}, groupQuorums)
	assert.Equal(t, [32]uint8{0, 0}, groupParents)
	assert.Equal(t, []common.Address{common.HexToAddress("0x1"), common.HexToAddress("0x2"), common.HexToAddress("0x3")}, signerAddresses)
	assert.Equal(t, []uint8{0, 0, 1}, signerGroups)
}

// Test case 2: Valid configuration with only root signers
// Configuration:
// Quorum: 1
// Signers: [0x1, 0x2]
// Group signers: []
func TestExtractSetConfigInputs_OnlyRootSigners(t *testing.T) {
	t.Parallel()

	signers := []common.Address{common.HexToAddress("0x1"), common.HexToAddress("0x2")}
	config, err := config.NewConfig(1, signers, []config.Config{})
	assert.NotNil(t, config)
	require.NoError(t, err)

	groupQuorums, groupParents, signerAddresses, signerGroups, err := ExtractSetConfigInputs(config)

	require.NoError(t, err)
	assert.Equal(t, [32]uint8{1, 0}, groupQuorums)
	assert.Equal(t, [32]uint8{0, 0}, groupParents)
	assert.Equal(t, []common.Address{common.HexToAddress("0x1"), common.HexToAddress("0x2")}, signerAddresses)
	assert.Equal(t, []uint8{0, 0}, signerGroups)
}

// Test case 3: Valid configuration with only groups
// Configuration:
// Quorum: 1
// Signers: []
//
//	Group signers: [{
//		 Quorum: 1
//		 Signers: [0x3]
//		 Group signers: []
//	},
//
//	{
//	  Quorum: 1
//	  Signers: [0x4]
//	  Group signers: []
//	},
//
//	{
//		 Quorum: 1
//		 Signers: [0x5]
//		 Group signers: []
//	}]
func TestExtractSetConfigInputs_OnlyGroups(t *testing.T) {
	t.Parallel()

	groupSigners := []config.Config{
		{Quorum: 1, Signers: []common.Address{common.HexToAddress("0x3")}},
		{Quorum: 1, Signers: []common.Address{common.HexToAddress("0x4")}},
		{Quorum: 1, Signers: []common.Address{common.HexToAddress("0x5")}},
	}
	config, err := config.NewConfig(2, []common.Address{}, groupSigners)
	assert.NotNil(t, config)
	require.NoError(t, err)

	groupQuorums, groupParents, signerAddresses, signerGroups, err := ExtractSetConfigInputs(config)

	require.NoError(t, err)
	assert.Equal(t, [32]uint8{2, 1, 1, 1}, groupQuorums)
	assert.Equal(t, [32]uint8{0, 0, 0, 0}, groupParents)
	assert.Equal(t, []common.Address{common.HexToAddress("0x3"), common.HexToAddress("0x4"), common.HexToAddress("0x5")}, signerAddresses)
	assert.Equal(t, []uint8{1, 2, 3}, signerGroups)
}

// Test case 4: Valid configuration with nested signers and groups
// Configuration:
// Quorum: 2
// Signers: [0x1, 0x2]
//
//		Group signers: [{
//			Quorum: 1
//			Signers: [0x3]
//			Group signers: [{
//				Quorum: 1
//				Signers: [0x4]
//				Group signers: []
//			}]
//		},
//	 {
//			Quorum: 1
//			Signers: [0x5]
//			Group signers: []
//		}]
func TestExtractSetConfigInputs_NestedSignersAndGroups(t *testing.T) {
	t.Parallel()

	signers := []common.Address{common.HexToAddress("0x1"), common.HexToAddress("0x2")}
	groupSigners := []config.Config{
		{Quorum: 1, Signers: []common.Address{common.HexToAddress("0x3")}, GroupSigners: []config.Config{
			{Quorum: 1, Signers: []common.Address{common.HexToAddress("0x4")}},
		}},
		{Quorum: 1, Signers: []common.Address{common.HexToAddress("0x5")}},
	}
	config, err := config.NewConfig(2, signers, groupSigners)
	assert.NotNil(t, config)
	require.NoError(t, err)

	groupQuorums, groupParents, signerAddresses, signerGroups, err := ExtractSetConfigInputs(config)

	require.NoError(t, err)
	assert.Equal(t, [32]uint8{2, 1, 1, 1}, groupQuorums)
	assert.Equal(t, [32]uint8{0, 0, 1, 0}, groupParents)
	assert.Equal(t, []common.Address{common.HexToAddress("0x1"), common.HexToAddress("0x2"), common.HexToAddress("0x3"), common.HexToAddress("0x4"), common.HexToAddress("0x5")}, signerAddresses)
	assert.Equal(t, []uint8{0, 0, 1, 2, 3}, signerGroups)
}

// Test case 5: Valid configuration with unsorted signers and groups
// Configuration:
// Quorum: 2
// Signers: [0x2, 0x1]
//
//		Group signers: [{
//			Quorum: 1
//			Signers: [0x3]
//			Group signers: [{
//				Quorum: 1
//				Signers: [0x4]
//				Group signers: []
//			}]
//		},
//	 	{
//			Quorum: 1
//			Signers: [0x5]
//			Group signers: []
//		}]
func TestExtractSetConfigInputs_UnsortedSignersAndGroups(t *testing.T) {
	t.Parallel()

	signers := []common.Address{common.HexToAddress("0x2"), common.HexToAddress("0x1")}
	groupSigners := []config.Config{
		{Quorum: 1, Signers: []common.Address{common.HexToAddress("0x3")}, GroupSigners: []config.Config{
			{Quorum: 1, Signers: []common.Address{common.HexToAddress("0x4")}},
		}},
		{Quorum: 1, Signers: []common.Address{common.HexToAddress("0x5")}},
	}
	config, err := config.NewConfig(2, signers, groupSigners)
	assert.NotNil(t, config)
	require.NoError(t, err)

	groupQuorums, groupParents, signerAddresses, signerGroups, err := ExtractSetConfigInputs(config)

	require.NoError(t, err)
	assert.Equal(t, [32]uint8{2, 1, 1, 1}, groupQuorums)
	assert.Equal(t, [32]uint8{0, 0, 1, 0}, groupParents)
	assert.Equal(t, []common.Address{common.HexToAddress("0x1"), common.HexToAddress("0x2"), common.HexToAddress("0x3"), common.HexToAddress("0x4"), common.HexToAddress("0x5")}, signerAddresses)
	assert.Equal(t, []uint8{0, 0, 1, 2, 3}, signerGroups)
}
