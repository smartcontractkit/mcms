package mcms

import (
	"github.com/smartcontractkit/mcms/types"
)

// BaseProposalBuilder is a generic builder for BaseProposal.
// T is the concrete builder type embedding this struct.
type BaseProposalBuilder[T any] struct {
	baseProposal *BaseProposal
	builder      T
}

// SetVersion sets the version field of the BaseProposal.
func (b *BaseProposalBuilder[T]) SetVersion(version string) T {
	b.baseProposal.Version = version
	return b.builder
}

// SetValidUntil sets the validUntil field of the BaseProposal.
func (b *BaseProposalBuilder[T]) SetValidUntil(validUntil uint32) T {
	b.baseProposal.ValidUntil = validUntil
	return b.builder
}

// AddSignature adds a signature to the BaseProposal.
func (b *BaseProposalBuilder[T]) AddSignature(signature types.Signature) T {
	b.baseProposal.Signatures = append(b.baseProposal.Signatures, signature)
	return b.builder
}

// SetOverridePreviousRoot sets the overridePreviousRoot field of the BaseProposal.
func (b *BaseProposalBuilder[T]) SetOverridePreviousRoot(override bool) T {
	b.baseProposal.OverridePreviousRoot = override
	return b.builder
}

// AddChainMetadata adds chain metadata item to the BaseProposal's chain metadata.
func (b *BaseProposalBuilder[T]) AddChainMetadata(selector types.ChainSelector, metadata types.ChainMetadata) T {
	b.baseProposal.ChainMetadata[selector] = metadata
	return b.builder
}

// SetChainMetadata sets the chain metadata of the BaseProposal.
func (b *BaseProposalBuilder[T]) SetChainMetadata(metadata map[types.ChainSelector]types.ChainMetadata) T {
	b.baseProposal.ChainMetadata = metadata
	return b.builder
}

// SetDescription sets the description of the BaseProposal.
func (b *BaseProposalBuilder[T]) SetDescription(description string) T {
	b.baseProposal.Description = description
	return b.builder
}

// UseSimulatedBackend sets the useSimulatedBackend field of the BaseProposal.
func (b *BaseProposalBuilder[T]) UseSimulatedBackend(useSim bool) T {
	b.baseProposal.useSimulatedBackend = useSim
	return b.builder
}

// ProposalBuilder is a builder for the MCMS Proposal.
type ProposalBuilder struct {
	BaseProposalBuilder[*ProposalBuilder]
	proposal Proposal
}

// NewProposalBuilder creates a new ProposalBuilder.
func NewProposalBuilder() *ProposalBuilder {
	builder := &ProposalBuilder{
		proposal: Proposal{
			BaseProposal: BaseProposal{
				Kind:          types.KindProposal,
				ChainMetadata: make(map[types.ChainSelector]types.ChainMetadata),
			},
			Transactions: []types.ChainOperation{},
		},
	}
	// Initialize the BaseProposalBuilder with a reference to the base proposal and the builder itself.
	builder.BaseProposalBuilder = BaseProposalBuilder[*ProposalBuilder]{
		baseProposal: &builder.proposal.BaseProposal,
		builder:      builder,
	}

	return builder
}

// AddTransaction adds a transaction to the Proposal.
func (b *ProposalBuilder) AddTransaction(transaction types.ChainOperation) *ProposalBuilder {
	b.proposal.Transactions = append(b.proposal.Transactions, transaction)

	return b
}

// SetTransactions sets all the transactions of the Proposal.
func (b *ProposalBuilder) SetTransactions(transaction []types.ChainOperation) *ProposalBuilder {
	b.proposal.Transactions = transaction

	return b
}

// Build validates and returns the constructed Proposal.
func (b *ProposalBuilder) Build() (*Proposal, error) {
	// Validate the proposal
	if err := b.proposal.Validate(); err != nil {
		return nil, err
	}

	return &b.proposal, nil
}
