package stellar

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"

	"github.com/smartcontractkit/mcms/types"
)

// ValidateAdditionalFields validates JSON in types.Transaction.AdditionalFields
// (optional StellarOp.value as 32-byte hex; see [Encoder] parseValueWord).
func ValidateAdditionalFields(additionalFields json.RawMessage) error {
	_, err := parseValueWord(additionalFields)
	if err != nil {
		return fmt.Errorf("stellar additional fields: %w", err)
	}

	return nil
}

// ValidateChainMetadata ensures MCMAddress parses as a Stellar contract id (strkey or 32-byte hex).
func ValidateChainMetadata(metadata types.ChainMetadata) error {
	if strings.TrimSpace(metadata.MCMAddress) == "" {
		return fmt.Errorf("mcm address is required")
	}

	if _, err := parseContractID(metadata.MCMAddress); err != nil {
		return fmt.Errorf("mcmAddress: %w", err)
	}

	return nil
}

// ValidateTimelockChainMetadata validates Stellar chain metadata for a timelock proposal:
// MCM contract id, timelock role JSON in AdditionalFields, and action-specific required callers.
// timelockExecutor is always required (see chainwrappers.BuildTimelockExecutor). Optional role
// fields are checked when non-empty.
func ValidateTimelockChainMetadata(metadata types.ChainMetadata, action types.TimelockAction) error {
	if err := ValidateChainMetadata(metadata); err != nil {
		return err
	}

	af, err := ParseTimelockProposalAdditionalFields(metadata.AdditionalFields)
	if err != nil {
		return err
	}

	if err := validateTimelockRoleAddress("timelockExecutor", af.TimelockExecutor, true); err != nil {
		return err
	}

	switch action {
	case types.TimelockActionSchedule:
		if err := validateTimelockRoleAddress("timelockProposer", af.TimelockProposer, true); err != nil {
			return err
		}
	case types.TimelockActionCancel:
		if err := validateTimelockRoleAddress("timelockCanceller", af.TimelockCanceller, true); err != nil {
			return err
		}
	case types.TimelockActionBypass:
		if err := validateTimelockRoleAddress("timelockBypasser", af.TimelockBypasser, true); err != nil {
			return err
		}
	default:
		return fmt.Errorf("stellar timelock: invalid timelock action: %s", action)
	}

	if err := validateTimelockRoleAddress("timelockAdmin", af.TimelockAdmin, false); err != nil {
		return err
	}
	switch action {
	case types.TimelockActionSchedule:
		if err := validateTimelockRoleAddress("timelockCanceller", af.TimelockCanceller, false); err != nil {
			return err
		}
		if err := validateTimelockRoleAddress("timelockBypasser", af.TimelockBypasser, false); err != nil {
			return err
		}
	case types.TimelockActionCancel:
		if err := validateTimelockRoleAddress("timelockProposer", af.TimelockProposer, false); err != nil {
			return err
		}
		if err := validateTimelockRoleAddress("timelockBypasser", af.TimelockBypasser, false); err != nil {
			return err
		}
	case types.TimelockActionBypass:
		if err := validateTimelockRoleAddress("timelockProposer", af.TimelockProposer, false); err != nil {
			return err
		}
		if err := validateTimelockRoleAddress("timelockCanceller", af.TimelockCanceller, false); err != nil {
			return err
		}
	}

	return nil
}

func validateTimelockRoleAddress(field, addr string, required bool) error {
	if strings.TrimSpace(addr) == "" {
		if required {
			return fmt.Errorf("stellar timelock: %s is required in chain metadata additionalFields", field)
		}
		return nil
	}
	if scval.ParseAddress(addr) == nil {
		return fmt.Errorf("stellar timelock: %s: invalid Stellar address (expect G... account or C... contract)", field)
	}
	return nil
}
