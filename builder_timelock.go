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
				ChainMetadata: make(map[types.ChainSelector]types.ChainMetadata),
			},
			TimelockAddresses: make(map[types.ChainSelector]string),
			Transactions:      []types.BatchChainOperation{},
		},
	}
	builder.BaseProposalBuilder = BaseProposalBuilder[*TimelockProposalBuilder]{
		baseProposal: &builder.proposal.BaseProposal,
		builder:      builder,
	}

	return builder
}

// SetOperation sets the operation of the timelock proposal.
func (b *TimelockProposalBuilder) SetOperation(operation types.TimelockAction) *TimelockProposalBuilder {
	b.proposal.Operation = operation
	return b
}

// SetDelay sets the delay of the timelock proposal.
func (b *TimelockProposalBuilder) SetDelay(delay string) *TimelockProposalBuilder {
	b.proposal.Delay = delay
	return b
}

// SetTimelockAddress adds a timelock address to the timelock proposal.
func (b *TimelockProposalBuilder) SetTimelockAddress(selector types.ChainSelector, address string) *TimelockProposalBuilder {
	b.proposal.TimelockAddresses[selector] = address
	return b
}

// AddTransaction adds a transaction to the timelock proposal.
func (b *TimelockProposalBuilder) AddTransaction(transaction types.BatchChainOperation) *TimelockProposalBuilder {
	b.proposal.Transactions = append(b.proposal.Transactions, transaction)

	return b
}

// SetTransactions sets all the transactions of the proposal
func (b *TimelockProposalBuilder) SetTransactions(transactions []types.BatchChainOperation) *TimelockProposalBuilder {
	b.proposal.Transactions = transactions

	return b
}

// Build validates and returns the constructed TimelockProposal.
func (b *TimelockProposalBuilder) Build() (*TimelockProposal, error) {
	// Validate the proposal
	if err := b.proposal.Validate(); err != nil {
		return nil, err
	}

	return &b.proposal, nil
}
