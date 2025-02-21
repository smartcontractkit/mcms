package mcms

import (
	"encoding/json"
	"fmt"

	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

func ValidateAdditionalFieldsMetadata(additionalFields json.RawMessage, csel types.ChainSelector) error {
	chainFamily, err := types.GetChainSelectorFamily(csel)
	if err != nil {
		return err
	}

	var validator sdk.Validator

	// For now only solana contains additional metadata fields. EVM just needs the address
	switch chainFamily {
	case cselectors.FamilySolana:
		fields := solana.AdditionalFieldsMetadata{}
		if len(additionalFields) != 0 {
			if err := json.Unmarshal(additionalFields, &fields); err != nil {
				return fmt.Errorf("failed to unmarshal Solana additional fields: %w", err)
			}
		}
		validator = fields
	}

	// Call Validate on the chain-specific struct
	if err := validator.Validate(); err != nil {
		return err
	}

	return nil
}
