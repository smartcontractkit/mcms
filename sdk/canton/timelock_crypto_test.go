package canton

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashTimelockOpID(t *testing.T) {
	t.Parallel()

	hash, err := hashTimelockOpID([]timelockCallForHash{{
		TargetInstanceAddress: "counter@party::abc",
		FunctionName:          "Increment",
		OperationData:         "deadbeef",
	}}, "00", "00")
	require.NoError(t, err)
	require.Len(t, hash, 64)
}

func TestIsInstanceAddressHex(t *testing.T) {
	t.Parallel()

	require.True(t, IsInstanceAddressHex("0xd4dcbc33d025740c32b65cb60d208a7eb8f99b3d90903ffe52616e14f9096995"))
	require.False(t, IsInstanceAddressHex("0xshort"))
	require.False(t, IsInstanceAddressHex("not-hex"))
}
