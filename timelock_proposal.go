package mcms

import (
	"context"
	"encoding/binary"
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
	Delay             types.Duration                 `json:"delay"`
	TimelockAddresses map[types.ChainSelector]string `json:"timelockAddresses" validate:"required,min=1"`
	Operations        []types.BatchOperation         `json:"operations" validate:"required,min=1,dive"`
	SaltOverride      *common.Hash                   `json:"salt,omitempty"`
}

var _ ProposalInterface = (*TimelockProposal)(nil)

// NewTimelockProposal unmarshal data from the reader to JSON and returns a new TimelockProposal.
func NewTimelockProposal(r io.Reader, opts ...ProposalOption) (*TimelockProposal, error) {
	options := &proposalOptions{}
	for _, opt := range opts {
		opt(options)
	}

	return newProposal[*TimelockProposal](r, options.predecessors)
}

func WriteTimelockProposal(w io.Writer, p *TimelockProposal) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	return enc.Encode(p)
}

// TransactionCounts returns the number of transactions for each chain in the proposal
func (m *TimelockProposal) TransactionCounts() map[types.ChainSelector]uint64 {
	counts := make(map[types.ChainSelector]uint64)
	for _, op := range m.Operations {
		counts[op.ChainSelector] += uint64(len(op.Transactions))
	}

	return counts
}

// Salt returns a unique salt for the proposal.
// We need the salt to be unique in case you use an identical operation again
// on the same chain across two different proposals. Predecessor protects against
// duplicates within the same proposal
func (m *TimelockProposal) Salt() [32]byte {
	if m.SaltOverride != nil {
		return *m.SaltOverride
	}

	// If the proposal doesn't have a salt, we create one from the
	// valid until timestamp.
	var salt [32]byte
	binary.BigEndian.PutUint32(salt[:], m.ValidUntil)

	return salt
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
			if err := validateAdditionalFields(tx.AdditionalFields, op.ChainSelector); err != nil {
				return err
			}
		}
	}

	if err := timeLockProposalValidateBasic(*m); err != nil {
		return err
	}

	return nil
}

// DeriveCancellationProposal derives a new proposal that cancels the current proposal.
func (m *TimelockProposal) DeriveCancellationProposal() (TimelockProposal, error) {
	if m.Action != types.TimelockActionSchedule {
		return TimelockProposal{}, fmt.Errorf("cannot derive a cancellation proposal from a non-schedule proposal. Action needs to be of type 'schedule'")
	}
	// Create a copy of the current proposal, we don't want to affect the original proposal
	newProposal := *m
	newProposal.Signatures = []types.Signature{}
	newProposal.Action = types.TimelockActionCancel
	return newProposal, nil
}

// DeriveBypassProposal derives a new proposal that bypasses the current proposal.
func (m *TimelockProposal) DeriveBypassProposal() (TimelockProposal, error) {
	if m.Action != types.TimelockActionSchedule {
		return TimelockProposal{}, fmt.Errorf("cannot derive a bypass proposal from a non-schedule proposal. Action needs to be of type 'schedule'")
	}
	// Create a copy of the current proposal, we don't want to affect the original proposal
	newProposal := *m
	newProposal.Signatures = []types.Signature{}
	newProposal.Action = types.TimelockActionBypass
	return newProposal, nil
}

// Convert the proposal to an MCMS only proposal and also return all predecessors for easy access later.
// Every transaction to be sent from the Timelock is encoded with the corresponding timelock method.
func (m *TimelockProposal) Convert(
	ctx context.Context,
	converters map[types.ChainSelector]sdk.TimelockConverter,
) (Proposal, []common.Hash, error) {
	// 1) Clone the base proposal, update the kind, etc.
	baseProposal := m.BaseProposal
	baseProposal.Kind = types.KindProposal

	// 2) Initialize the global predecessors slice
	predecessors := make([]common.Hash, len(m.Operations))

	// 3) Keep track of the last operation ID per chain
	lastOpID := make(map[types.ChainSelector]common.Hash)
	// Initialize them to ZERO_HASH
	for sel := range m.ChainMetadata {
		lastOpID[sel] = ZERO_HASH
	}

	// 4) Rebuild chainMetadata in baseProposal
	chainMetadataMap := make(map[types.ChainSelector]types.ChainMetadata)
	for chain, metadata := range m.ChainMetadata {
		chainMetadataMap[chain] = metadata
	}
	baseProposal.ChainMetadata = chainMetadataMap

	// 5) We’ll build the final MCMS-only proposal
	result := Proposal{
		BaseProposal: baseProposal,
	}

	// 6) Loop through operations in *global* order
	for i, bop := range m.Operations {
		chainSelector := bop.ChainSelector

		// If the chain isn't in converters, bail out
		converter, ok := converters[chainSelector]
		if !ok {
			return Proposal{}, nil, fmt.Errorf("unable to find converter for chain selector %d", chainSelector)
		}

		chainMetadata, ok := m.ChainMetadata[chainSelector]
		if !ok {
			return Proposal{}, nil, fmt.Errorf("missing chain metadata for chainSelector %d", chainSelector)
		}

		// The predecessor for this op is the lastOpID for its chain
		predecessor := lastOpID[chainSelector]
		predecessors[i] = predecessor

		timelockAddr := m.TimelockAddresses[chainSelector]

		// Convert the batch operation
		convertedOps, operationID, err := converter.ConvertBatchToChainOperations(
			ctx,
			chainMetadata,
			bop,
			timelockAddr,
			chainMetadata.MCMAddress,
			m.Delay,
			m.Action,
			predecessor,
			m.Salt(),
		)
		if err != nil {
			return Proposal{}, nil, err
		}

		// Append the converted operation to the MCMS only proposal
		result.Operations = append(result.Operations, convertedOps...)

		// Update lastOpID for that chain
		lastOpID[chainSelector] = operationID
	}

	// 7) Return the MCMS-only proposal + the single slice of predecessors
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
