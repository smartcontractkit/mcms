package solana

import (
	"encoding/json"
	"fmt"

	"github.com/gagliardetto/solana-go"

	"github.com/smartcontractkit/mcms/types"
)

type AdditionalFields struct {
	Accounts []*solana.AccountMeta `json:"accounts" validate:"required"`
}

// Validate ensures the solana-specific fields are correct
func (f AdditionalFields) Validate() error {
	return nil
}

func NewTransaction(
	contractID string,
	data []byte,
	accounts []*solana.AccountMeta,
	contractType string,
	tags []string,
) (types.Transaction, error) {
	key, err := solana.PublicKeyFromBase58(contractID)
	if err != nil {
		return types.Transaction{}, err
	}
	additionalFields := AdditionalFields{
		Accounts: accounts,
	}

	marshalledAdditionalFields, err := json.Marshal(additionalFields)
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
