package manifest

import "github.com/smartcontractkit/mcms/pkg/proposal/timelock/types"

type MCMProposalV1 struct {
	BaseProposal

	Spec MCMProposalV1Spec `json:"spec"`
}

type MCMProposalV1Spec struct {
	SpecConfig

	Chains     []Chain     `json:"chains"`
	Operations []Operation `json:"operations"`
}

type TimelockProposalV1 struct {
	BaseProposal

	Spec TimelockProposalV1Spec `json:"spec"`
}

type TimelockProposalV1Spec struct {
	SpecConfig

	// Operation is the type of timelock operation to be performed. Always 'schedule', 'cancel', or 'bypass'
	Action types.TimelockOperationType `json:"action"`

	// i.e. 1d, 1w, 1m, 1y
	MinDelay string `json:"minDelay"`

	Chains     []Chain          `json:"chains"`
	Operations []BatchOperation `json:"operations"`
}
