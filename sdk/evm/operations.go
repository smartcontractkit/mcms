package evm

import (
	"fmt"
	"math/big"
)

// OperationFieldsEVM EVM-specific operation fields. Implements the sdk.Validator interface
type OperationFieldsEVM struct {
	Value *big.Int `json:"value"`
}

// Validate ensures the EVM-specific fields are correct
func (o OperationFieldsEVM) Validate() error {
	if o.Value == nil || o.Value.Sign() < 0 {
		return fmt.Errorf("invalid EVM value: %v", o.Value)
	}
	return nil
}
