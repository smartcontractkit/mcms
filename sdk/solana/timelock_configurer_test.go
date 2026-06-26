package solana

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/solana/mocks"
	"github.com/smartcontractkit/mcms/types"
)

func TestNewTimelockConfigurer(t *testing.T) {
	t.Parallel()

	client := &rpc.Client{}
	auth, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	configurer := NewTimelockConfigurer(client, auth)

	require.NotNil(t, configurer)
	require.Equal(t, auth, configurer.auth)
	require.Equal(t, auth.PublicKey(), configurer.authorityAccount)
	require.False(t, configurer.skipSend)
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

func TestTimelockConfigurer_GrantRoleRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	auth, err := solana.PrivateKeyFromBase58(dummyPrivateKey)
	require.NoError(t, err)

	timelockAddress := fmt.Sprintf("%s.%s", testTimelockProgramID.String(), testTimelockSeed)
	target, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	configurer := NewTimelockConfigurer(&rpc.Client{}, auth)

	_, err = configurer.GrantRole(t.Context(), "bad ...format", sdk.TimelockRoleProposer, target.PublicKey().String())
	require.ErrorIs(t, err, ErrInvalidContractAddressFormat)

	_, err = configurer.GrantRole(t.Context(), timelockAddress, sdk.TimelockRoleAdmin, target.PublicKey().String())
	require.EqualError(t, err, "admin role is not grantable via access controller on solana")

	_, err = configurer.GrantRole(t.Context(), timelockAddress, sdk.TimelockRole(99), target.PublicKey().String())
	require.EqualError(t, err, "invalid timelock role: 99")

	_, err = configurer.GrantRole(t.Context(), timelockAddress, sdk.TimelockRoleProposer, "not-a-pubkey")
	require.EqualError(t, err, "invalid target address: not-a-pubkey")

	_, err = configurer.GrantRole(t.Context(), timelockAddress, sdk.TimelockRoleProposer, solana.PublicKey{}.String())
	require.EqualError(t, err, "invalid target address: "+solana.PublicKey{}.String())
}

func TestTimelockConfigurer_GrantRole(t *testing.T) { //nolint:paralleltest
	auth, err := solana.PrivateKeyFromBase58(dummyPrivateKey)
	require.NoError(t, err)

	timelockAddress := fmt.Sprintf("%s.%s", testTimelockProgramID.String(), testTimelockSeed)
	target, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	configPDA, err := FindTimelockConfigPDA(testTimelockProgramID, testTimelockSeed)
	require.NoError(t, err)

	config := createTimelockConfig(t)
	accessControllerProgramID, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)
	overrideAuthority, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	tests := []struct {
		name      string
		options   []timelockConfigurerOption
		setup     func(*testing.T, *mocks.JSONRPCClient)
		wantHash  string
		assertion assert.ErrorAssertionFunc
	}{
		{
			name: "success send",
			setup: func(t *testing.T, m *mocks.JSONRPCClient) {
				t.Helper()
				mockGetAccountInfo(t, m, configPDA, config, nil)
				mockGetAccountOwner(t, m, config.ProposerRoleAccessController, accessControllerProgramID.PublicKey(), nil)
				mockSolanaTransaction(t, m, 20, 5,
					"2QUBE2GqS8PxnGP1EBrWpLw3La4XkEUz5NKXJTdTHoA43ANkf5fqKwZ8YPJVAi3ApefbbbCYJipMVzUa7kg3a7v6",
					nil, nil)
			},
			wantHash:  "2QUBE2GqS8PxnGP1EBrWpLw3La4XkEUz5NKXJTdTHoA43ANkf5fqKwZ8YPJVAi3ApefbbbCYJipMVzUa7kg3a7v6",
			assertion: assert.NoError,
		},
		{
			name:    "success no send",
			options: []timelockConfigurerOption{WithDoNotSendTimelockInstructionsOnChain()},
			setup: func(t *testing.T, m *mocks.JSONRPCClient) {
				t.Helper()
				mockGetAccountInfo(t, m, configPDA, config, nil)
				mockGetAccountOwner(t, m, config.ProposerRoleAccessController, accessControllerProgramID.PublicKey(), nil)
			},
			wantHash:  "",
			assertion: assert.NoError,
		},
		{
			name: "success authority override",
			options: []timelockConfigurerOption{
				WithDoNotSendTimelockInstructionsOnChain(),
				WithTimelockAuthorityAccount(overrideAuthority.PublicKey()),
			},
			setup: func(t *testing.T, m *mocks.JSONRPCClient) {
				t.Helper()
				mockGetAccountInfo(t, m, configPDA, config, nil)
				mockGetAccountOwner(t, m, config.ProposerRoleAccessController, accessControllerProgramID.PublicKey(), nil)
			},
			wantHash:  "",
			assertion: assert.NoError,
		},
		{
			name: "error send fails",
			setup: func(t *testing.T, m *mocks.JSONRPCClient) {
				t.Helper()
				t.Setenv("MCMS_SOLANA_MAX_RETRIES", "1")
				mockGetAccountInfo(t, m, configPDA, config, nil)
				mockGetAccountOwner(t, m, config.ProposerRoleAccessController, accessControllerProgramID.PublicKey(), nil)
				mockSolanaTransaction(t, m, 20, 5,
					"2QUBE2GqS8PxnGP1EBrWpLw3La4XkEUz5NKXJTdTHoA43ANkf5fqKwZ8YPJVAi3ApefbbbCYJipMVzUa7kg3a7v6",
					nil, errors.New("send failed"))
			},
			assertion: func(t assert.TestingT, err error, _ ...any) bool {
				return assert.EqualError(t, err, "unable to grant role: unable to send instruction: send failed")
			},
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			jsonRPCClient := mocks.NewJSONRPCClient(t)
			client := rpc.NewWithCustomRPCClient(jsonRPCClient)
			configurer := NewTimelockConfigurer(client, auth, tt.options...)
			tt.setup(t, jsonRPCClient)

			got, err := configurer.GrantRole(t.Context(), timelockAddress, sdk.TimelockRoleProposer, target.PublicKey().String())

			tt.assertion(t, err)
			assert.Equal(t, tt.wantHash, got.Hash)
			if err == nil && tt.wantHash == "" {
				require.Equal(t, chainsel.FamilySolana, got.ChainFamily)
				tx, ok := got.RawData.(types.Transaction)
				require.True(t, ok)
				require.Equal(t, testTimelockProgramID.String(), tx.To)
				require.Equal(t, "RBACTimelock", tx.OperationMetadata.ContractType)
				require.Equal(t, []string{"RBACTimelock", "GrantRole"}, tx.OperationMetadata.Tags)

				var additionalFields AdditionalFields
				require.NoError(t, json.Unmarshal(tx.AdditionalFields, &additionalFields))
				require.NotEmpty(t, additionalFields.Accounts)
			}
		})
	}
}
