package chainsmetadata

import (
	"encoding/json"
	"fmt"

	"github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"
)

func SuiMetadata(chainMetadata types.ChainMetadata) (sui.AdditionalFieldsMetadata, error) {
	var metadata sui.AdditionalFieldsMetadata
	err := json.Unmarshal([]byte(chainMetadata.AdditionalFields), &metadata)
	if err != nil {
		return sui.AdditionalFieldsMetadata{}, fmt.Errorf("error unmarshaling sui chain metadata: %w", err)
	}

	err = metadata.Validate()
	if err != nil {
		return sui.AdditionalFieldsMetadata{}, fmt.Errorf("error validating sui chain metadata: %w", err)
	}

	return metadata, nil
}
