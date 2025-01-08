package solana

import (
	"encoding/json"

	"github.com/gagliardetto/solana-go"

	"github.com/smartcontractkit/mcms/types"
)

type AdditionalFields struct {
	Accounts []solana.AccountMeta `json:"accounts" validate:"required"`
}

// Validate ensures the solana-specific fields are correct
func (f AdditionalFields) Validate() error {
	return nil
}

func NewTransaction(
	contractID string,
	data []byte,
	accounts []solana.AccountMeta,
	contractType string,
	tags []string,
) types.Transaction {
	additionalFields := AdditionalFields{
		Accounts: accounts,
	}

	marshalledAdditionalFields, err := json.Marshal(additionalFields)
	if err != nil {
		panic(err)
	}

	return types.Transaction{
		To:               contractID,
		Data:             data,
		AdditionalFields: marshalledAdditionalFields,
		OperationMetadata: types.OperationMetadata{
			ContractType: contractType,
			Tags:         tags,
		},
	}
}
