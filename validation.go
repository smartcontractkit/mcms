package mcms

import (
	"encoding/json"
	"fmt"

	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"
)

func validateAdditionalFields(additionalFields json.RawMessage, csel types.ChainSelector) error {
	chainFamily, err := types.GetChainSelectorFamily(csel)
	if err != nil {
		return err
	}

	switch chainFamily {
	case cselectors.FamilyEVM:
		return evm.ValidateAdditionalFields(additionalFields)

	case cselectors.FamilySolana:
		return solana.ValidateAdditionalFields(additionalFields)

	case cselectors.FamilyAptos:
		return aptos.ValidateAdditionalFields(additionalFields)

	case cselectors.FamilySui:
		return sui.ValidateAdditionalFields(additionalFields)
	}

	return nil
}

// validateChainMetadata validates the chain metadata for the given chain selector
func validateChainMetadata(metadata types.ChainMetadata, csel types.ChainSelector) error {
	chainFamily, err := types.GetChainSelectorFamily(csel)
	if err != nil {
		return fmt.Errorf("unable to get chain selector family: %w", err)
	}

	switch chainFamily {
	case cselectors.FamilySolana:
		return solana.ValidateChainMetadata(metadata)
	case cselectors.FamilyEVM:
		return nil
	case cselectors.FamilyAptos:
		return nil
	case cselectors.FamilySui:
		return sui.ValidateChainMetadata(metadata)
	default:
		return fmt.Errorf("unsupported chain family: %s", chainFamily)
	}
}
