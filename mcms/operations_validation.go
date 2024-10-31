package mcms

import (
	"encoding/json"
	"fmt"

	"github.com/smartcontractkit/mcms/internal/core"
	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
	evm_mcms "github.com/smartcontractkit/mcms/sdk/evm/proposal/mcms"
	"github.com/smartcontractkit/mcms/types"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
)

func ValidateAdditionalFields(operation json.RawMessage, identifier types.ChainSelector) error {
	chainFamily, err := chain_selectors.GetSelectorFamily(uint64(identifier))
	if err != nil {
		return err
	}

	var validator mcms.Validator

	switch chainFamily {
	case chain_selectors.FamilyEVM:
		// Unmarshal and validate for EVM
		var fields evm_mcms.EVMAdditionalFields
		if err := json.Unmarshal(operation, &fields); err != nil {
			return fmt.Errorf("failed to unmarshal EVM additional fields: %w", err)
		}
		validator = fields
	default:
		return core.NewUnknownChainSelectorFamilyError(uint64(identifier), chainFamily)
	}

	// Call Validate on the chain-specific struct
	if err := validator.Validate(); err != nil {
		return err
	}

	return nil
}
