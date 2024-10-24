package mcms

import (
	"sort"
	"time"

	"github.com/smartcontractkit/mcms/internal/core"
)

// MCMSProposal is a struct where the target contract is an MCMS contract
// with no forwarder contracts. This type does not support any type of atomic contract
// call batching, as the MCMS contract natively doesn't support batching
type MCMSProposal struct {
	Version              string      `json:"version"`
	ValidUntil           uint32      `json:"validUntil"`
	Signatures           []Signature `json:"signatures"`
	OverridePreviousRoot bool        `json:"overridePreviousRoot"`

	// Map of chain identifier to chain metadata
	ChainMetadata map[ChainSelector]ChainMetadata `json:"chainMetadata"`

	// This is intended to be displayed as-is to signers, to give them
	// context for the change. File authors should templatize strings for
	// this purpose in their pipelines.
	Description string `json:"description"`

	// Operations to be executed
	Transactions []ChainOperation `json:"transactions"`
}

func NewProposal(
	version string,
	validUntil uint32,
	signatures []Signature,
	overridePreviousRoot bool,
	chainMetadata map[ChainSelector]ChainMetadata,
	description string,
	transactions []ChainOperation,
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

func (m *MCMSProposal) ChainIdentifiers() []ChainSelector {
	chainIdentifiers := make([]ChainSelector, 0, len(m.ChainMetadata))
	for chainID := range m.ChainMetadata {
		chainIdentifiers = append(chainIdentifiers, chainID)
	}
	sort.Slice(chainIdentifiers, func(i, j int) bool { return chainIdentifiers[i] < chainIdentifiers[j] })

	return chainIdentifiers
}

func (m *MCMSProposal) TransactionCounts() map[ChainSelector]uint64 {
	txCounts := make(map[ChainSelector]uint64)
	for _, tx := range m.Transactions {
		txCounts[tx.ChainSelector]++
	}

	return txCounts
}

func (m *MCMSProposal) AddSignature(signature Signature) {
	m.Signatures = append(m.Signatures, signature)
}
