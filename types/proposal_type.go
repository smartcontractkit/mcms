package types

type ProposalType string

const (
	// Proposal is a proposal type for the MCMS contract.
	Proposal ProposalType = "Proposal"
	// TimelockProposal is a proposal type for the MCMS contract with RBACTimelock.
	TimelockProposal ProposalType = "TimelockProposal"
)
