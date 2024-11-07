package types

type ProposalKind string

const (
	// KindProposal is a proposal type for the MCMS contract.
	KindProposal ProposalKind = "Proposal"
	// KindTimelockProposal is a proposal type for the MCMS contract with RBACTimelock.
	KindTimelockProposal ProposalKind = "TimelockProposal"
)
