package types //nolint:revive

type ProposalKind string

const (
	// KindProposal is a proposal type for the MCMS contract.
	KindProposal ProposalKind = "Proposal"
	// KindTimelockProposal is a proposal type for the MCMS contract with RBACTimelock.
	KindTimelockProposal ProposalKind = "TimelockProposal"
)

// StringToProposalKind converts a string to a ProposalKind.
var StringToProposalKind = map[string]ProposalKind{
	"Proposal":         KindProposal,
	"TimelockProposal": KindTimelockProposal,
}
