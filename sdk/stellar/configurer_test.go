package stellar

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/types"
)

func TestConfigurer_SetConfig_nilConfig(t *testing.T) {
	t.Parallel()

	c := NewConfigurer(&recordingInvoker{})
	ctx := context.Background()

	_, err := c.SetConfig(ctx, "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20", nil, false)
	require.Error(t, err)
}

func TestConfigurer_SetConfig_routesToSetConfig(t *testing.T) {
	t.Parallel()

	inv := &recordingInvoker{}
	c := NewConfigurer(inv)
	ctx := context.Background()

	cfg := &types.Config{
		Quorum:  1,
		Signers: []common.Address{{1}},
	}

	res, err := c.SetConfig(ctx, "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20", cfg, true)
	require.NoError(t, err)
	require.Equal(t, chainsel.FamilyStellar, res.ChainFamily)
	require.Equal(t, "set_config", inv.lastFn)
}
