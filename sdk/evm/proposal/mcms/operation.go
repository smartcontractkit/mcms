package evm

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Validator = EVMAdditionalFields{}

type EVMAdditionalFields struct {
	Value *big.Int `json:"value"`
}

// Validate ensures the EVM-specific fields are correct
func (f EVMAdditionalFields) Validate() error {
	if f.Value == nil || f.Value.Sign() < 0 {
		return fmt.Errorf("invalid EVM value: %v", f.Value)
	}

	return nil
}

func NewEVMOperation(
	to common.Address,
	data []byte,
	value *big.Int,
	contractType string,
	tags []string,
) types.Operation {
	additionalFields := EVMAdditionalFields{
		Value: value,
	}

	marshalledAdditionalFields, err := json.Marshal(additionalFields)
	if err != nil {
		panic(err)
	}

	return types.Operation{
		To:               to.Hex(),
		Data:             data,
		AdditionalFields: marshalledAdditionalFields,
		OperationMetadata: types.OperationMetadata{
			ContractType: contractType,
			Tags:         tags,
		},
	}
}
