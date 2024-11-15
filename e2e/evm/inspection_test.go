//go:build e2e

package evm

import (
	"testing"

	"github.com/smartcontractkit/chainlink-testing-framework/framework"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/stretchr/testify/require"
)

type Config struct {
	BlockchainA *blockchain.Input `toml:"blockchain_a" validate:"required"`
}

func TestMe(t *testing.T) {
	in, err := framework.Load[Config](t)
	require.NoError(t, err)

	bc, err := blockchain.NewBlockchainNetwork(in.BlockchainA)
	require.NoError(t, err)

	t.Run("test something", func(t *testing.T) {
		require.NotEmpty(t, bc.Nodes[0].HostHTTPUrl)
	})
}
