package solana

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/gagliardetto/solana-go"
	"github.com/go-playground/validator/v10"

	"github.com/smartcontractkit/mcms/types"
)

func ValidateAdditionalFields(additionalFields json.RawMessage) error {
	fields := AdditionalFields{
		Value: big.NewInt(0),
	}
	if len(additionalFields) != 0 {
		if err := json.Unmarshal(additionalFields, &fields); err != nil {
			return fmt.Errorf("failed to unmarshal EVM additional fields: %w", err)
		}
	}

	if err := fields.Validate(); err != nil {
		return err
	}

	return nil
}

type AdditionalFields struct {
	Accounts []*solana.AccountMeta `json:"accounts" validate:"required"`
	Value    *big.Int              `json:"value" validate:"omitempty"`
}

// Validate ensures the solana-specific fields are correct
func (f AdditionalFields) Validate() error {
	var validate = validator.New()
	if err := validate.Struct(f); err != nil {
		return err
	}

	return nil
}

func NewTransaction(
	contractID string,
	data []byte,
	value *big.Int,
	accounts []*solana.AccountMeta,
	contractType string,
	tags []string,
) (types.Transaction, error) {
	key, err := solana.PublicKeyFromBase58(contractID)
	if err != nil {
		return types.Transaction{}, err
	}

	marshalledAdditionalFields, err := json.Marshal(AdditionalFields{Accounts: accounts, Value: value})
	if err != nil {
		return types.Transaction{}, fmt.Errorf("unable to marshal additional fields: %w", err)
	}

	return types.Transaction{
		To:               key.String(),
		Data:             data,
		AdditionalFields: marshalledAdditionalFields,
		OperationMetadata: types.OperationMetadata{
			ContractType: contractType,
			Tags:         tags,
		},
	}, nil
}

func NewTransactionFromInstruction(
	instruction solana.Instruction,
	contractType string,
	tags []string,
) (types.Transaction, error) {
	data, err := instruction.Data()
	if err != nil {
		return types.Transaction{}, fmt.Errorf("unable to get instruction data: %w", err)
	}

	// ensure PDAs don't have the "IsSigner" flag
	for _, account := range instruction.Accounts() {
		if account != nil && !solana.IsOnCurve(account.PublicKey[:]) {
			account.IsSigner = false
		}
	}

	return NewTransaction(instruction.ProgramID().String(), data, nil, instruction.Accounts(), contractType, tags)
}
