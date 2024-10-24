package internal

import (
	"sort"
	"time"

	"github.com/smartcontractkit/mcms/internal/core"
	"github.com/smartcontractkit/mcms/internal/core/proposal"
	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
)

// MCMSProposal is a struct where the target contract is an MCMS contract
// with no forwarder contracts. This type does not support any type of atomic contract
// call batching, as the MCMS contract natively doesn't support batching
type MCMSProposal struct {
	Version              string           `json:"version"`
	ValidUntil           uint32           `json:"validUntil"`
	Signatures           []mcms.Signature `json:"signatures"`
	OverridePreviousRoot bool             `json:"overridePreviousRoot"`

	// Map of chain identifier to chain metadata
	ChainMetadata map[mcms.ChainSelector]mcms.ChainMetadata `json:"chainMetadata"`

	// This is intended to be displayed as-is to signers, to give them
	// context for the change. File authors should templatize strings for
	// this purpose in their pipelines.
	Description string `json:"description"`

	// Operations to be executed
	Transactions []mcms.ChainOperation `json:"transactions"`
}

func NewProposal(
	version string,
	validUntil uint32,
	signatures []mcms.Signature,
	overridePreviousRoot bool,
	chainMetadata map[mcms.ChainSelector]mcms.ChainMetadata,
	description string,
	transactions []mcms.ChainOperation,
) (*MCMSProposal, error) {
	proposal := MCMSProposal{
		Version:              version,
		ValidUntil:           validUntil,
		Signatures:           signatures,
		OverridePreviousRoot: overridePreviousRoot,
		ChainMetadata:        chainMetadata,
		Description:          description,
		Transactions:         transactions,
	}

	err := proposal.Validate()
	if err != nil {
		return nil, err
	}

	return &proposal, nil
}

func NewProposalFromFile(filePath string) (*MCMSProposal, error) {
	var out MCMSProposal
	err := core.FromFile(filePath, &out)
	if err != nil {
		return nil, err
	}

	return &out, nil
}

// proposalValidateBasic basic validation for an MCMS proposal
func proposalValidateBasic(proposal MCMSProposal) error {
	// Get the current Unix timestamp as an int64
	currentTime := time.Now().Unix()

	currentTimeCasted, err := core.SafeCastIntToUint32(int(currentTime))
	if err != nil {
		return err
	}
	if proposal.ValidUntil <= currentTimeCasted {
		// ValidUntil is a Unix timestamp, so it should be greater than the current time
		return &core.InvalidValidUntilError{
			ReceivedValidUntil: proposal.ValidUntil,
		}
	}
	if len(proposal.ChainMetadata) == 0 {
		return &core.NoChainMetadataError{}
	}

	if len(proposal.Transactions) == 0 {
		return &core.NoTransactionsError{}
	}

	if proposal.Description == "" {
		return &core.InvalidDescriptionError{
			ReceivedDescription: proposal.Description,
		}
	}

	return nil
}

func (m *MCMSProposal) Validate() error {
	if m.Version == "" {
		return &core.InvalidVersionError{
			ReceivedVersion: m.Version,
		}
	}

	if err := proposalValidateBasic(*m); err != nil {
		return err
	}

	// Validate all chains in transactions have an entry in chain metadata
	for _, t := range m.Transactions {
		if _, ok := m.ChainMetadata[t.ChainSelector]; !ok {
			return &core.MissingChainDetailsError{
				ChainIdentifier: uint64(t.ChainSelector),
				Parameter:       "chain metadata",
			}
		}
	}

	return nil
}

func (m *MCMSProposal) ChainIdentifiers() []mcms.ChainSelector {
	chainIdentifiers := make([]mcms.ChainSelector, 0, len(m.ChainMetadata))
	for chainID := range m.ChainMetadata {
		chainIdentifiers = append(chainIdentifiers, chainID)
	}
	sort.Slice(chainIdentifiers, func(i, j int) bool { return chainIdentifiers[i] < chainIdentifiers[j] })

	return chainIdentifiers
}

func (m *MCMSProposal) TransactionCounts() map[mcms.ChainSelector]uint64 {
	txCounts := make(map[mcms.ChainSelector]uint64)
	for _, tx := range m.Transactions {
		txCounts[tx.ChainSelector]++
	}

	return txCounts
}

func (m *MCMSProposal) AddSignature(signature mcms.Signature) {
	m.Signatures = append(m.Signatures, signature)
}

func (m *MCMSProposal) GetEncoders(isSim bool) (map[mcms.ChainSelector]mcms.Encoder, error) {
	txCounts := m.TransactionCounts()
	encoders := make(map[mcms.ChainSelector]mcms.Encoder)
	for chainID := range m.ChainMetadata {
		encoder, err := NewEncoder(chainID, txCounts[chainID], m.OverridePreviousRoot, isSim)
		if err != nil {
			return nil, err
		}

		encoders[chainID] = encoder
	}

	return encoders, nil
}

func (m *MCMSProposal) Signable(isSim bool) (proposal.Signable, error) {
	encoders, err := m.GetEncoders(isSim)
	if err != nil {
		return nil, err
	}

	return NewSignable(m, encoders)
}
