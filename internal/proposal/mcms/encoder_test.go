package mcms

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
)

func TestEncoder_EVM_NoSim(t *testing.T) {
	t.Parallel()

	chainSelector := chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector
	encoder, err := NewEncoder(
		mcms.ChainSelector(chainSelector),
		5,
		false,
		false,
	)
	require.NoError(t, err)

	hash, err := encoder.HashMetadata(mcms.ChainMetadata{
		StartingOpCount: 0,
		MCMAddress:      common.HexToAddress("0x1").Hex(),
	})
	require.NoError(t, err)
	assert.Equal(t, "0x44dd7d0176cfe0066ac303c0a98a421c8dd09c83dd7efb42a31ba56ee95cc9a5", hash.Hex())
}

func TestEncoder_EVM_Sim(t *testing.T) {
	t.Parallel()

	chainSelector := chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector
	encoder, err := NewEncoder(
		mcms.ChainSelector(chainSelector),
		5,
		false,
		true,
	)
	require.NoError(t, err)

	hash, err := encoder.HashMetadata(mcms.ChainMetadata{
		StartingOpCount: 0,
		MCMAddress:      common.HexToAddress("0x1").Hex(),
	})
	require.NoError(t, err)
	assert.Equal(t, "0x02a34863f6c89ad32ec5fa49a271d801e8dd7c6187fd8342a78dbbc8b34713e1", hash.Hex())
}

func TestEncoder_UnknownSelector(t *testing.T) {
	t.Parallel()

	_, err := NewEncoder(
		0,
		5,
		false,
		true,
	)
	require.Error(t, err)
	require.Equal(t, "invalid chain ID: 0", err.Error())
}
