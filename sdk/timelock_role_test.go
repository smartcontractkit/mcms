package sdk

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTimelockRole_Valid(t *testing.T) {
	t.Parallel()

	require.True(t, TimelockRoleAdmin.Valid())
	require.False(t, TimelockRole(99).Valid())
}
