package mcms

import (
	"encoding/json"
	"time"

	"github.com/smartcontractkit/mcms/pkg/proposal/mcms/types"

	"io"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/pkg/errors"
)

type ChainMetadata struct {
	StartingOpCount uint64         `json:"startingOpCount"`
	MCMAddress      common.Address `json:"mcmAddress"`
}

// MCMSProposal is a struct where the target contract is an MCMS contract
// with no forwarder contracts. This type does not support any type of atomic contract
// call batching, as the MCMS contract natively doesn't support batching
type MCMSProposal struct {
	Version              string      `json:"version"`
	ValidUntil           uint32      `json:"validUntil"`
	Signatures           []Signature `json:"signatures"`
	OverridePreviousRoot bool        `json:"overridePreviousRoot"`

	// Map of chain identifier to chain metadata
	ChainMetadata map[types.ChainIdentifier]ChainMetadata `json:"chainMetadata"`

	// This is intended to be displayed as-is to signers, to give them
	// context for the change. File authors should templatize strings for
	// this purpose in their pipelines.
	Description string `json:"description"`

	// Operations to be executed
	Transactions []types.ChainOperation `json:"transactions"`
}

func NewProposal(
	version string,
	validUntil uint32,
	signatures []Signature,
	overridePreviousRoot bool,
	chainMetadata map[types.ChainIdentifier]ChainMetadata,
	description string,
	transactions []types.ChainOperation,
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

// MarshalJSON implements the JSON marshaller for MCMSProposal
func (m MCMSProposal) MarshalJSON() ([]byte, error) {
	// Validate the proposal before marshalling
	if err := m.Validate(); err != nil {
		return nil, err
	}

	// Use an alias type to avoid recursion
	// We could exclude fields here in the future if necessary
	type Alias MCMSProposal

	return json.Marshal((*Alias)(&m))
}

// UnmarshalJSON implements the JSON unmarshaller for MCMSProposal
func (m *MCMSProposal) UnmarshalJSON(data []byte) error {
	// Use an alias type to avoid recursion
	type Alias MCMSProposal

	// Unmarshal the JSON data into the alias struct to avoid recursion
	if err := json.Unmarshal(data, (*Alias)(m)); err != nil {
		return err
	}

	// Check if AdditionalFields contains "null"
	for i := range m.Transactions {
		if string(m.Transactions[i].Operation.AdditionalFields) == "null" {
			m.Transactions[i].Operation.AdditionalFields = nil
		}
	}
	// Run validation after unmarshalling
	return m.Validate()
}

// NewProposalFromReader reads the proposal from an io.Reader (e.g., file, network response)
func NewProposalFromReader(reader io.Reader) (*MCMSProposal, error) {
	var out MCMSProposal
	err := json.NewDecoder(reader).Decode(&out)
	if err != nil {
		return nil, err
	}

	return &out, nil
}

// proposalValidateBasic basic validation for an MCMS proposal
func proposalValidateBasic(proposal MCMSProposal) error {
	// Get the current Unix timestamp as an int64
	currentTime := time.Now().Unix()

	currentTimeCasted, err := SafeCastIntToUint32(int(currentTime))
	if err != nil {
		return err
	}
	if proposal.ValidUntil <= currentTimeCasted {
		// ValidUntil is a Unix timestamp, so it should be greater than the current time
		return &errors.InvalidValidUntilError{
			ReceivedValidUntil: proposal.ValidUntil,
		}
	}
	if len(proposal.ChainMetadata) == 0 {
		return &errors.NoChainMetadataError{}
	}

	// We skip validation on timelock proposals. For time lock proposals this transaction list
	// will be empty as it is validated in the timelock proposal struct.
	if len(proposal.Transactions) == 0 && proposal.Version != string(MCMSWithTimelock) {
		return &errors.NoTransactionsError{}
	}

	if proposal.Description == "" {
		return &errors.InvalidDescriptionError{
			ReceivedDescription: proposal.Description,
		}
	}

	return nil
}

// Validate validates the MCMS proposal, including chain specific fields from the additionalFields field of the proposal
func (m *MCMSProposal) Validate() error {
	if m.Version == "" {
		return &errors.InvalidVersionError{
			ReceivedVersion: m.Version,
		}
	}

	if err := proposalValidateBasic(*m); err != nil {
		return err
	}

	// Validate all chains in transactions have an entry in chain metadata
	for _, t := range m.Transactions {
		if _, ok := m.ChainMetadata[t.ChainIdentifier]; !ok {
			return &errors.MissingChainDetailsError{
				ChainIdentifier: uint64(t.ChainIdentifier),
				Parameter:       "chain metadata",
			}
		}
		// Chain specific validations.
		if err := ValidateAdditionalFields(t.Operation.AdditionalFields, t.ChainIdentifier); err != nil {
			return err
		}
	}

	return nil
}

func (m *MCMSProposal) ToExecutor(sim bool) (*Executor, error) {
	// Create a new executor
	executor, err := NewProposalExecutor(m, sim)
	if err != nil {
		return nil, err
	}

	return executor, nil
}

func (m *MCMSProposal) AddSignature(signature Signature) {
	m.Signatures = append(m.Signatures, signature)
}
