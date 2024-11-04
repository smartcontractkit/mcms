package mcms

import (
	"encoding/json"
	"io"
	"maps"
	"slices"
	"time"

	"github.com/smartcontractkit/mcms/internal/core"
	"github.com/smartcontractkit/mcms/internal/core/proposal"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

// MCMSProposal is a struct where the target contract is an MCMS contract
// with no forwarder contracts. This type does not support any type of atomic contract
// call batching, as the MCMS contract natively doesn't support batching
type MCMSProposal struct {
	Version              string            `json:"version"`
	ValidUntil           uint32            `json:"validUntil"`
	Signatures           []types.Signature `json:"signatures"`
	OverridePreviousRoot bool              `json:"overridePreviousRoot"`

	// Map of chain identifier to chain metadata
	ChainMetadata map[types.ChainSelector]types.ChainMetadata `json:"chainMetadata"`

	// This is intended to be displayed as-is to signers, to give them
	// context for the change. File authors should templatize strings for
	// this purpose in their pipelines.
	Description string `json:"description"`

	// Operations to be executed
	Transactions []types.ChainOperation `json:"transactions"`
}

var _ proposal.Proposal = (*MCMSProposal)(nil)

func NewProposal(
	version string,
	validUntil uint32,
	signatures []types.Signature,
	overridePreviousRoot bool,
	chainMetadata map[types.ChainSelector]types.ChainMetadata,
	description string,
	transactions []types.ChainOperation,
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

func NewProposalFromReader(reader io.Reader) (*MCMSProposal, error) {
	var out MCMSProposal
	err := json.NewDecoder(reader).Decode(&out)
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
		return core.ErrNoChainMetadata
	}

	if len(proposalObj.Transactions) == 0 {
		return core.ErrNoTransactions
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
			return NewChainMetadataNotFoundError(t.ChainSelector)
		}
	}

	return nil
}

// ChainSelectors returns a sorted list of chain selectors from the chains' metadata
func (m *MCMSProposal) ChainSelectors() []types.ChainSelector {
	return slices.Sorted(maps.Keys(m.ChainMetadata))
}

func (m *MCMSProposal) TransactionCounts() map[types.ChainSelector]uint64 {
	txCounts := make(map[types.ChainSelector]uint64)
	for _, tx := range m.Transactions {
		txCounts[tx.ChainSelector]++
	}

	return txCounts
}

func (m *MCMSProposal) AddSignature(signature types.Signature) {
	m.Signatures = append(m.Signatures, signature)
}

func (m *MCMSProposal) GetEncoders(isSim bool) (map[types.ChainSelector]sdk.Encoder, error) {
	txCounts := m.TransactionCounts()
	encoders := make(map[types.ChainSelector]sdk.Encoder)
	for chainID := range m.ChainMetadata {
		encoder, err := sdk.NewEncoder(chainID, txCounts[chainID], m.OverridePreviousRoot, isSim)
		if err != nil {
			return nil, err
		}

		encoders[chainID] = encoder
	}

	return encoders, nil
}

// TODO: isSim is very EVM and test Specific. Should be removed
func (m *MCMSProposal) Signable(isSim bool, inspectors map[types.ChainSelector]sdk.Inspector) (proposal.Signable, error) {
	encoders, err := m.GetEncoders(isSim)
	if err != nil {
		return nil, err
	}

	return NewSignable(m, encoders, inspectors)
}

func (m *MCMSProposal) Executable(isSim bool, executors map[types.ChainSelector]sdk.Executor) (*Executable, error) {
	encoders, err := m.GetEncoders(isSim)
	if err != nil {
		return nil, err
	}

	inspectors := make(map[types.ChainSelector]sdk.Inspector)
	for key, executor := range executors {
		inspectors[key] = executor // since Executor implements Inspector, this works
	}

	signable, err := NewSignable(m, encoders, inspectors) // TODO: we should be able to pass executors here?
	if err != nil {
		return nil, err
	}

	return NewExecutable(signable, executors), nil
}
