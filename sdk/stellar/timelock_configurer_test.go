package stellar

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTimelockConfigurer_UpdateDelayRequiresAdmin(t *testing.T) {
	t.Parallel()
	c := NewTimelockConfigurer(&timelockSimInvoker{}, "")
	_, err := c.UpdateDelay(t.Context(), stringsRepeatHexAddr('c'), 10)
	require.ErrorContains(t, err, "admin caller")
}
