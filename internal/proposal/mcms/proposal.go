package mcms

import (
	"sort"
	"time"

	"github.com/smartcontractkit/mcms/internal/core"
	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"

	"github.com/smartcontractkit/mcms/internal/core/proposal"
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
	proposalObj := MCMSProposal{
		Version:              version,
		ValidUntil:           validUntil,
		Signatures:           signatures,
		OverridePreviousRoot: overridePreviousRoot,
		ChainMetadata:        chainMetadata,
		Description:          description,
		Transactions:         transactions,
	}

	err := proposalObj.Validate()
	if err != nil {
		return nil, err
	}

	return &proposalObj, nil
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
func proposalValidateBasic(proposalObj MCMSProposal) error {
	validUntil := time.Unix(int64(proposalObj.ValidUntil), 0)

	if time.Now().After(validUntil) {
		return &core.InvalidValidUntilError{
			ReceivedValidUntil: proposalObj.ValidUntil,
		}
	}

	if len(proposalObj.ChainMetadata) == 0 {
		return &core.NoChainMetadataError{}
	}

	if len(proposalObj.Transactions) == 0 {
		return &core.NoTransactionsError{}
	}

	if proposalObj.Description == "" {
		return &core.InvalidDescriptionError{
			ReceivedDescription: proposalObj.Description,
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

func (m *MCMSProposal) Signable(isSim bool, inspectors map[mcms.ChainSelector]mcms.Inspector) (proposal.Signable, error) {
	encoders, err := m.GetEncoders(isSim)
	if err != nil {
		return nil, err
	}

	return NewSignable(m, encoders, inspectors)
}

func (m *MCMSProposal) Executable(isSim bool, executors map[mcms.ChainSelector]mcms.Executor) (proposal.Executable, error) {
	encoders, err := m.GetEncoders(isSim)
	if err != nil {
		return nil, err
	}

	inspectors := make(map[mcms.ChainSelector]mcms.Inspector)
	for key, executor := range executors {
		inspectors[key] = executor // since Executor implements Inspector, this works
	}

	signable, err := NewSignable(m, encoders, inspectors) // TODO: we should be able to pass executors here?
	if err != nil {
		return nil, err
	}

	return NewExecutable(signable, executors), nil
}
