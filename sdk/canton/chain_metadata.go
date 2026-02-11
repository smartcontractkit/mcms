package canton

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

// AdditionalFieldsMetadata represents the Canton-specific metadata fields
type AdditionalFieldsMetadata struct {
	ChainId              int64  `json:"chainId"`
	MultisigId           string `json:"multisigId"`
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

// NewChainMetadata creates new Canton chain metadata
func NewChainMetadata(
	preOpCount uint64,
	postOpCount uint64,
	chainId int64,
	multisigId string,
	mcmsContractID string,
	overridePreviousRoot bool,
) (types.ChainMetadata, error) {
	if mcmsContractID == "" {
		return types.ChainMetadata{}, errors.New("MCMS contract ID is required")
	}

	additionalFields := AdditionalFieldsMetadata{
		ChainId:              chainId,
		MultisigId:           multisigId,
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
		StartingOpCount:  preOpCount,
		AdditionalFields: additionalFieldsBytes,
		MCMAddress:       mcmsContractID,
	}, nil
}
