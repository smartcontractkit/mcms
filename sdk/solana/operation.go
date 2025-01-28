package solana

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/gagliardetto/solana-go"

	"github.com/smartcontractkit/mcms/types"
)

type AdditionalFields struct {
	Accounts []*solana.AccountMeta `json:"accounts" validate:"required"`
	Value    *big.Int              `json:"value" validate:"omitempty"`
}

// Validate ensures the solana-specific fields are correct
func (f AdditionalFields) Validate() error {
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
