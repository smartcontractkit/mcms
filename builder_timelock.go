package mcms

import "github.com/smartcontractkit/mcms/types"

// TimelockProposalBuilder builder for timelock proposals types.
type TimelockProposalBuilder struct {
	BaseProposalBuilder[*TimelockProposalBuilder]
	proposal TimelockProposal
}

// NewTimelockProposalBuilder creates a new TimelockProposalBuilder.
func NewTimelockProposalBuilder() *TimelockProposalBuilder {
	builder := &TimelockProposalBuilder{
		proposal: TimelockProposal{
			BaseProposal: BaseProposal{
				Kind:          types.KindTimelockProposal,
				ChainMetadata: make(map[types.ChainSelector]types.ChainMetadata),
			},
			TimelockIDs: make(map[types.ChainSelector]types.ContractID),
			Operations:  []types.BatchOperation{},
		},
	}
	builder.BaseProposalBuilder = BaseProposalBuilder[*TimelockProposalBuilder]{
		baseProposal: &builder.proposal.BaseProposal,
		builder:      builder,
	}

	return builder
}

// SetAction sets the action of the timelock proposal.
func (b *TimelockProposalBuilder) SetAction(action types.TimelockAction) *TimelockProposalBuilder {
	b.proposal.Action = action
	return b
}

// SetDelay sets the delay of the timelock proposal.
func (b *TimelockProposalBuilder) SetDelay(delay types.Duration) *TimelockProposalBuilder {
	b.proposal.Delay = delay
	return b
}

// SetTimelockAddress adds a timelock address to the timelock proposal.
func (b *TimelockProposalBuilder) SetTimelockAddresses(
	addrs map[types.ChainSelector]types.ContractID,
) *TimelockProposalBuilder {
	b.proposal.TimelockIDs = addrs
	return b
}

// AddTimelockAddress adds a timelock address for the given selector to the timelock proposal.
func (b *TimelockProposalBuilder) AddTimelockAddress(
	selector types.ChainSelector, timelockID types.ContractID,
) *TimelockProposalBuilder {
	b.proposal.TimelockIDs[selector] = timelockID
	return b
}

// AddOperation adds an operation to the timelock proposal.
func (b *TimelockProposalBuilder) AddOperation(bop types.BatchOperation) *TimelockProposalBuilder {
	b.proposal.Operations = append(b.proposal.Operations, bop)

	return b
}

// SetOperations sets all the operations of the proposal
func (b *TimelockProposalBuilder) SetOperations(bops []types.BatchOperation) *TimelockProposalBuilder {
	b.proposal.Operations = bops

	return b
}

// Build validates and returns the constructed TimelockProposal.
func (b *TimelockProposalBuilder) Build() (*TimelockProposal, error) {
	if err := b.proposal.Validate(); err != nil {
		return nil, err
	}

	return &b.proposal, nil
}
