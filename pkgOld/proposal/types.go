package proposal

import (
	"errors"

	"github.com/smartcontractkit/mcms/pkgOld/proposal/mcms"
	"github.com/smartcontractkit/mcms/pkgOld/proposal/timelock"
)

var StringToProposalType = map[string]mcms.ProposalType{
	"MCMS":             mcms.MCMS,
	"MCMSWithTimelock": mcms.MCMSWithTimelock,
}

type Proposal interface {
	ToExecutor(sim bool) (*mcms.Executor, error)
	AddSignature(signature mcms.Signature)
	Validate() error
}

func LoadProposal(proposalType mcms.ProposalType, filePath string) (Proposal, error) {
	switch proposalType {
	case mcms.MCMS:
		return mcms.NewProposalFromFile(filePath)
	case mcms.MCMSWithTimelock:
		return timelock.NewMCMSWithTimelockProposalFromFile(filePath)
	default:
		return nil, errors.New("unknown proposal type")
	}
}
