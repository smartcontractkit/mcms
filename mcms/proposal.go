package mcms

import (
	"encoding/json"
	"io"
	"sort"
	"time"

	"github.com/go-playground/validator/v10"

	"github.com/smartcontractkit/mcms/internal/core"
	"github.com/smartcontractkit/mcms/internal/core/proposal"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

type BaseProposal struct {
	Version              string                                      `json:"version" validate:"required"`
	ValidUntil           uint32                                      `json:"validUntil" validate:"required"`
	Signatures           []types.Signature                           `json:"signatures" validate:"required"`
	OverridePreviousRoot bool                                        `json:"overridePreviousRoot"`
	ChainMetadata        map[types.ChainSelector]types.ChainMetadata `json:"chainMetadata" validate:"required,dive,keys,required,endkeys"`
	Description          string                                      `json:"description" validate:"required"`
}

type MCMSProposal struct {
	BaseProposal
	Transactions []types.ChainOperation `json:"transactions" validate:"required,dive,required"`
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

// Validate checks that the proposal is well-formed and adheres to the MCMS specification.
func (m *MCMSProposal) Validate() error {
	var validate = validator.New()
	// Basic struct validation using the validate package
	if err := validate.Struct(m); err != nil {
		return err
	}

	// Additional custom validation for transaction chains in ChainMetadata
	for _, t := range m.Transactions {
		if _, ok := m.ChainMetadata[t.ChainSelector]; !ok {
			return &core.MissingChainDetailsError{
				ChainIdentifier: uint64(t.ChainSelector),
				Parameter:       "chain metadata",
			}
		}
	}

	// Check ValidUntil
	validUntil := time.Unix(int64(m.ValidUntil), 0)
	if time.Now().After(validUntil) {
		return &core.InvalidValidUntilError{
			ReceivedValidUntil: m.ValidUntil,
		}
	}

	return nil
}
func (m *MCMSProposal) ChainIdentifiers() []types.ChainSelector {
	chainIdentifiers := make([]types.ChainSelector, 0, len(m.ChainMetadata))
	for chainID := range m.ChainMetadata {
		chainIdentifiers = append(chainIdentifiers, chainID)
	}
	sort.Slice(chainIdentifiers, func(i, j int) bool { return chainIdentifiers[i] < chainIdentifiers[j] })

	return chainIdentifiers
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
