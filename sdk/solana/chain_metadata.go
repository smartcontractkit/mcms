package solana

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/go-playground/validator/v10"

	"github.com/smartcontractkit/mcms/types"
)

type AdditionalFieldsMetadata struct {
	ProposerRoleAccessController  solana.PublicKey `json:"proposerRoleAccessController" validate:"required"`
	CancellerRoleAccessController solana.PublicKey `json:"cancellerRoleAccessController" validate:"required"`
	BypasserRoleAccessController  solana.PublicKey `json:"bypasserRoleAccessController" validate:"required"`
}

func (f AdditionalFieldsMetadata) Validate() error {
	var validate = validator.New()
	if err := validate.Struct(f); err != nil {
		return err
	}
	if f.ProposerRoleAccessController.IsZero() {
		return errors.New("ProposerRoleAccessController cannot be the zero address")
	}
	if f.CancellerRoleAccessController.IsZero() {
		return errors.New("CancellerRoleAccessController cannot be the zero address")
	}
	if f.BypasserRoleAccessController.IsZero() {
		return errors.New("BypasserRoleAccessController cannot be the zero address")
	}

	return nil
}

func ValidateChainMetadata(metadata types.ChainMetadata) error {
	var additionalFields AdditionalFieldsMetadata
	if err := json.Unmarshal(metadata.AdditionalFields, &additionalFields); err != nil {
		return fmt.Errorf("unable to unmarshal additional fields: %w", err)
	}

	if err := additionalFields.Validate(); err != nil {
		return fmt.Errorf("additional fields are invalid: %w", err)
	}

	return nil
}

// NewChainMetadata creates a new ChainMetadata instance for Solana chains
func NewChainMetadata(
	startingOpCount uint64,
	mcmProgramID solana.PublicKey,
	mcmInstanceSeed PDASeed,
	proposerAccessController,
	cancellerAccessController,
	bypasserAccessController solana.PublicKey) (types.ChainMetadata, error) {
	contractID := ContractAddress(mcmProgramID, mcmInstanceSeed)
	additionalFields := AdditionalFieldsMetadata{
		ProposerRoleAccessController:  proposerAccessController,
		CancellerRoleAccessController: cancellerAccessController,
		BypasserRoleAccessController:  bypasserAccessController,
	}
	additionalFieldsJSON, err := json.Marshal(additionalFields)
	if err != nil {
		return types.ChainMetadata{}, err
	}

	return types.ChainMetadata{
		StartingOpCount:  startingOpCount,
		MCMAddress:       contractID,
		AdditionalFields: additionalFieldsJSON,
	}, nil
}

// NewChainMetadataFromTimelock creates a new ChainMetadata from an RPC client
// useful when access controller accounts are not available for the client
func NewChainMetadataFromTimelock(
	ctx context.Context,
	client *rpc.Client,
	startingOpCount uint64,
	mcmProgramID solana.PublicKey,
	mcmSeed PDASeed,
	timelockProgramID solana.PublicKey,
	timelockSeed PDASeed,
) (types.ChainMetadata, error) {
	configPDA, err := FindTimelockConfigPDA(timelockProgramID, timelockSeed)
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("unable to find timelock config pda: %w", err)
	}

	config, err := GetTimelockConfig(ctx, client, configPDA)
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("unable to read timelock config pda: %w", err)
	}

	return NewChainMetadata(startingOpCount, mcmProgramID, mcmSeed,
		config.ProposerRoleAccessController, config.CancellerRoleAccessController,
		config.BypasserRoleAccessController)
}
