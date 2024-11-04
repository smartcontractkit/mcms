package mcms

import (
	"encoding/json"
	"io"
	"maps"
	"slices"
	"time"

	"github.com/go-playground/validator/v10"

	"github.com/smartcontractkit/mcms/internal/core"
	"github.com/smartcontractkit/mcms/internal/core/proposal"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

// BaseProposal is the base struct for all MCMS proposals, contains shared fields for all proposal types.
type BaseProposal struct {
	Version              string                                      `json:"version" validate:"required"`
	ValidUntil           uint32                                      `json:"validUntil" validate:"required"`
	Signatures           []types.Signature                           `json:"signatures" validate:"omitempty,dive,required"`
	OverridePreviousRoot bool                                        `json:"overridePreviousRoot"`
	ChainMetadata        map[types.ChainSelector]types.ChainMetadata `json:"chainMetadata" validate:"required,min=1,dive,keys,required,endkeys"`
	Description          string                                      `json:"description" validate:"required"`
}

// MCMSProposal is a struct where the target contract is an MCMS contract
// with no forwarder contracts. This type does not support any type of atomic contract
// call batching, as the MCMS contract natively doesn't support batching
type MCMSProposal struct {
	BaseProposal
	Transactions []types.ChainOperation `json:"transactions" validate:"required,min=1,dive,required"`
}

var _ proposal.Proposal = (*MCMSProposal)(nil)

// MarshalJSON marshals the proposal to JSON
func (m *MCMSProposal) MarshalJSON() ([]byte, error) {
	// First, check the proposal is valid
	if err := m.Validate(); err != nil {
		return nil, err
	}

	// Let the default JSON marshaller handle everything
	type Alias MCMSProposal

	return json.Marshal((*Alias)(m))
}

// UnmarshalJSON unmarshals the JSON to a proposal
func (m *MCMSProposal) UnmarshalJSON(data []byte) error {
	// Unmarshal all fields using the default unmarshaller
	type Alias MCMSProposal
	if err := json.Unmarshal(data, (*Alias)(m)); err != nil {
		return err
	}

	// Validate the proposal after unmarshalling
	if err := m.Validate(); err != nil {
		return err
	}

	return nil
}
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
		BaseProposal: BaseProposal{
			Version:              version,
			ValidUntil:           validUntil,
			Signatures:           signatures,
			OverridePreviousRoot: overridePreviousRoot,
			ChainMetadata:        chainMetadata,
			Description:          description,
		},
		Transactions: transactions,
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

	return nil
}

func (m *MCMSProposal) Validate() error {
	// Run tag-based validation
	var validate = validator.New()
	if err := validate.Struct(m); err != nil {
		return err
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
