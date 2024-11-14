package mcms

import (
	"encoding/json"
	"fmt"

	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

func ValidateAdditionalFields(operation json.RawMessage, csel types.ChainSelector) error {
	chainFamily, err := types.GetChainSelectorFamily(csel)
	if err != nil {
		return err
	}

	var validator sdk.Validator

	switch chainFamily {
	case cselectors.FamilyEVM:
		// Unmarshal and validate for EVM
		var fields evm.AdditionalFields
		if err := json.Unmarshal(operation, &fields); err != nil {
			return fmt.Errorf("failed to unmarshal EVM additional fields: %w", err)
		}
		validator = fields
	}

	// Call Validate on the chain-specific struct
	if err := validator.Validate(); err != nil {
		return err
	}

	return nil
}
