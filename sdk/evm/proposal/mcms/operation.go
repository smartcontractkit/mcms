package evm

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
)

type EVMAdditionalFields struct {
	Value *big.Int `json:"value"`
}

func NewEVMOperation(
	to common.Address,
	data []byte,
	value *big.Int,
	contractType string,
	tags []string,
) mcms.Operation {
	additionalFields := EVMAdditionalFields{
		Value: value,
	}

	marshalledAdditionalFields, err := json.Marshal(additionalFields)
	if err != nil {
		panic(err)
	}

	return mcms.Operation{
		To:               to.Hex(),
		Data:             data,
		AdditionalFields: marshalledAdditionalFields,
		OperationMetadata: mcms.OperationMetadata{
			ContractType: contractType,
			Tags:         tags,
		},
	}
}

// Validate ensures the EVM-specific fields are correct
func (o EVMAdditionalFields) Validate() error {
	if o.Value == nil || o.Value.Sign() < 0 {
		return fmt.Errorf("invalid EVM value: %v", o.Value)
	}

	return nil
}
