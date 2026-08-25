package stellar

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/stellar/mocks"

	"github.com/smartcontractkit/mcms/sdk"
)

func TestTimelockConfigurer_UpdateDelayAndGrantRole(t *testing.T) {
	t.Parallel()

	const timelockAddr = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"
	const caller = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"
	const target = "CB7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"

	invoker := mocks.NewInvoker(t)

	invoker.
		On(
			"InvokeContract",
			mock.Anything,
			timelockAddr,
			"update_delay",
			mock.Anything,
		).
		Return(nil, nil).
		Once()

	invoker.
		On(
			"InvokeContract",
			mock.Anything,
			timelockAddr,
			"grant_role",
			mock.Anything,
		).
		Return(nil, nil).
		Once()

	configurer := NewTimelockConfigurerFromInvoker(invoker, caller)

	res, err := configurer.UpdateDelay(
		context.Background(),
		timelockAddr,
		200,
	)
	require.NoError(t, err)
	require.Equal(t, "stellar", res.ChainFamily)

	invoker.AssertNumberOfCalls(t, "InvokeContract", 1)
	invoker.AssertCalled(
		t,
		"InvokeContract",
		mock.Anything,
		timelockAddr,
		"update_delay",
		mock.Anything,
	)

	res, err = configurer.GrantRole(
		context.Background(),
		timelockAddr,
		sdk.TimelockRoleProposer,
		target,
	)
	require.NoError(t, err)
	require.Equal(t, "stellar", res.ChainFamily)

	invoker.AssertNumberOfCalls(t, "InvokeContract", 2)
	invoker.AssertCalled(
		t,
		"InvokeContract",
		mock.Anything,
		timelockAddr,
		"grant_role",
		mock.Anything,
	)
}
