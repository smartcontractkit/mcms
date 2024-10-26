package evm

import (
	"encoding/json"
	"math/big"

	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
)

type EVMAdditionalFields struct {
	Value *big.Int `json:"value"`
}

func NewEVMOperation(
	to string,
	data []byte,
	value *big.Int,
	contractType string,
	tags []string,
) mcms.Operation {
	additionalFields := EVMAdditionalFields{
		Value: value,
	}

	marshalledAdditionalFields, _ := json.Marshal(additionalFields)
	return mcms.Operation{
		To:               to,
		Data:             data,
		AdditionalFields: marshalledAdditionalFields,
		OperationMetadata: mcms.OperationMetadata{
			ContractType: contractType,
			Tags:         tags,
		},
	}
}
