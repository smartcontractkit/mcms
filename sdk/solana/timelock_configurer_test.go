package solana

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/solana/mocks"
)

func TestNewTimelockConfigurer(t *testing.T) {
	t.Parallel()

	client := &rpc.Client{}
	auth, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	configurer := NewTimelockConfigurer(client, auth)

	require.NotNil(t, configurer)
	require.Equal(t, auth, configurer.auth)
}

func TestTimelockConfigurer_GrantRolesPanics(t *testing.T) {
	t.Parallel()

	configurer := NewTimelockConfigurer(nil, solana.PrivateKey{})

	require.PanicsWithValue(t, "not implemented", func() {
		_, _ = configurer.GrantRoles(t.Context(), "timelock", sdk.TimelockRoleProposer, []string{"address"})
	})
}

func TestTimelockConfigurer_UpdateDelay(t *testing.T) { //nolint:paralleltest
	auth, err := solana.PrivateKeyFromBase58(dummyPrivateKey)
	require.NoError(t, err)

	timelockAddress := fmt.Sprintf("%s.%s", testTimelockProgramID.String(), testTimelockSeed)

	tests := []struct {
		name      string
		address   string
		newDelay  uint64
		setup     func(*testing.T, *TimelockConfigurer, *mocks.JSONRPCClient)
		want      string
		assertion assert.ErrorAssertionFunc
	}{
		{
			name:     "success",
			address:  timelockAddress,
			newDelay: 120,
			setup: func(t *testing.T, _ *TimelockConfigurer, m *mocks.JSONRPCClient) {
				t.Helper()
				mockSolanaTransaction(t, m, 20, 5,
					"2QUBE2GqS8PxnGP1EBrWpLw3La4XkEUz5NKXJTdTHoA43ANkf5fqKwZ8YPJVAi3ApefbbbCYJipMVzUa7kg3a7v6",
					nil, nil)
			},
			want:      "2QUBE2GqS8PxnGP1EBrWpLw3La4XkEUz5NKXJTdTHoA43ANkf5fqKwZ8YPJVAi3ApefbbbCYJipMVzUa7kg3a7v6",
			assertion: assert.NoError,
		},
		{
			name:     "error: invalid address",
			address:  "bad ...format",
			newDelay: 120,
			setup:    func(t *testing.T, _ *TimelockConfigurer, _ *mocks.JSONRPCClient) { t.Helper() },
			assertion: func(t assert.TestingT, err error, _ ...any) bool {
				return assert.ErrorIs(t, err, ErrInvalidContractAddressFormat)
			},
		},
		{
			name:     "error: get latest blockhash fails",
			address:  timelockAddress,
			newDelay: 120,
			setup: func(t *testing.T, _ *TimelockConfigurer, m *mocks.JSONRPCClient) {
				t.Helper()
				t.Setenv("MCMS_SOLANA_MAX_RETRIES", "1")

				mockSolanaTransaction(t, m, 20, 5,
					"2QUBE2GqS8PxnGP1EBrWpLw3La4XkEUz5NKXJTdTHoA43ANkf5fqKwZ8YPJVAi3ApefbbbCYJipMVzUa7kg3a7v6",
					nil, errors.New("send failed"))
			},
			assertion: func(t assert.TestingT, err error, _ ...any) bool {
				return assert.ErrorContains(t, err, "unable to update delay")
			},
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			jsonRPCClient := mocks.NewJSONRPCClient(t)
			client := rpc.NewWithCustomRPCClient(jsonRPCClient)
			configurer := NewTimelockConfigurer(client, auth)
			tt.setup(t, configurer, jsonRPCClient)

			got, err := configurer.UpdateDelay(t.Context(), tt.address, tt.newDelay)

			tt.assertion(t, err)
			assert.Equal(t, tt.want, got.Hash)
		})
	}
}
