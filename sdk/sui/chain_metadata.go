package sui

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/smartcontractkit/mcms/types"
)

type TimelockRole uint8

func (t TimelockRole) String() string {
	switch t {
	case TimelockRoleBypasser:
		return "bypasser"
	case TimelockRoleProposer:
		return "proposer"
	case TimelockRoleCanceller:
		return "canceller"
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

type AdditionalFieldsMetadata struct {
	Role          TimelockRole `json:"role"`
	McmsPackageID string       `json:"mcms_package_id"`
}

func (f AdditionalFieldsMetadata) Validate() error {
	if f.Role > TimelockRoleProposer {
		return errors.New("invalid timelock role")
	}
	if f.McmsPackageID == "" {
		return errors.New("mcms package ID is required")
	}

	return nil
}

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
