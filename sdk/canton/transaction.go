package canton

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func ValidateAdditionalFields(additionalFields json.RawMessage) error {
	if len(additionalFields) == 0 {
		return nil
	}

	var fields AdditionalFields
	if err := json.Unmarshal(additionalFields, &fields); err != nil {
		return fmt.Errorf("failed to unmarshal Canton additional fields: %w", err)
	}

	return fields.Validate()
}

func (f AdditionalFields) Validate() error {
	if f.TargetInstanceAddress != "" && !strings.Contains(f.TargetInstanceAddress, "@") {
		return errors.New("targetInstanceAddress must be in instanceId@partyId format")
	}

	if f.OperationData != "" {
		if _, err := hex.DecodeString(strings.TrimPrefix(f.OperationData, "0x")); err != nil {
			return fmt.Errorf("operationData must be hex-encoded: %w", err)
		}
	}

	return nil
}
