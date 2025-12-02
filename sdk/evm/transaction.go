package evm

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

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

	return fields.Validate()
}

type AdditionalFields struct {
	Value *big.Int `json:"value"`
}

// Validate ensures the EVM-specific fields are correct
func (f AdditionalFields) Validate() error {
	if f.Value == nil || f.Value.Sign() < 0 {
		return fmt.Errorf("invalid EVM value: %v", f.Value)
	}

	return nil
}

func NewTransaction(
	to common.Address,
	data []byte,
	value *big.Int,
	contractType string,
	tags []string,
) types.Transaction {
	additionalFields := AdditionalFields{
		Value: value,
	}

	marshalledAdditionalFields, err := json.Marshal(additionalFields)
	if err != nil {
		panic(err)
	}

	return types.Transaction{
		To:               to.Hex(),
		Data:             data,
		AdditionalFields: marshalledAdditionalFields,
		OperationMetadata: types.OperationMetadata{
			ContractType: contractType,
			Tags:         tags,
		},
	}
}
