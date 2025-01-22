package solana

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
)

func Test_NewTimelockExecutor(t *testing.T) {
	t.Parallel()

	client := &rpc.Client{}
	auth, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	executor := NewTimelockExecutor(client, auth)

	require.NotNil(t, executor)
	require.Equal(t, executor.client, client)
	require.Equal(t, executor.auth, auth)
	require.NotNil(t, executor.TimelockInspector)
}

func Test_TimelockExecutor_Client(t *testing.T) {
	t.Parallel()

	client := &rpc.Client{}
	auth, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	executor := NewTimelockExecutor(client, auth)

	require.NotNil(t, executor.Client(), client)
}

func Test_TimelockExecutor_AuthPublicKey(t *testing.T) {
	t.Parallel()

	client := &rpc.Client{}
	auth, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	executor := NewTimelockExecutor(client, auth)

	require.NotNil(t, executor.AuthPublicKey(), auth.PublicKey())
}

func Test_TimelockExecutor_Execute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := &rpc.Client{}
	auth, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	tests := []struct {
		name            string
		bop             types.BatchOperation
		timelockAddress string
		predecessor     common.Hash
		salt            common.Hash
		want            string
		wantErr         string
	}{
		{
			name:    "error: not implemented",
			wantErr: "not implemented",
		},
	}
	for _, tt := range tests {
		executor := NewTimelockExecutor(client, auth)

		got, err := executor.Execute(ctx, tt.bop, tt.timelockAddress, tt.predecessor, tt.salt)

		if tt.wantErr == "" {
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		} else {
			require.ErrorContains(t, err, tt.wantErr)
		}
	}
}
