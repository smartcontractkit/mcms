package solana

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/google/go-cmp/cmp"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/timelock"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/assert"

	"github.com/smartcontractkit/mcms/sdk/solana/mocks"
	"github.com/smartcontractkit/mcms/types"
)

func TestNewChainMetadataFromTimelock(t *testing.T) {
	t.Parallel()

	type params struct {
		startingOpCount uint64
		mcmProgramID    solana.PublicKey
		mcmInstanceSeed PDASeed
		timelock        solana.PublicKey
		timelockSeed    PDASeed
	}

	programID := solana.NewWallet().PublicKey()
	timelockProgramID := solana.NewWallet().PublicKey()
	MCMSeed := PDASeed([32]byte{1, 2, 3, 4})
	timelockSeed := PDASeed([32]byte{1, 2, 3, 4})

	configPDA, err := FindTimelockConfigPDA(timelockProgramID, timelockSeed)
	require.NoError(t, err)

	tests := []struct {
		name         string
		params       params
		setupMock    func(mock *mocks.JSONRPCClient)
		wantMetadata *types.ChainMetadata
		wantErr      error
	}{
		{
			name: "valid metadata",
			params: params{
				startingOpCount: 100,
				mcmProgramID:    programID,
				mcmInstanceSeed: MCMSeed,
				timelock:        timelockProgramID,
				timelockSeed:    timelockSeed,
			},
			setupMock: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mockGetAccountInfo(t, mockJSONRPCClient, configPDA, &timelock.Config{}, nil)
			},
			wantMetadata: &types.ChainMetadata{
				StartingOpCount:  100,
				MCMAddress:       ContractAddress(programID, MCMSeed),
				AdditionalFields: json.RawMessage(`{"proposerRoleAccessController":"11111111111111111111111111111111","cancellerRoleAccessController":"11111111111111111111111111111111","bypasserRoleAccessController":"11111111111111111111111111111111"}`),
			},
		},
		{
			name: "error rpc call",
			params: params{
				startingOpCount: 100,
				mcmProgramID:    programID,
				mcmInstanceSeed: MCMSeed,
				timelock:        timelockProgramID,
				timelockSeed:    timelockSeed,
			},
			wantErr: errors.New("unable to read timelock config pda: rpc error"),
			setupMock: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				err := errors.New("rpc error")
				mockGetAccountInfo(t, mockJSONRPCClient, configPDA, &timelock.Config{}, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			jsonRpc := mocks.NewJSONRPCClient(t)
			tt.setupMock(jsonRpc)
			client := rpc.NewWithCustomRPCClient(jsonRpc)
			metadata, err := NewChainMetadataFromTimelock(
				context.Background(),
				client,
				tt.params.startingOpCount,
				tt.params.mcmProgramID,
				tt.params.mcmInstanceSeed,
				tt.params.timelock,
				tt.params.timelockSeed)
			if tt.wantErr == nil {
				require.NoError(t, err, "expected no error but got one")
				require.Empty(t, cmp.Diff(tt.wantMetadata, &metadata))
			} else {
				// Assert the error message matches the expected error.
				require.NotNil(t, metadata)
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestAdditionalFieldsMetadata_Validate(t *testing.T) {
	t.Parallel()

	// Create valid public keys for testing.
	validPK1, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)
	validPK2, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)
	validPK3, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)
	zeroPK := solana.PublicKey{} // zero value public key

	tests := []struct {
		name        string
		fields      AdditionalFieldsMetadata
		expectedErr error
	}{
		{
			name: "all valid keys",
			fields: AdditionalFieldsMetadata{
				ProposerRoleAccessController:  validPK1.PublicKey(),
				CancellerRoleAccessController: validPK2.PublicKey(),
				BypasserRoleAccessController:  validPK3.PublicKey(),
			},
			expectedErr: nil,
		},
		{
			name: "zero proposer key",
			fields: AdditionalFieldsMetadata{
				ProposerRoleAccessController:  zeroPK,
				CancellerRoleAccessController: validPK2.PublicKey(),
				BypasserRoleAccessController:  validPK3.PublicKey(),
			},
			expectedErr: errors.New("Key: 'AdditionalFieldsMetadata.ProposerRoleAccessController' Error:Field validation for 'ProposerRoleAccessController' failed on the 'required' tag"),
		},
		{
			name: "zero canceller key",
			fields: AdditionalFieldsMetadata{
				ProposerRoleAccessController:  validPK1.PublicKey(),
				CancellerRoleAccessController: zeroPK,
				BypasserRoleAccessController:  validPK3.PublicKey(),
			},
			expectedErr: errors.New("Key: 'AdditionalFieldsMetadata.CancellerRoleAccessController' Error:Field validation for 'CancellerRoleAccessController' failed on the 'required' tag"),
		},
		{
			name: "zero bypasser key",
			fields: AdditionalFieldsMetadata{
				ProposerRoleAccessController:  validPK1.PublicKey(),
				CancellerRoleAccessController: validPK2.PublicKey(),
				BypasserRoleAccessController:  zeroPK,
			},
			expectedErr: errors.New("Key: 'AdditionalFieldsMetadata.BypasserRoleAccessController' Error:Field validation for 'BypasserRoleAccessController' failed on the 'required' tag"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.fields.Validate()
			if tt.expectedErr == nil {
				require.NoError(t, err, "expected no error but got one")
			} else {
				// Assert the error message matches the expected error.
				require.EqualError(t, err, tt.expectedErr.Error())
			}
		})
	}
}

func TestValidateChainMetadata(t *testing.T) {
	t.Parallel()

	// Create some public keys for testing.
	zeroPK := solana.PublicKey{} // zero value public key

	// Valid additional fields.
	validFields := AdditionalFieldsMetadata{
		ProposerRoleAccessController:  solana.NewWallet().PublicKey(),
		CancellerRoleAccessController: solana.NewWallet().PublicKey(),
		BypasserRoleAccessController:  solana.NewWallet().PublicKey(),
	}
	validJSON, err := json.Marshal(validFields)
	require.NoError(t, err)

	// Missing required field.
	// Here we omit CancellerRoleAccessController so that field remains at its zero value.
	// Using an inline struct with only two fields.
	missingField := struct {
		ProposerRoleAccessController solana.PublicKey `json:"proposerRoleAccessController"`
		BypasserRoleAccessController solana.PublicKey `json:"bypasserRoleAccessController"`
	}{
		ProposerRoleAccessController: validFields.ProposerRoleAccessController,
		BypasserRoleAccessController: validFields.BypasserRoleAccessController,
	}
	missingFieldJSON, err := json.Marshal(missingField)
	require.NoError(t, err)

	// Zero value field: Proposer is zero.
	zeroField := AdditionalFieldsMetadata{
		ProposerRoleAccessController:  zeroPK,
		CancellerRoleAccessController: validFields.CancellerRoleAccessController,
		BypasserRoleAccessController:  validFields.BypasserRoleAccessController,
	}
	zeroFieldJSON, err := json.Marshal(zeroField)
	require.NoError(t, err)

	tests := []struct {
		name        string
		metadata    types.ChainMetadata
		expectedErr bool
	}{
		{
			name: "valid additional fields",
			metadata: types.ChainMetadata{
				AdditionalFields: validJSON,
			},
			expectedErr: false,
		},
		{
			name: "invalid JSON",
			metadata: types.ChainMetadata{
				AdditionalFields: []byte("not a json"),
			},
			expectedErr: true,
		},
		{
			name: "missing required field",
			metadata: types.ChainMetadata{
				AdditionalFields: missingFieldJSON,
			},
			expectedErr: true,
		},
		{
			name: "zero value in one field",
			metadata: types.ChainMetadata{
				AdditionalFields: zeroFieldJSON,
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateChainMetadata(tt.metadata)
			if tt.expectedErr {
				require.Error(t, err, "expected an error for test case: %s", tt.name)
			} else {
				require.NoError(t, err, "expected no error for test case: %s", tt.name)
			}
		})
	}
}

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
