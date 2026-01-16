package chainwrappers

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

func TestBuildConverters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		metadata    map[types.ChainSelector]types.ChainMetadata
		expectTypes map[types.ChainSelector]any
		expectErr   string
	}{
		{
			name: "supported families",
			metadata: map[types.ChainSelector]types.ChainMetadata{
				chaintest.Chain2Selector: {},
				chaintest.Chain4Selector: {},
				chaintest.Chain5Selector: {},
			},
			expectTypes: map[types.ChainSelector]any{
				chaintest.Chain2Selector: (*evm.TimelockConverter)(nil),
				chaintest.Chain4Selector: (*solana.TimelockConverter)(nil),
				chaintest.Chain5Selector: (*aptos.TimelockConverter)(nil),
			},
		},
		{
			name: "unsupported family",
			metadata: map[types.ChainSelector]types.ChainMetadata{
				chaintest.Chain6Selector: {},
			},
			expectErr: "unsupported chain family",
		},
		{
			name: "invalid selector",
			metadata: map[types.ChainSelector]types.ChainMetadata{
				chaintest.ChainInvalidSelector: {},
			},
			expectErr: "error getting chain family",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			converters, err := BuildConverters(tc.metadata)

			if tc.expectErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectErr)

				return
			}

			require.NoError(t, err)
			require.Len(t, converters, len(tc.expectTypes))
			for selector, expectedType := range tc.expectTypes {
				converter, ok := converters[selector]
				require.True(t, ok)
				require.IsType(t, expectedType, converter)
			}
		})
	}
}
