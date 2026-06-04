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
		return errors.New("Canton additional fields are required")
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

	if f.OperationData != "" {
		if strings.HasPrefix(f.OperationData, "0x") || strings.HasPrefix(f.OperationData, "0X") {
			return errors.New("operationData must be hex-encoded without 0x prefix")
		}
		if len(f.OperationData)%2 != 0 {
			return errors.New("operationData must be hex-encoded with an even number of digits")
		}
		if _, err := hex.DecodeString(f.OperationData); err != nil {
			return fmt.Errorf("operationData must be hex-encoded: %w", err)
		}
		if f.TargetInstanceAddress == "" {
			return errors.New("targetInstanceAddress is required when operationData is set")
		}
		if f.FunctionName == "" {
			return errors.New("functionName is required when operationData is set")
		}
	}

	return nil
}
