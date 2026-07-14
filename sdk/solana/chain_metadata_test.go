package solana

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/timelock"

	"github.com/smartcontractkit/mcms/sdk/solana/mocks"
	"github.com/smartcontractkit/mcms/types"
)

func validAdditionalFields(t *testing.T) AdditionalFieldsMetadata {
	t.Helper()
	return AdditionalFieldsMetadata{
		ProposerRoleAccessController:  solana.NewWallet().PublicKey(),
		CancellerRoleAccessController: solana.NewWallet().PublicKey(),
		BypasserRoleAccessController:  solana.NewWallet().PublicKey(),
	}
}

func TestNewChainMetadataFromTimelock(t *testing.T) {
	t.Parallel()

	programID := solana.NewWallet().PublicKey()
	timelockProgramID := solana.NewWallet().PublicKey()
	mcmSeed := PDASeed([32]byte{1, 2, 3, 4})
	timelockSeed := PDASeed([32]byte{1, 2, 3, 4})

	configPDA, err := FindTimelockConfigPDA(timelockProgramID, timelockSeed)
	require.NoError(t, err)

	newClient := func(t *testing.T, rpcErr error) *rpc.Client {
		t.Helper()
		jsonRPC := mocks.NewJSONRPCClient(t)
		mockGetAccountInfo(t, jsonRPC, configPDA, &timelock.Config{}, rpcErr)
		return rpc.NewWithCustomRPCClient(jsonRPC)
	}

	t.Run("returns metadata from timelock config", func(t *testing.T) {
		t.Parallel()
		metadata, err := NewChainMetadataFromTimelock(
			t.Context(), newClient(t, nil), 100, programID, mcmSeed, timelockProgramID, timelockSeed)
		require.NoError(t, err)
		require.Equal(t, uint64(100), metadata.StartingOpCount)
		require.Equal(t, ContractAddress(programID, mcmSeed), metadata.MCMAddress)
	})

	t.Run("wraps RPC errors", func(t *testing.T) {
		t.Parallel()
		_, err := NewChainMetadataFromTimelock(
			t.Context(), newClient(t, errors.New("rpc error")), 100, programID, mcmSeed, timelockProgramID, timelockSeed)
		require.EqualError(t, err, "unable to read timelock config pda: rpc error")
	})
}

func TestAdditionalFieldsMetadata_ExecutePayer(t *testing.T) {
	t.Parallel()

	base := validAdditionalFields(t)
	payer := solana.NewWallet().PublicKey()

	t.Run("WithExecutePayer returns copy without mutating original", func(t *testing.T) {
		t.Parallel()
		updated := base.WithExecutePayer(payer)
		require.Nil(t, base.ExecutePayer)
		require.True(t, updated.ExecutePayer.Equals(payer))
		require.True(t, updated.ProposerRoleAccessController.Equals(base.ProposerRoleAccessController))
	})

	t.Run("HasExecutePayer is false for nil and zero key, true when set", func(t *testing.T) {
		t.Parallel()
		require.False(t, base.HasExecutePayer())

		zero := solana.PublicKey{}
		withZero := base
		withZero.ExecutePayer = &zero
		require.False(t, withZero.HasExecutePayer())

		require.True(t, base.WithExecutePayer(payer).HasExecutePayer())
	})

	t.Run("JSON round-trips executePayer, omits when nil", func(t *testing.T) {
		t.Parallel()

		raw, err := json.Marshal(base)
		require.NoError(t, err)
		require.NotContains(t, string(raw), "executePayer")

		raw, err = json.Marshal(base.WithExecutePayer(payer))
		require.NoError(t, err)

		var roundTrip AdditionalFieldsMetadata
		require.NoError(t, json.Unmarshal(raw, &roundTrip))
		require.True(t, roundTrip.ExecutePayer.Equals(payer))
	})
}

func TestAdditionalFieldsMetadata_Validate(t *testing.T) {
	t.Parallel()

	require.NoError(t, validAdditionalFields(t).Validate(), "all valid keys")
	require.NoError(t, validAdditionalFields(t).WithExecutePayer(solana.NewWallet().PublicKey()).Validate(), "with execute payer")

	for _, field := range []string{"ProposerRoleAccessController", "CancellerRoleAccessController", "BypasserRoleAccessController"} {
		t.Run("rejects zero "+field, func(t *testing.T) {
			t.Parallel()
			fields := validAdditionalFields(t)
			switch field {
			case "ProposerRoleAccessController":
				fields.ProposerRoleAccessController = solana.PublicKey{}
			case "CancellerRoleAccessController":
				fields.CancellerRoleAccessController = solana.PublicKey{}
			case "BypasserRoleAccessController":
				fields.BypasserRoleAccessController = solana.PublicKey{}
			}
			require.ErrorContains(t, fields.Validate(), field)
		})
	}
}

func TestValidateChainMetadata(t *testing.T) {
	t.Parallel()

	raw, err := json.Marshal(validAdditionalFields(t))
	require.NoError(t, err)
	require.NoError(t, ValidateChainMetadata(types.ChainMetadata{AdditionalFields: raw}), "valid fields")

	require.ErrorContains(t, ValidateChainMetadata(types.ChainMetadata{AdditionalFields: []byte("bad")}), "unable to unmarshal")

	invalid := validAdditionalFields(t)
	invalid.ProposerRoleAccessController = solana.PublicKey{}
	raw, err = json.Marshal(invalid)
	require.NoError(t, err)
	require.ErrorContains(t, ValidateChainMetadata(types.ChainMetadata{AdditionalFields: raw}), "additional fields are invalid")
}

func TestNewChainMetadata(t *testing.T) {
	t.Parallel()

	proposer := solana.NewWallet().PublicKey()
	canceller := solana.NewWallet().PublicKey()
	bypasser := solana.NewWallet().PublicKey()
	programID := solana.NewWallet().PublicKey()
	seed := PDASeed([32]byte{1, 2, 3, 4})

	metadata, err := NewChainMetadata(100, programID, seed, proposer, canceller, bypasser)
	require.NoError(t, err)
	require.Equal(t, uint64(100), metadata.StartingOpCount)
	require.Equal(t, ContractAddress(programID, seed), metadata.MCMAddress)

	var additional AdditionalFieldsMetadata
	require.NoError(t, json.Unmarshal(metadata.AdditionalFields, &additional))
	require.True(t, additional.ProposerRoleAccessController.Equals(proposer))
	require.True(t, additional.CancellerRoleAccessController.Equals(canceller))
	require.True(t, additional.BypasserRoleAccessController.Equals(bypasser))
}
