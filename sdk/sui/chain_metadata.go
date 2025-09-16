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
	AccountObj    string       `json:"account_obj"`
	RegistryObj   string       `json:"registry_obj"`
	TimelockObj   string       `json:"timelock_obj"`
}

func (f AdditionalFieldsMetadata) Validate() error {
	if f.Role > TimelockRoleProposer {
		return errors.New("invalid timelock role")
	}
	if f.McmsPackageID == "" {
		return errors.New("mcms package ID is required")
	}
	if f.AccountObj == "" {
		return errors.New("account object ID is required")
	}
	if f.RegistryObj == "" {
		return errors.New("registry object ID is required")
	}
	if f.TimelockObj == "" {
		return errors.New("timelock object ID is required")
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

func NewChainMetadata(startingOpCount uint64, role TimelockRole, mcmsPackageID string, mcmsObj string, accountObj string, registryObj string, timelockObj string) (types.ChainMetadata, error) {
	if mcmsObj == "" {
		return types.ChainMetadata{}, errors.New("mcms object ID is required")
	}

	additionalFields := AdditionalFieldsMetadata{
		Role:          role,
		McmsPackageID: mcmsPackageID,
		AccountObj:    accountObj,
		RegistryObj:   registryObj,
		TimelockObj:   timelockObj,
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
		MCMAddress:       mcmsObj,
	}, nil
}
