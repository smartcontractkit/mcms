package mcms

import (
	"encoding/json"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-playground/validator/v10"

	"github.com/smartcontractkit/mcms/internal/core"
	"github.com/smartcontractkit/mcms/internal/core/proposal"
	"github.com/smartcontractkit/mcms/internal/utils/safecast"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var ZERO_HASH = common.Hash{}

type MCMSWithTimelockProposal struct {
	BaseProposal

	Operation         types.TimelockAction           `json:"operation" validate:"required,oneof=schedule cancel bypass"`
	Delay             string                         `json:"delay"` // Will validate conditionally in Validate method
	TimelockAddresses map[types.ChainSelector]string `json:"timelockAddresses" validate:"required,min=1"`
	Transactions      []types.BatchChainOperation    `json:"transactions" validate:"required,min=1"`
}

// TODO: Could the input params be simplified here?
func NewProposalWithTimeLock(
	version string,
	validUntil uint32,
	signatures []types.Signature,
	overridePreviousRoot bool,
	chainMetadata map[types.ChainSelector]types.ChainMetadata,
	description string,
	timelockAddresses map[types.ChainSelector]string,
	batches []types.BatchChainOperation,
	timelockAction types.TimelockAction,
	timelockDelay string,
) (*MCMSWithTimelockProposal, error) {
	p := MCMSWithTimelockProposal{
		BaseProposal: BaseProposal{
			Version:              version,
			ValidUntil:           validUntil,
			Signatures:           signatures,
			OverridePreviousRoot: overridePreviousRoot,
			Description:          description,
			ChainMetadata:        chainMetadata,
		},
		Operation:         timelockAction,
		Delay:             timelockDelay,
		TimelockAddresses: timelockAddresses,
		Transactions:      batches,
	}

	errValidate := p.Validate()
	if errValidate != nil {
		return nil, errValidate
	}

	return &p, nil
}

// MarshalJSON convert the proposal to JSON
func (m *MCMSWithTimelockProposal) MarshalJSON() ([]byte, error) {
	// First, check the proposal is valid
	if err := m.Validate(); err != nil {
		return nil, err
	}

	// Let the default JSON marshaller handle everything
	type Alias MCMSWithTimelockProposal

	return json.Marshal((*Alias)(m))
}

// UnmarshalJSON convert the JSON to a proposal
func (m *MCMSWithTimelockProposal) UnmarshalJSON(data []byte) error {
	// Unmarshal all fields using the default unmarshaller
	type Alias MCMSWithTimelockProposal
	if err := json.Unmarshal(data, (*Alias)(m)); err != nil {
		return err
	}

	// Validate the proposal after unmarshalling
	if err := m.Validate(); err != nil {
		return err
	}

	return nil
}

func (m *MCMSWithTimelockProposal) Validate() error {
	// Run tag-based validation
	var validate = validator.New()
	if err := validate.Struct(m); err != nil {
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
		for _, op := range t.Batch {
			// Chain specific validations.
			if err := ValidateAdditionalFields(op.AdditionalFields, t.ChainSelector); err != nil {
				return err
			}
		}
	}

	if err := timeLockProposalValidateBasic(*m); err != nil {
		return err
	}

	return nil
}

func (m *MCMSWithTimelockProposal) Signable(inspectors map[types.ChainSelector]sdk.Inspector) (proposal.Signable, error) {
	// Convert the proposal to an MCMS only proposal
	mcmOnly, errToMcms := m.Convert()
	if errToMcms != nil {
		return nil, errToMcms
	}

	return mcmOnly.Signable(inspectors)
}

// Convert the proposal to an MCMS only proposal.
// Every transaction to be sent from the Timelock is encoded with the corresponding timelock method.
func (m *MCMSWithTimelockProposal) Convert() (MCMSProposal, error) {
	baseProposal := m.BaseProposal

	// Start predecessor map with all chains pointing to the zero hash
	predecessorMap := make(map[types.ChainSelector]common.Hash)
	for chain := range m.ChainMetadata {
		predecessorMap[chain] = ZERO_HASH
	}

	// Convert chain metadata
	baseProposal.ChainMetadata = make(map[types.ChainSelector]types.ChainMetadata)
	for chain, metadata := range m.ChainMetadata {
		baseProposal.ChainMetadata[chain] = types.ChainMetadata{
			StartingOpCount: metadata.StartingOpCount,
			MCMAddress:      metadata.MCMAddress,
		}
	}

	// Convert transactions into timelock wrapped transactions using the helper function
	result := MCMSProposal{
		BaseProposal: baseProposal,
	}
	for _, t := range m.Transactions {
		timelockAddress := m.TimelockAddresses[t.ChainSelector]
		predecessor := predecessorMap[t.ChainSelector]

		chainOp, operationId, err := BatchToChainOperation(
			t, timelockAddress, m.Delay, m.Operation, predecessor,
		)
		if err != nil {
			return MCMSProposal{}, err
		}

		// Append the converted operation to the MCMS only proposal
		result.Transactions = append(result.Transactions, chainOp)

		// Update predecessor for the chain
		predecessorMap[t.ChainSelector] = operationId
	}

	return result, nil
}

func (m *MCMSWithTimelockProposal) AddSignature(signature types.Signature) {
	m.Signatures = append(m.Signatures, signature)
}

// timeLockProposalValidateBasic basic validation for an MCMS proposal
func timeLockProposalValidateBasic(timelockProposal MCMSWithTimelockProposal) error {
	// Get the current Unix timestamp as an int64
	currentTime := time.Now().Unix()

	currentTimeCasted, err := safecast.Int64ToUint32(currentTime)
	if err != nil {
		return err
	}
	if timelockProposal.ValidUntil <= currentTimeCasted {
		// ValidUntil is a Unix timestamp, so it should be greater than the current time
		return &core.InvalidValidUntilError{
			ReceivedValidUntil: timelockProposal.ValidUntil,
		}
	}

	// Validate the delay is a valid duration but is only required
	// for Schedule operations
	if timelockProposal.Operation == types.TimelockActionSchedule {
		if _, err := time.ParseDuration(timelockProposal.Delay); err != nil {
			return &core.InvalidDelayError{
				ReceivedDelay: timelockProposal.Delay,
			}
		}
	}

	if len(timelockProposal.Transactions) > 0 && len(timelockProposal.Transactions[0].Batch) == 0 {
		return core.ErrNoTransactionsInBatch
	}

	return nil
}
