package stellar

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
)

func TestConfigurer_SetConfig(t *testing.T) {
	t.Parallel()

	const mcmAddr = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"
	signer1 := common.HexToAddress("0x1234")

	invoker := newMockInvoker()
	configurer := NewConfigurer(invoker)

	cfg, err := types.NewConfig(1, []common.Address{signer1}, nil)
	require.NoError(t, err)

	res, err := configurer.SetConfig(context.Background(), mcmAddr, &cfg, true)
	require.NoError(t, err)
	require.Equal(t, "stellar", res.ChainFamily)

	require.Len(t, invoker.calls, 1)
	require.Equal(t, "set_config", invoker.calls[0].FunctionName)
	require.Equal(t, mcmAddr, invoker.calls[0].ContractID)
}

func TestConfigurer_Ownership(t *testing.T) {
	t.Parallel()

	const mcmAddr = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"
	const newOwner = "CB7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"

	invoker := newMockInvoker()
	configurer := NewConfigurer(invoker)

	// Transfer
	res, err := configurer.TransferOwnership(context.Background(), mcmAddr, newOwner)
	require.NoError(t, err)
	require.Equal(t, "stellar", res.ChainFamily)
	require.Len(t, invoker.calls, 1)
	require.Equal(t, "transfer_ownership", invoker.calls[0].FunctionName)

	// Accept
	res, err = configurer.AcceptOwnership(context.Background(), mcmAddr)
	require.NoError(t, err)
	require.Equal(t, "stellar", res.ChainFamily)
	require.Len(t, invoker.calls, 2)
	require.Equal(t, "accept_ownership", invoker.calls[1].FunctionName)

	// Cancel
	res, err = configurer.CancelOwnershipTransfer(context.Background(), mcmAddr)
	require.NoError(t, err)
	require.Equal(t, "stellar", res.ChainFamily)
	require.Len(t, invoker.calls, 3)
	require.Equal(t, "cancel_ownership_transfer", invoker.calls[2].FunctionName)
}
