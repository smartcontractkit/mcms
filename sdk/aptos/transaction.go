package aptos

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aptos-labs/aptos-go-sdk"

	"github.com/smartcontractkit/mcms/types"
)

func ValidateAdditionalFields(additionalFields json.RawMessage) error {
	fields := AdditionalFields{}
	if err := json.Unmarshal(additionalFields, &fields); err != nil {
		return fmt.Errorf("failed to unmarshal Aptos additional fields: %w", err)
	}

	return fields.Validate()
}

type AdditionalFields struct {
	PackageName string `json:"package_name"`
	ModuleName  string `json:"module_name"`
	Function    string `json:"function"`
}

func (af AdditionalFields) Validate() error {
	if af.PackageName == "" {
		return errors.New("package name is required")
	}
	if len(af.ModuleName) <= 0 || len(af.ModuleName) > 64 {
		return errors.New("module name length must be between 1 and 64 characters")
	}
	if len(af.Function) <= 0 || len(af.Function) > 64 {
		return errors.New("function length must be between 1 and 64 characters")
	}

	return nil
}

// ArgsToData takes the separate encoded arguments returned by the Aptos contract bindings and
// concatenates them into a single []byte of data
func ArgsToData(args [][]byte) []byte {
	return bytes.Join(args, nil)
}

func NewTransaction(packageName, moduleName, function string, to aptos.AccountAddress, data []byte, contractType string, tags []string) (types.Transaction, error) {
	additionalFields := AdditionalFields{
		PackageName: packageName,
		ModuleName:  moduleName,
		Function:    function,
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
		To:               to.StringLong(),
		Data:             data,
		AdditionalFields: marshalledAdditionalFields,
	}, nil
}
