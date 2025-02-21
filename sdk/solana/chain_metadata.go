package solana

import (
	"encoding/json"
	"errors"

	"github.com/gagliardetto/solana-go"
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

// NewSolanaChainMetadata creates a new ChainMetadata instance for Solana chains
func NewSolanaChainMetadata(
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
