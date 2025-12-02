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
			return fmt.Errorf("failed to unmarshal solana additional fields: %w", err)
		}
	}

	return fields.Validate()
}

type AdditionalFields struct {
	Accounts []*solana.AccountMeta `json:"accounts" validate:"omitempty"` //nolint:revive
	Value    *big.Int              `json:"value" validate:"omitempty"`    //nolint:revive
}

// Validate ensures the solana-specific fields are correct
func (f AdditionalFields) Validate() error {
	return validator.New().Struct(f)
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

	return NewTransaction(instruction.ProgramID().String(), data, nil, instruction.Accounts(), contractType, tags)
}
