package solana

import (
	"encoding/json"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/assert"
)

func TestNewSolanaChainMetadata(t *testing.T) {
	t.Parallel()

	// Create sample public keys.
	mcmProgramID, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	proposerKey, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	cancellerKey, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	bypasserKey, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	tests := []struct {
		name            string
		startingOpCount uint64
		mcmProgramID    solana.PublicKey
		mcmInstanceSeed PDASeed
		proposerKey     solana.PublicKey
		cancellerKey    solana.PublicKey
		bypasserKey     solana.PublicKey
		wantErr         string
	}{
		{
			name:            "valid metadata",
			startingOpCount: 100,
			mcmProgramID:    mcmProgramID.PublicKey(),
			mcmInstanceSeed: PDASeed([32]byte{1, 2, 3, 4}),
			proposerKey:     proposerKey.PublicKey(),
			cancellerKey:    cancellerKey.PublicKey(),
			bypasserKey:     bypasserKey.PublicKey(),
		},
		{
			name:            "invalid metadata",
			startingOpCount: 100,
			mcmProgramID:    solana.PublicKey{},
			mcmInstanceSeed: PDASeed([32]byte{1, 2, 3, 4}),
			proposerKey:     proposerKey.PublicKey(),
			cancellerKey:    cancellerKey.PublicKey(),
			bypasserKey:     bypasserKey.PublicKey(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			metadata, err := NewChainMetadata(tc.startingOpCount, tc.mcmProgramID, tc.mcmInstanceSeed, tc.proposerKey, tc.cancellerKey, tc.bypasserKey)
			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tc.startingOpCount, metadata.StartingOpCount)

			expectedMCMAddress := ContractAddress(tc.mcmProgramID, tc.mcmInstanceSeed)
			assert.Equal(t, expectedMCMAddress, metadata.MCMAddress)

			var additionalFields AdditionalFieldsMetadata
			err = json.Unmarshal(metadata.AdditionalFields, &additionalFields)
			require.NoError(t, err)

			expectedAdditionalFields := AdditionalFieldsMetadata{
				ProposerRoleAccessController:  tc.proposerKey,
				CancellerRoleAccessController: tc.cancellerKey,
				BypasserRoleAccessController:  tc.bypasserKey,
			}
			assert.Equal(t, expectedAdditionalFields, additionalFields)
		})
	}
}
