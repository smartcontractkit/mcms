package mcms

import (
	"encoding/json"
	"fmt"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/sdk/ton"
)

func validateAdditionalFields(additionalFields json.RawMessage, csel types.ChainSelector) error {
	chainFamily, err := types.GetChainSelectorFamily(csel)
	if err != nil {
		return err
	}

	switch chainFamily {
	case chainsel.FamilyEVM:
		return evm.ValidateAdditionalFields(additionalFields)

	case chainsel.FamilySolana:
		return solana.ValidateAdditionalFields(additionalFields)

	case chainsel.FamilyAptos:
		return aptos.ValidateAdditionalFields(additionalFields)

	case chainsel.FamilySui:
		return sui.ValidateAdditionalFields(additionalFields)

	case chainsel.FamilyTon:
		return ton.ValidateAdditionalFields(additionalFields)
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
	case chainsel.FamilySolana:
		return solana.ValidateChainMetadata(metadata)
	case chainsel.FamilyEVM:
		return nil
	case chainsel.FamilyAptos:
		return nil
	case chainsel.FamilySui:
		return sui.ValidateChainMetadata(metadata)
	case chainsel.FamilyTon:
		// TODO (ton): do we need special chain metadata for TON?
		// Yes! We could attach MCMS -> Timelock value here which is now hardcoded default in timelock converter
		return nil
	default:
		return fmt.Errorf("unsupported chain family: %s", chainFamily)
	}
}
