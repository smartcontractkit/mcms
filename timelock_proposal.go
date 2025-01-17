package mcms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-playground/validator/v10"

	"github.com/smartcontractkit/mcms/internal/utils/safecast"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var ZERO_HASH = common.Hash{}

type TimelockProposal struct {
	BaseProposal

	Action            types.TimelockAction           `json:"action" validate:"required,oneof=schedule cancel bypass"`
	Delay             types.Duration                 `json:"delay" validate:"required_if=Action schedule"`
	TimelockAddresses map[types.ChainSelector]string `json:"timelockAddresses" validate:"required,min=1"`
	Operations        []types.BatchOperation         `json:"operations" validate:"required,min=1,dive"`
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

func (m *TimelockProposal) Validate() error {
	// Run tag-based validation
	validate := validator.New()
	if err := validate.Struct(m); err != nil {
		return err
	}

	if m.Kind != types.KindTimelockProposal {
		return NewInvalidProposalKindError(m.Kind, types.KindTimelockProposal)
	}

	// Validate all chains in transactions have an entry in chain metadata
	for _, op := range m.Operations {
		if _, ok := m.ChainMetadata[op.ChainSelector]; !ok {
			return NewChainMetadataNotFoundError(op.ChainSelector)
		}

		for _, tx := range op.Transactions {
			// Chain specific validations.
			if err := ValidateAdditionalFields(tx.AdditionalFields, op.ChainSelector); err != nil {
				return err
			}
		}
	}

	if err := timeLockProposalValidateBasic(*m); err != nil {
		return err
	}

	return nil
}

// Convert the proposal to an MCMS only proposal and also return all predecessors for easy access later.
// Every transaction to be sent from the Timelock is encoded with the corresponding timelock method.
func (m *TimelockProposal) Convert(
	ctx context.Context,
	converters map[types.ChainSelector]sdk.TimelockConverter,
) (Proposal, []common.Hash, error) {
	baseProposal := m.BaseProposal

	// Start predecessor map with all chains pointing to the zero hash
	predecessors := make([]common.Hash, len(m.Operations)+1)
	predecessors[0] = ZERO_HASH

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
	for i, bop := range m.Operations {
		timelockAddress := m.TimelockAddresses[bop.ChainSelector]
		predecessor := predecessors[i]

		converter, ok := converters[bop.ChainSelector]
		if !ok {
			return Proposal{}, []common.Hash{}, fmt.Errorf("unable to find converter for chain selector: %d", bop.ChainSelector)
		}

		convertedOps, operationID, err := converter.ConvertBatchToChainOperation(
			ctx, bop, timelockAddress, m.Delay, m.Action, predecessor, baseProposal.Salt(),
		)
		if err != nil {
			return Proposal{}, nil, err
		}

		// Append the converted operation to the MCMS only proposal
		result.Operations = append(result.Operations, convertedOps...)

		// Update predecessor for the chain
		predecessors[i+1] = operationID
	}

	return result, predecessors, nil
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

	return nil
}
