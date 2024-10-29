package proposal

import (
	"github.com/smartcontractkit/mcms/pkg/proposal/mcms"
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
