package mcms

import (
	"encoding/json"
	"fmt"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk/evm"
)

// Validator interface used to validate the fields of a chain operation across different chains.
type Validator interface {
	Validate() error
}

func ValidateAdditionalFields(operation json.RawMessage, identifier ChainIdentifier) error {
	chainFamily, err := chain_selectors.GetSelectorFamily(uint64(identifier))
	if err != nil {
		return err
	}

	var validator Validator

	switch chainFamily {
	case chain_selectors.FamilyEVM:
		// Unmarshal and validate for EVM
		var fields evm.OperationFieldsEVM
		if err := json.Unmarshal(operation, &fields); err != nil {
			return fmt.Errorf("failed to unmarshal EVM additional fields: %w", err)
		}
		validator = fields

	case chain_selectors.FamilySolana:
		// Solana struct and validation
		// Example: validator = solanaFields
		panic("not implemented")

	default:
		return &UnkownChainSelectorFamilyError{
			ChainFamily:   chainFamily,
			ChainSelector: uint64(identifier),
		}
	}

	// Call Validate on the chain-specific struct
	if err := validator.Validate(); err != nil {
		return err
	}

	return nil
}
