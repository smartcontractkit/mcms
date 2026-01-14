package utils

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"
)

func SuiMetadataFromProposal(selector types.ChainSelector, proposal *mcms.TimelockProposal) (sui.AdditionalFieldsMetadata, error) {
	if proposal == nil {
		return sui.AdditionalFieldsMetadata{}, errors.New("sui timelock proposal is needed")
	}

	var metadata sui.AdditionalFieldsMetadata
	err := json.Unmarshal([]byte(proposal.ChainMetadata[selector].AdditionalFields), &metadata)
	if err != nil {
		return sui.AdditionalFieldsMetadata{}, fmt.Errorf("error unmarshaling sui chain metadata: %w", err)
	}

	err = metadata.Validate()
	if err != nil {
		return sui.AdditionalFieldsMetadata{}, fmt.Errorf("error validating sui chain metadata: %w", err)
	}

	return metadata, nil
}
