package solana

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk/solana/mocks"
	"github.com/smartcontractkit/mcms/types"
)

func TestNewTimelockExecutor(t *testing.T) {
	t.Parallel()

	client := &rpc.Client{}
	auth, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)
	executor := NewTimelockExecutor(client, auth)

	require.NotNil(t, executor)
	require.Equal(t, executor.auth, auth)
}

func Test_TimelockExecutor_Execute(t *testing.T) { //nolint:paralleltest
	type args struct {
		bop             types.BatchOperation
		salt            [32]byte
		predecessor     [32]byte
		timelockAddress string
	}
	selector := cselectors.SOLANA_DEVNET.Selector
	auth, err := solana.PrivateKeyFromBase58(dummyPrivateKey)
	require.NoError(t, err)

	data := []byte{1, 2, 3, 4}
	ctx := context.Background()
	accounts := []*solana.AccountMeta{
		{
			PublicKey:  solana.NewWallet().PublicKey(),
			IsSigner:   false,
			IsWritable: true,
		},
		{
			PublicKey:  solana.NewWallet().PublicKey(),
			IsSigner:   false,
			IsWritable: false,
		},
	}
	configPDA, err := FindTimelockConfigPDA(testTimelockProgramID, testTimelockSeed)
	require.NoError(t, err)
	config := createTimelockConfig(t)
	tx, err := NewTransaction(testTimelockProgramID.String(), data, big.NewInt(0), accounts, "solana-testing", []string{})
	require.NoError(t, err)
	tests := []struct {
		name      string
		args      args
		setup     func(*testing.T, *TimelockExecutor, *mocks.JSONRPCClient)
		want      string
		assertion assert.ErrorAssertionFunc
	}{
		{
			name: "success: ExecuteOperation",
			args: args{
				bop: types.BatchOperation{
					Transactions:  []types.Transaction{tx},
					ChainSelector: types.ChainSelector(selector),
				},
				timelockAddress: fmt.Sprintf("%s.%s", testTimelockProgramID.String(), testTimelockSeed),
			},
			setup: func(t *testing.T, e *TimelockExecutor, m *mocks.JSONRPCClient) {
				t.Helper()
				mockGetAccountInfo(t, m, configPDA, config, nil)
				mockSolanaTransaction(t, m, 20, 5,
					"2QUBE2GqS8PxnGP1EBrWpLw3La4XkEUz5NKXJTdTHoA43ANkf5fqKwZ8YPJVAi3ApefbbbCYJipMVzUa7kg3a7v6", nil, nil)
			},
			want:      "2QUBE2GqS8PxnGP1EBrWpLw3La4XkEUz5NKXJTdTHoA43ANkf5fqKwZ8YPJVAi3ApefbbbCYJipMVzUa7kg3a7v6",
			assertion: assert.NoError,
		},
		{
			name: "error: invalid timelock address",
			args: args{
				bop: types.BatchOperation{
					Transactions:  []types.Transaction{tx},
					ChainSelector: types.ChainSelector(selector),
				},
				timelockAddress: "bad ...format",
			},
			setup:     func(t *testing.T, e *TimelockExecutor, m *mocks.JSONRPCClient) { t.Helper() },
			assertion: assertErrorEquals("invalid solana contract address format: \"bad ...format\""),
		},
		{
			name: "error: invalid additional fields",
			args: args{
				bop: types.BatchOperation{
					Transactions: []types.Transaction{{
						Data:             bytes.Repeat([]byte{1}, 100),
						To:               ContractAddress(testTimelockProgramID, testTimelockSeed),
						AdditionalFields: []byte(`invalid JSON`),
					}},
					ChainSelector: types.ChainSelector(selector),
				},
				timelockAddress: fmt.Sprintf("%s.%s", testTimelockProgramID.String(), testTimelockSeed),
			},
			setup: func(t *testing.T, e *TimelockExecutor, m *mocks.JSONRPCClient) { t.Helper() },
			assertion: assertErrorEquals("unable to get InstructionData from batch operation: " +
				"unable to unmarshal additional fields: " +
				"invalid character 'i' looking for beginning of value\n" +
				"invalid JSON"),
		},
		{
			name: "error: invalid To program field",
			args: args{
				bop: types.BatchOperation{
					Transactions: []types.Transaction{{
						Data:             bytes.Repeat([]byte{1}, 100),
						To:               "invalid.address",
						AdditionalFields: []byte("{}"),
					}},
					ChainSelector: types.ChainSelector(selector),
				},
				timelockAddress: fmt.Sprintf("%s.%s", testTimelockProgramID.String(), testTimelockSeed),
			},
			setup: func(t *testing.T, e *TimelockExecutor, m *mocks.JSONRPCClient) { t.Helper() },
			assertion: assertErrorEquals("unable to get InstructionData from batch operation: " +
				"unable to parse program id from To field: " +
				"unable to parse base58 solana program id: " +
				"decode: invalid base58 digit ('l')"),
		},
		{
			name: "error: GetAccountInfo fails",
			args: args{
				bop: types.BatchOperation{
					Transactions:  []types.Transaction{tx},
					ChainSelector: types.ChainSelector(selector),
				},
				timelockAddress: fmt.Sprintf("%s.%s", testTimelockProgramID.String(), testTimelockSeed),
			},
			setup: func(t *testing.T, e *TimelockExecutor, m *mocks.JSONRPCClient) {
				t.Helper()
				mockGetAccountInfo(t, m, configPDA, config, errors.New("GetAccountInfo error"))
			},
			assertion: assertErrorEquals("unable to read config pda: GetAccountInfo error"),
		},
		{
			name: "error: send tx error",
			args: args{
				bop: types.BatchOperation{
					Transactions:  []types.Transaction{tx},
					ChainSelector: types.ChainSelector(selector),
				},
				timelockAddress: fmt.Sprintf("%s.%s", testTimelockProgramID.String(), testTimelockSeed),
			},
			setup: func(t *testing.T, e *TimelockExecutor, m *mocks.JSONRPCClient) {
				t.Helper()
				t.Setenv("MCMS_SOLANA_MAX_RETRIES", "1")

				mockGetAccountInfo(t, m, configPDA, config, nil)
				mockSolanaTransaction(t, m, 20, 5,
					"2QUBE2GqS8PxnGP1EBrWpLw3La4XkEUz5NKXJTdTHoA43ANkf5fqKwZ8YPJVAi3ApefbbbCYJipMVzUa7kg3a7v6",
					nil, errors.New("invalid tx"))
			},
			assertion: func(t assert.TestingT, err error, i ...any) bool {
				return assert.EqualError(t, err, "unable to call execute operation instruction: unable to send instruction: invalid tx")
			},
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			jsonRPCClient := mocks.NewJSONRPCClient(t)
			client := rpc.NewWithCustomRPCClient(jsonRPCClient)
			e := NewTimelockExecutor(client, auth)
			tt.setup(t, e, jsonRPCClient)

			got, err := e.Execute(ctx, tt.args.bop, tt.args.timelockAddress, tt.args.predecessor, tt.args.salt)

			tt.assertion(t, err, fmt.Sprintf("%q. Executor.ExecuteOperation()", tt.name))
			assert.Equalf(t, tt.want, got.Hash, "%q. Executor.ExecuteOperation()", tt.name)
		})
	}
}

func assertErrorEquals(message string) func(t assert.TestingT, err error, i ...any) bool {
	return func(t assert.TestingT, err error, i ...any) bool {
		return assert.EqualError(t, err, message)
	}
}
