package sui

import (
	"encoding/json"
	"errors"
	"fmt"

	suibindings "github.com/smartcontractkit/chainlink-sui/bindings"

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

var NewCCIPEntrypointArgEncoder = suibindings.NewCCIPEntrypointArgEncoder

func SuiRoleFromAction(action types.TimelockAction) (TimelockRole, error) {
	switch action {
	case types.TimelockActionBypass:
		return TimelockRoleBypasser, nil
	case types.TimelockActionSchedule:
		return TimelockRoleProposer, nil
	case types.TimelockActionCancel:
		return TimelockRoleCanceller, nil
	default:
		return 0, errors.New("unknown timelock action")
	}
}
