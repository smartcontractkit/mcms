package chainwrappers

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

func TestBuildRoleMetadataDeriver(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		selector   types.ChainSelector
		expectType any
		expectErr  string
	}{
		{name: "evm", selector: chaintest.Chain2Selector, expectType: (*evm.RoleMetadataDeriver)(nil)},
		{name: "unsupported family", selector: chaintest.Chain8Selector, expectErr: "unsupported chain family"},
		{name: "invalid selector", selector: chaintest.ChainInvalidSelector, expectErr: "error getting chain family"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deriver, err := BuildRoleMetadataDeriver(nil, tc.selector)
			if tc.expectErr != "" {
				require.ErrorContains(t, err, tc.expectErr)
				return
			}
			require.NoError(t, err)
			require.IsType(t, tc.expectType, deriver)
		})
	}
}
