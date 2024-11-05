package mcms

import (
	"encoding/json"
	"io"
	"maps"
	"slices"
	"time"

	"github.com/go-playground/validator/v10"

	"github.com/smartcontractkit/mcms/internal/core"
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

	// This field is passed to SDK implementations to indicate whether the proposal is being run
	// against a simulated environment. This is only used for testing purposes.
	useSimulatedBackend bool `json:"-"`
}

// MCMSProposal is a struct where the target contract is an MCMS contract
// with no forwarder contracts. This type does not support any type of atomic contract
// call batching, as the MCMS contract natively doesn't support batching
type MCMSProposal struct {
	BaseProposal
	Transactions []types.ChainOperation `json:"transactions" validate:"required,min=1,dive,required"`
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

// UseSimulatedBackend indicates whether the proposal should be run against a simulated backend.
//
// Simulated backends are used to test the proposal without actually sending transactions to the
// chain. The functionality toggled by this flag is implemented in the SDKs.
//
// Note that not all chain families may support this feature, so ensure your tests are only running
// against chains that support it.
func (m *MCMSProposal) UseSimulatedBackend(b bool) {
	m.useSimulatedBackend = b
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

func (m *MCMSProposal) GetEncoders() (map[types.ChainSelector]sdk.Encoder, error) {
	txCounts := m.TransactionCounts()
	encoders := make(map[types.ChainSelector]sdk.Encoder)
	for chainID := range m.ChainMetadata {
		encoder, err := sdk.NewEncoder(chainID, txCounts[chainID], m.OverridePreviousRoot, m.useSimulatedBackend)
		if err != nil {
			return nil, err
		}

		encoders[chainID] = encoder
	}

	return encoders, nil
}

// TODO: isSim is very EVM and test Specific. Should be removed
func (m *MCMSProposal) Signable(inspectors map[types.ChainSelector]sdk.Inspector) (*Signable, error) {
	encoders, err := m.GetEncoders()
	if err != nil {
		return nil, err
	}

	return NewSignable(m, encoders, inspectors)
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
