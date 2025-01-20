package solana

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/solana/mocks"
	"github.com/smartcontractkit/mcms/types"
)

func TestNewTimelockExecutor(t *testing.T) {
	t.Parallel()

	client := &rpc.Client{}
	auth := solana.MustPrivateKeyFromBase58("DmPfeHBC8Brf8s5qQXi25bmJ996v6BHRtaLc6AH51yFGSqQpUMy1oHkbbXobPNBdgGH2F29PAmoq9ZZua4K9vCc")
	wallet := solana.NewWallet()
	executor := NewTimelockExecutor(client, auth, wallet.PublicKey())

	require.NotNil(t, executor)
	require.Equal(t, executor.roleAccessController, wallet.PublicKey())
	require.Equal(t, executor.auth, auth)
}

func TestTimelockExecutor_Execute(t *testing.T) {
	t.Parallel()

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
	tx, err := NewTransaction(testTimelockProgramID.String(), data, accounts, "solana-testing", []string{})
	require.NoError(t, err)
	tests := []struct {
		name string
		args args

		mockSetup func(*mocks.JSONRPCClient)
		want      string
		assertion assert.ErrorAssertionFunc
		wantErr   error
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
			mockSetup: func(m *mocks.JSONRPCClient) {
				mockSolanaTransaction(t, m, 20, 5, "2QUBE2GqS8PxnGP1EBrWpLw3La4XkEUz5NKXJTdTHoA43ANkf5fqKwZ8YPJVAi3ApefbbbCYJipMVzUa7kg3a7v6", nil)
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
			mockSetup: func(m *mocks.JSONRPCClient) {},
			wantErr:   fmt.Errorf("invalid solana contract address format: \"bad ...format\""),
			assertion: func(t assert.TestingT, err error, i ...any) bool {
				return assert.EqualError(t, err, "invalid solana contract address format: \"bad ...format\"")
			},
		},
		{
			name: "error: invalid additional fields",
			args: args{
				bop: types.BatchOperation{
					Transactions: []types.Transaction{{
						Data:             bytes.Repeat([]byte{1}, 100),
						To:               fmt.Sprintf("%s.%s", testTimelockProgramID.String(), testTimelockSeed),
						AdditionalFields: []byte(`invalid JSON`),
					}},
					ChainSelector: types.ChainSelector(selector),
				},
				timelockAddress: fmt.Sprintf("%s.%s", testTimelockProgramID.String(), testTimelockSeed),
			},
			mockSetup: func(m *mocks.JSONRPCClient) {},
			wantErr:   fmt.Errorf("invalid character 'i' looking for beginning of value"),
			assertion: func(t assert.TestingT, err error, i ...any) bool {
				return assert.EqualError(t, err, "invalid character 'i' looking for beginning of value")
			},
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
			mockSetup: func(m *mocks.JSONRPCClient) {},
			wantErr:   fmt.Errorf("unable to get hash from base58 To address: decode: invalid base58 digit ('l')"),
			assertion: func(t assert.TestingT, err error, i ...any) bool {
				return assert.EqualError(t, err, "unable to get hash from base58 To address: decode: invalid base58 digit ('l')")
			},
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
			mockSetup: func(m *mocks.JSONRPCClient) {
				mockSolanaTransaction(t, m, 20, 5, "2QUBE2GqS8PxnGP1EBrWpLw3La4XkEUz5NKXJTdTHoA43ANkf5fqKwZ8YPJVAi3ApefbbbCYJipMVzUa7kg3a7v6", fmt.Errorf("invalid tx"))
			},
			wantErr: fmt.Errorf("unable to call execute operation instruction: unable to send instruction: invalid tx"),
			assertion: func(t assert.TestingT, err error, i ...any) bool {
				return assert.EqualError(t, err, "unable to call execute operation instruction: unable to send instruction: invalid tx")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			jsonRPCClient := mocks.NewJSONRPCClient(t)
			roleAcc := solana.NewWallet()
			client := rpc.NewWithCustomRPCClient(jsonRPCClient)
			tt.mockSetup(jsonRPCClient)
			e := NewTimelockExecutor(client, auth, roleAcc.PublicKey())

			got, err := e.Execute(ctx, tt.args.bop, tt.args.timelockAddress, tt.args.predecessor, tt.args.salt)
			if tt.wantErr != nil {
				tt.assertion(t, err, fmt.Sprintf("%q. Executor.ExecuteOperation()", tt.name))
			} else {
				require.NoError(t, err)
				require.Equalf(t, tt.want, got, "%q. Executor.ExecuteOperation()", tt.name)
			}
		})
	}
}
