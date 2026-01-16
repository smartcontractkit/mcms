package sui

import (
	"encoding/json"
	"fmt"

	"github.com/smartcontractkit/mcms/types"
)

func SuiMetadata(chainMetadata types.ChainMetadata) (AdditionalFieldsMetadata, error) {
	var metadata AdditionalFieldsMetadata
	err := json.Unmarshal([]byte(chainMetadata.AdditionalFields), &metadata)
	if err != nil {
		return AdditionalFieldsMetadata{}, fmt.Errorf("error unmarshaling sui chain metadata: %w", err)
	}

	err = metadata.Validate()
	if err != nil {
		return AdditionalFieldsMetadata{}, fmt.Errorf("error validating sui chain metadata: %w", err)
	}

	return metadata, nil
}
