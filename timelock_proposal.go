package mcms

import (
	"encoding/json"
	"io"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-playground/validator/v10"

	"github.com/smartcontractkit/mcms/internal/utils/safecast"
	"github.com/smartcontractkit/mcms/types"
)

var ZERO_HASH = common.Hash{}

type TimelockProposal struct {
	BaseProposal

	Operation         types.TimelockAction           `json:"operation" validate:"required,oneof=schedule cancel bypass"`
	Delay             string                         `json:"delay"` // Will validate conditionally in Validate method
	TimelockAddresses map[types.ChainSelector]string `json:"timelockAddresses" validate:"required,min=1"`
	Transactions      []types.BatchChainOperation    `json:"transactions" validate:"required,min=1"`
}

// NewTimelockProposal unmarshal data from the reader to JSON and returns a new TimelockProposal.
func NewTimelockProposal(r io.Reader) (*TimelockProposal, error) {
	var p TimelockProposal
	if err := json.NewDecoder(r).Decode(&p); err != nil {
		return nil, err
	}

	if err := p.Validate(); err != nil {
		return nil, err
	}

	return &p, nil
}

func WriteTimelockProposal(w io.Writer, p *TimelockProposal) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	return enc.Encode(p)
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
) (*TimelockProposal, error) {
	p := TimelockProposal{
		BaseProposal: BaseProposal{
			Version:              version,
			Kind:                 types.KindTimelockProposal,
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

func (m *TimelockProposal) Validate() error {
	// Run tag-based validation
	var validate = validator.New()
	if err := validate.Struct(m); err != nil {
		return err
	}

	if m.Kind != types.KindTimelockProposal {
		return NewInvalidProposalKindError(m.Kind, types.KindTimelockProposal)
	}

	// Validate all chains in transactions have an entry in chain metadata
	for _, t := range m.Transactions {
		if _, ok := m.ChainMetadata[t.ChainSelector]; !ok {
			return NewChainMetadataNotFoundError(t.ChainSelector)
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

// Convert the proposal to an MCMS only proposal.
// Every transaction to be sent from the Timelock is encoded with the corresponding timelock method.
func (m *TimelockProposal) Convert() (Proposal, error) {
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
	result := Proposal{
		BaseProposal: baseProposal,
	}
	for _, t := range m.Transactions {
		timelockAddress := m.TimelockAddresses[t.ChainSelector]
		predecessor := predecessorMap[t.ChainSelector]

		chainOp, operationId, err := BatchToChainOperation(
			t, timelockAddress, m.Delay, m.Operation, predecessor,
		)
		if err != nil {
			return Proposal{}, err
		}

		// Append the converted operation to the MCMS only proposal
		result.Transactions = append(result.Transactions, chainOp)

		// Update predecessor for the chain
		predecessorMap[t.ChainSelector] = operationId
	}

	return result, nil
}

// timeLockProposalValidateBasic basic validation for an MCMS proposal
func timeLockProposalValidateBasic(timelockProposal TimelockProposal) error {
	// Get the current Unix timestamp as an int64
	currentTime := time.Now().Unix()

	currentTimeCasted, err := safecast.Int64ToUint32(currentTime)
	if err != nil {
		return err
	}
	if timelockProposal.ValidUntil <= currentTimeCasted {
		// ValidUntil is a Unix timestamp, so it should be greater than the current time
		return NewInvalidValidUntilError(timelockProposal.ValidUntil)
	}

	// Validate the delay is a valid duration but is only required
	// for Schedule operations
	if timelockProposal.Operation == types.TimelockActionSchedule {
		if _, err := time.ParseDuration(timelockProposal.Delay); err != nil {
			return NewInvalidDelayError(timelockProposal.Delay)
		}
	}

	if len(timelockProposal.Transactions) > 0 && len(timelockProposal.Transactions[0].Batch) == 0 {
		return ErrNoTransactionsInBatch
	}

	return nil
}
