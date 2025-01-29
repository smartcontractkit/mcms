package evm

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/types"
)

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
