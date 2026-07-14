package solana

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/gagliardetto/solana-go"
	"github.com/go-playground/validator/v10"

	"github.com/smartcontractkit/mcms/types"
)

const rbacTimelockContractType = "RBACTimelock"

var errNilAccountInAdditionalFields = errors.New("nil account in solana additional fields")

func ValidateAdditionalFields(additionalFields json.RawMessage) error {
	fields, err := ParseAdditionalFields(additionalFields)
	if err != nil {
		return err
	}

	return fields.Validate()
}

type AdditionalFields struct {
	Accounts []*solana.AccountMeta `json:"accounts" validate:"omitempty"`
	Value    *big.Int              `json:"value" validate:"omitempty"`
}

// ParseAdditionalFields unmarshals raw JSON into AdditionalFields and validates
// that no account entry is nil. Returns a zero-value AdditionalFields when raw is empty.
func ParseAdditionalFields(raw json.RawMessage) (AdditionalFields, error) {
	var fields AdditionalFields
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &fields); err != nil {
			return AdditionalFields{}, fmt.Errorf("unable to unmarshal additional fields: %w", err)
		}
	}
	for _, account := range fields.Accounts {
		if account == nil {
			return AdditionalFields{}, errNilAccountInAdditionalFields
		}
	}

	return fields, nil
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
