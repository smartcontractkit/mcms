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

// AdditionalFieldsMetadata represents the Canton-specific metadata fields.
// MultisigId must be makeMcmsId(instanceId, role) e.g. "mcms-001-proposer" (DAML SetRoot/ExecuteOp).
// InstanceId is the base MCMS instanceId for self-dispatch TargetInstanceId (DAML E_NOT_SELF_DISPATCH).
type AdditionalFieldsMetadata struct {
	ChainId              int64  `json:"chainId"`
	MultisigId           string `json:"multisigId"`
	InstanceId           string `json:"instanceId,omitempty"` // base instanceId; converter uses for TargetInstanceId in ScheduleBatch etc.
	PreOpCount           uint64 `json:"preOpCount"`
	PostOpCount          uint64 `json:"postOpCount"`
	OverridePreviousRoot bool   `json:"overridePreviousRoot"`
}

func (f AdditionalFieldsMetadata) Validate() error {
	if f.ChainId == 0 {
		return errors.New("chainId is required")
	}
	if f.MultisigId == "" {
		return errors.New("multisigId is required")
	}
	if f.PostOpCount < f.PreOpCount {
		return errors.New("postOpCount must be >= preOpCount")
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
// multisigId must be makeMcmsId(instanceId, role) e.g. "mcms-001-proposer" (DAML expects this in SetRoot/Op).
// baseInstanceId is the MCMS contract instanceId; if non-empty, converter uses it for TargetInstanceId in self-dispatch ops.
// mcmsInstanceAddress is the MCMS InstanceAddress hex (32-byte Keccak256 of "instanceId@party"); may be prefixed with "0x".
func NewChainMetadata(
	preOpCount uint64,
	postOpCount uint64,
	chainId int64,
	multisigId string,
	mcmsInstanceAddress string,
	overridePreviousRoot bool,
	baseInstanceId string,
) (types.ChainMetadata, error) {
	if mcmsInstanceAddress == "" {
		return types.ChainMetadata{}, errors.New("MCMS InstanceAddress is required")
	}
	hexStr := strings.TrimPrefix(mcmsInstanceAddress, "0x")
	if len(hexStr) != 64 {
		return types.ChainMetadata{}, fmt.Errorf("MCMS InstanceAddress hex must be 64 characters (with or without 0x prefix), got %d", len(hexStr))
	}

	additionalFields := AdditionalFieldsMetadata{
		ChainId:              chainId,
		MultisigId:           multisigId,
		InstanceId:           baseInstanceId,
		PreOpCount:           preOpCount,
		PostOpCount:          postOpCount,
		OverridePreviousRoot: overridePreviousRoot,
	}

	if err := additionalFields.Validate(); err != nil {
		return types.ChainMetadata{}, fmt.Errorf("additional fields are invalid: %w", err)
	}

	additionalFieldsBytes, err := json.Marshal(additionalFields)
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("unable to marshal additional fields: %w", err)
	}

	return types.ChainMetadata{
		StartingOpCount:   preOpCount,
		AdditionalFields: additionalFieldsBytes,
		MCMAddress:        mcmsInstanceAddress,
	}, nil
}
