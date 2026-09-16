package stellar

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/smartcontractkit/mcms/types"
)

// chainMetadataAdditionalFields is the optional chain-metadata shape accepted
// by the Stellar encoder's HashMetadata: both fields may be omitted, but no
// other keys are allowed.
type chainMetadataAdditionalFields struct {
	ConfigVersion   *uint64 `json:"configVersion"`
	EncodingVersion *uint32 `json:"encodingVersion"`
}

// ValidateAdditionalFields validates a Stellar MCMS transaction's additional fields.
func ValidateAdditionalFields(additionalFields json.RawMessage) error {
	_, err := decodeTransactionAdditionalFields(additionalFields)

	return err
}

// ValidateChainMetadata validates Stellar chain metadata. Chain-level
// AdditionalFields are optional (HashMetadata tolerates their absence and
// defaults configVersion to 1); when present they must decode to the
// documented shape without unknown keys, and a present encodingVersion must
// match the current encoding version. MCMAddress must be a valid Stellar
// contract address (strkey C... or 64-hex).
func ValidateChainMetadata(metadata types.ChainMetadata) error {
	if !IsAddress(metadata.MCMAddress) {
		return fmt.Errorf("invalid Stellar MCMAddress %q", metadata.MCMAddress)
	}

	if len(metadata.AdditionalFields) == 0 {
		return nil
	}

	var fields chainMetadataAdditionalFields
	decoder := json.NewDecoder(bytes.NewReader(metadata.AdditionalFields))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fields); err != nil {
		return fmt.Errorf("decode Stellar chain metadata additional fields: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}

		return fmt.Errorf("decode Stellar chain metadata additional fields: %w", err)
	}
	if fields.EncodingVersion != nil && *fields.EncodingVersion != encodingVersion {
		return fmt.Errorf("%w: %d", ErrUnsupportedEncodingVersion, *fields.EncodingVersion)
	}

	return nil
}
