package mcms

import (
	"encoding/json"
	"fmt"
	"math/big"

	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

func ValidateAdditionalFields(additionalFields json.RawMessage, csel types.ChainSelector) error {
	chainFamily, err := types.GetChainSelectorFamily(csel)
	if err != nil {
		return err
	}

	var validator sdk.Validator

	switch chainFamily {
	case cselectors.FamilyEVM:
		// Unmarshal and validate for EVM
		fields := evm.AdditionalFields{
			Value: big.NewInt(0),
		}

		if len(additionalFields) != 0 {
			if err := json.Unmarshal(additionalFields, &fields); err != nil {
				return fmt.Errorf("failed to unmarshal EVM additional fields: %w", err)
			}
		}

		validator = fields
	case cselectors.FamilySolana:
		fields := evm.AdditionalFields{Value: big.NewInt(0)}
		validator = fields
	}

	// Call Validate on the chain-specific struct
	if err := validator.Validate(); err != nil {
		return err
	}

	return nil
}
