package aptos

import (
	"errors"
)

type AdditionalFields struct {
	ModuleName string `json:"module_name"`
	Function   string `json:"function"`
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
