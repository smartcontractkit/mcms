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

func Test_NewTimelockConverter(t *testing.T) {
	t.Parallel()

	client := &rpc.Client{}
	auth, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	converter := NewTimelockConverter(client, auth.PublicKey())

	require.NotNil(t, converter)
	require.Equal(t, client, converter.client)
	require.NotNil(t, auth, converter.auth)
}

func Test_TimelockConverter_ConvertBatchToChainOperations(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := &rpc.Client{}
	auth, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	tests := []struct {
		name             string
		batchOp          types.BatchOperation
		timelockAddress  string
		delay            types.Duration
		action           types.TimelockAction
		predecessor      common.Hash
		salt             common.Hash
		wantOperations   []types.Operation
		wantPredecessors common.Hash
		wantErr          string
	}{
		{
			name:    "failure: not implemented",
			wantErr: "not implemented",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			converter := NewTimelockConverter(client, auth.PublicKey())

			operations, predecessors, err := converter.ConvertBatchToChainOperations(ctx, tt.batchOp, tt.timelockAddress,
				tt.delay, tt.action, tt.predecessor, tt.salt)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.wantOperations, operations)
				require.Equal(t, tt.wantPredecessors, predecessors)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
