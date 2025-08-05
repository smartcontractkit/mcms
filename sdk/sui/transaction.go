package sui

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/smartcontractkit/mcms/types"
)

func ValidateAdditionalFields(additionalFields json.RawMessage) error {
	fields := AdditionalFields{}
	if err := json.Unmarshal(additionalFields, &fields); err != nil {
		return fmt.Errorf("failed to unmarshal Aptos additional fields: %w", err)
	}

	if err := fields.Validate(); err != nil {
		return err
	}

	return nil
}

func (af AdditionalFields) Validate() error {
	if len(af.ModuleName) <= 0 || len(af.ModuleName) > 64 {
		return errors.New("module name length must be between 1 and 64 characters")
	}
	if len(af.Function) <= 0 || len(af.Function) > 64 {
		return errors.New("function length must be between 1 and 64 characters")
	}

	return nil
}

func NewTransaction(moduleName, function string, to string, data []byte, contractType string, tags []string) (types.Transaction, error) {
	additionalFields := AdditionalFields{
		ModuleName: moduleName,
		Function:   function,
	}
	marshalledAdditionalFields, err := json.Marshal(additionalFields)
	if err != nil {
		return types.Transaction{}, fmt.Errorf("failed to marshal additional fields: %w", err)
	}

	return types.Transaction{
		OperationMetadata: types.OperationMetadata{
			ContractType: contractType,
			Tags:         tags,
		},
		To:               to,
		Data:             data,
		AdditionalFields: marshalledAdditionalFields,
	}, nil
}
