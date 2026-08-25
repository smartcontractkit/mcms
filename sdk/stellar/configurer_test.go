package stellar

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/stellar/mocks"
	"github.com/smartcontractkit/mcms/types"
)

func TestConfigurer_SetConfig(t *testing.T) {
	t.Parallel()

	const mcmAddr = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"
	signer1 := common.HexToAddress("0x1234")

	minvoker := mocks.NewInvoker(t)
	minvoker.EXPECT().InvokeContract(mock.Anything, mcmAddr, "set_config", mock.Anything).Return(nil, nil)
	configurer := NewConfigurer(minvoker)

	cfg, err := types.NewConfig(1, []common.Address{signer1}, nil)
	require.NoError(t, err)

	res, err := configurer.SetConfig(t.Context(), mcmAddr, &cfg, true)
	require.NoError(t, err)
	require.Equal(t, "stellar", res.ChainFamily)
}

func TestConfigurer_Ownership(t *testing.T) {
	t.Parallel()

	const mcmAddr = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"
	const newOwner = "CB7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"

	minvoker := mocks.NewInvoker(t)
	minvoker.EXPECT().InvokeContract(mock.Anything, mcmAddr, "transfer_ownership", mock.Anything).Return(nil, nil)
	minvoker.EXPECT().InvokeContract(mock.Anything, mcmAddr, "accept_ownership", mock.Anything).Return(nil, nil)
	minvoker.EXPECT().InvokeContract(mock.Anything, mcmAddr, "cancel_ownership_transfer", mock.Anything).Return(nil, nil)
	configurer := NewConfigurer(minvoker)

	// Transfer
	res, err := configurer.TransferOwnership(t.Context(), mcmAddr, newOwner)
	require.NoError(t, err)
	require.Equal(t, "stellar", res.ChainFamily)

	// Accept
	res, err = configurer.AcceptOwnership(t.Context(), mcmAddr)
	require.NoError(t, err)
	require.Equal(t, "stellar", res.ChainFamily)

	// Cancel
	res, err = configurer.CancelOwnershipTransfer(t.Context(), mcmAddr)
	require.NoError(t, err)
	require.Equal(t, "stellar", res.ChainFamily)
}
