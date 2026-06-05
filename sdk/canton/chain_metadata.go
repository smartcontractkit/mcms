package canton

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/smartcontractkit/mcms/types"
)

type TimelockRole uint8

func (t TimelockRole) String() string {
	switch t {
	case TimelockRoleBypasser:
		return "Bypasser"
	case TimelockRoleProposer:
		return "Proposer"
	case TimelockRoleCanceller:
		return "Canceller"
	}

	return "unknown"
}

func (t TimelockRole) Byte() uint8 {
	return uint8(t)
}

const (
	TimelockRoleBypasser TimelockRole = iota
	TimelockRoleCanceller
	TimelockRoleProposer
)

func CantonRoleFromAction(action types.TimelockAction) (TimelockRole, error) {
	switch action {
	case types.TimelockActionBypass:
		return TimelockRoleBypasser, nil
	case types.TimelockActionSchedule:
		return TimelockRoleProposer, nil
	case types.TimelockActionCancel:
		return TimelockRoleCanceller, nil
	default:
		return 0, errors.New("unknown timelock action")
	}
}

// AdditionalFieldsMetadata holds Canton fields that must be supplied in chain metadata additionalFields.
// PreOpCount, PostOpCount, and OverridePreviousRoot come from StartingOpCount, proposal tx count / encoder,
// and the proposal's OverridePreviousRoot flag respectively — not from additionalFields.
type AdditionalFieldsMetadata struct {
	ChainId    int64  `json:"chainId"`
	MultisigId string `json:"multisigId"`
	InstanceId string `json:"instanceId,omitempty"` // base instanceId; converter uses for TargetInstanceId in ScheduleBatch etc.
}

func (f AdditionalFieldsMetadata) Validate() error {
	if f.ChainId <= 0 {
		return errors.New("chainId must be positive")
	}
	if f.MultisigId == "" {
		return errors.New("multisigId is required")
	}

	return nil
}

// ValidateChainMetadata validates Canton chain metadata
func ValidateChainMetadata(metadata types.ChainMetadata) error {
	var additionalFields AdditionalFieldsMetadata
	if err := json.Unmarshal(metadata.AdditionalFields, &additionalFields); err != nil {
		return fmt.Errorf("unable to unmarshal additional fields: %w", err)
	}

	if err := additionalFields.Validate(); err != nil {
		return fmt.Errorf("additional fields are invalid: %w", err)
	}

	return nil
}

// NewChainMetadata creates new Canton chain metadata.
// multisigId is "<instanceId>@<party>-<role>" (DAML SetRoot/Op); must match the role used at execution time.
// baseInstanceId is the MCMS contract instanceId; if non-empty, converter uses it for TargetInstanceId in self-dispatch ops.
// mcmsInstanceAddress is the MCMS InstanceAddress hex (32-byte Keccak256 of "instanceId@party"); may be prefixed with "0x".
func NewChainMetadata(
	startingOpCount uint64,
	chainId int64,
	multisigId string,
	mcmsInstanceAddress string,
	baseInstanceId string,
) (types.ChainMetadata, error) {
	if mcmsInstanceAddress == "" {
		return types.ChainMetadata{}, errors.New("MCMS InstanceAddress is required")
	}
	hexStr := strings.TrimPrefix(mcmsInstanceAddress, "0x")
	if len(hexStr) != instanceAddressHexLen {
		return types.ChainMetadata{}, fmt.Errorf("MCMS InstanceAddress hex must be 64 characters (with or without 0x prefix), got %d", len(hexStr))
	}

	additionalFields := AdditionalFieldsMetadata{
		ChainId:    chainId,
		MultisigId: multisigId,
		InstanceId: baseInstanceId,
	}

	if err := additionalFields.Validate(); err != nil {
		return types.ChainMetadata{}, fmt.Errorf("additional fields are invalid: %w", err)
	}

	additionalFieldsBytes, err := json.Marshal(additionalFields)
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("unable to marshal additional fields: %w", err)
	}

	return types.ChainMetadata{
		StartingOpCount:  startingOpCount,
		AdditionalFields: additionalFieldsBytes,
		MCMAddress:       mcmsInstanceAddress,
	}, nil
}
