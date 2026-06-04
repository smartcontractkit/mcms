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
		return errors.New("canton additional fields are required")
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

	if f.TargetInstanceAddress != "" && f.FunctionName == "" {
		return errors.New("functionName is required when targetInstanceAddress is set")
	}

	if f.FunctionName != "" && f.TargetInstanceAddress == "" {
		return errors.New("targetInstanceAddress is required when functionName is set")
	}

	return nil
}

// operationDataHex returns hex-encoded wire bytes from tx.Data for Canton hashing and ledger transport.
func operationDataHex(data []byte) string {
	return hex.EncodeToString(data)
}
