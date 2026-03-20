package mcms

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	"github.com/go-playground/validator/v10"

	"github.com/smartcontractkit/mcms/chainwrappers"
	"github.com/smartcontractkit/mcms/internal/utils/safecast"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var (
	ZeroHash          = common.Hash{}
	DefaultValidUntil = 72 * time.Hour
)

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

	proposal, err := newProposal[*TimelockProposal](r, options.predecessors)
	if err != nil {
		return nil, err
	}

	_, err = proposal.SetOperationIDs(context.Background(), true) // TODO: OPT-400
	if err != nil {
		return nil, fmt.Errorf("failed to set operation IDs: %w", err)
	}

	return proposal, nil
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

	return timeLockProposalValidateBasic(*m)
}

func replaceChainMetadataWithAddresses(p *TimelockProposal, addresses map[types.ChainSelector]types.ChainMetadata) error {
	for chain := range p.ChainMetadata {
		newMeta, ok := addresses[chain]
		if !ok {
			return fmt.Errorf("cannot replace addresses in chain metadata, missing address for chain %d", chain)
		}
		p.ChainMetadata[chain] = newMeta
	}

	return nil
}

// deriveNewProposal creates a copy of the current proposal with overridden action, signatures, salt, and metadata.
func (m *TimelockProposal) deriveNewProposal(action types.TimelockAction, metadata map[types.ChainSelector]types.ChainMetadata) (TimelockProposal, error) {
	// Create a copy of the current proposal, we don't want to affect the original proposal
	newProposal := *m
	newProposal.Signatures = []types.Signature{}
	ts := time.Now().Add(DefaultValidUntil).Unix()
	ts32, err := safecast.Int64ToUint32(ts)
	if err != nil {
		return TimelockProposal{}, err
	}
	// #nosec G115
	newProposal.ValidUntil = ts32
	bytesSalt := m.Salt()
	salt := common.BytesToHash(bytesSalt[:])
	newProposal.SaltOverride = &salt
	newProposal.Action = action
	err = replaceChainMetadataWithAddresses(&newProposal, metadata)
	if err != nil {
		return TimelockProposal{}, err
	}

	return newProposal, nil
}

// DeriveCancellationProposal derives a new proposal that cancels the current proposal.
func (m *TimelockProposal) DeriveCancellationProposal(cancellerMetadata map[types.ChainSelector]types.ChainMetadata) (TimelockProposal, error) {
	if m.Action != types.TimelockActionSchedule {
		return TimelockProposal{}, errors.New("cannot derive a cancellation proposal from a non-schedule proposal. Action needs to be of type 'schedule'")
	}

	return m.deriveNewProposal(types.TimelockActionCancel, cancellerMetadata)
}

// DeriveBypassProposal derives a new proposal that bypasses the current proposal.
func (m *TimelockProposal) DeriveBypassProposal(bypasserAddresses map[types.ChainSelector]types.ChainMetadata) (TimelockProposal, error) {
	if m.Action != types.TimelockActionSchedule {
		return TimelockProposal{}, errors.New("cannot derive a bypass proposal from a non-schedule proposal. Action needs to be of type 'schedule'")
	}

	return m.deriveNewProposal(types.TimelockActionBypass, bypasserAddresses)
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
	// Initialize them to ZeroHash
	for sel := range m.ChainMetadata {
		lastOpID[sel] = ZeroHash
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

func (m *TimelockProposal) OperationIDs(ctx context.Context) ([]common.Hash, []common.Hash, error) {
	predecessors, err := m.SetOperationIDs(ctx, false)
	if err != nil {
		return nil, nil, err
	}
	operationIDs := lo.Map(m.Operations, func(op types.BatchOperation, _ int) common.Hash { return op.OperationID })

	return operationIDs, predecessors, nil
}

func (m *TimelockProposal) OperationID(ctx context.Context, index int) (common.Hash, error) {
	if index < 0 || index >= len(m.Operations) {
		return common.Hash{}, fmt.Errorf("operation index %d out of range", index)
	}

	if lo.IsEmpty(m.Operations[index].OperationID) {
		_, err := m.SetOperationIDs(ctx, false)
		if err != nil {
			return common.Hash{}, fmt.Errorf("failed to set operation IDs: %w", err)
		}
	}

	return m.Operations[index].OperationID, nil
}

// Decode decodes the raw transactions into a list of human-readable operations.
func (m *TimelockProposal) Decode(decoders map[types.ChainSelector]sdk.Decoder, contractInterfaces map[string]string) ([][]sdk.DecodedOperation, error) {
	decodedOps := make([][]sdk.DecodedOperation, len(m.Operations))
	for i, op := range m.Operations {
		// Get the decoder for the chain selector
		decoder, ok := decoders[op.ChainSelector]
		if !ok {
			return nil, fmt.Errorf("no decoder found for chain selector %d", op.ChainSelector)
		}

		for _, tx := range op.Transactions {
			// Get the contract interfaces for the contract type
			contractInterface, ok := contractInterfaces[tx.ContractType]
			if !ok {
				return nil, fmt.Errorf("no contract interfaces found for contract type %s", tx.ContractType)
			}

			decodedOp, err := decoder.Decode(tx, contractInterface)
			if err != nil {
				return nil, fmt.Errorf("unable to decode operation: %w", err)
			}

			decodedOps[i] = append(decodedOps[i], decodedOp)
		}
	}

	return decodedOps, nil
}

// buildTimelockConverters builds a map of chain selectors to their corresponding TimelockConverter implementations.
func (m *TimelockProposal) buildTimelockConverters(_ context.Context) (map[types.ChainSelector]sdk.TimelockConverter, error) {
	return chainwrappers.BuildConverters(m.ChainMetadata)
}

// SetOperationIDs computes and saves the id of each batch operation. It returns the
// list of predecessor operation ids (for each batch operation). If "updateMetadata"
// is true, it introspects the "metadata" field and replaces any occurrences of an
// old operation id with its updated value.
func (m *TimelockProposal) SetOperationIDs(ctx context.Context, updateMetadata bool) ([]common.Hash, error) {
	predecessors := make([]common.Hash, len(m.Operations))
	lastOpID := make(map[types.ChainSelector]common.Hash)
	for sel := range m.ChainMetadata {
		lastOpID[sel] = ZeroHash
	}

	oldToNewOperationIDsMap := make(map[common.Hash]common.Hash)

	for i, batchOp := range m.Operations {
		predecessors[i] = lastOpID[batchOp.ChainSelector]
		prevOperationID := batchOp.OperationID

		calculateOperationID, err := operationIDFn(ctx, batchOp.ChainSelector)
		if err != nil {
			return nil, fmt.Errorf("no operation ID function found for chain selector %d: %w", batchOp.ChainSelector, err)
		}

		newOperationID, err := calculateOperationID(batchOp, m.Action, predecessors[i], m.Salt())
		if err != nil {
			return nil, fmt.Errorf("failed to calculate operation ID for chain selector %d: %w", batchOp.ChainSelector, err)
		}

		if lo.IsNotEmpty(prevOperationID) && prevOperationID != newOperationID {
			oldToNewOperationIDsMap[prevOperationID] = newOperationID
		}

		lastOpID[batchOp.ChainSelector] = newOperationID
		m.Operations[i].OperationID = newOperationID
	}

	if updateMetadata && m.Metadata != nil && len(oldToNewOperationIDsMap) > 0 {
		updatedMetadata, ok := updateOpIDsInMetadata(m.Metadata, oldToNewOperationIDsMap).(map[string]any)
		if !ok {
			return nil, fmt.Errorf("failed to update operation IDs in metadata: unexpected type after update: %T", updatedMetadata)
		}
		m.Metadata = updatedMetadata
	}

	return predecessors, nil
}

// OperationCounts returns per-chain counts *after* conversion for all chains in
// the proposal, as some chains have different operation counts after conversion.
func (m *TimelockProposal) OperationCounts(ctx context.Context) (map[types.ChainSelector]uint64, error) {
	// Start with raw counts (works for all non-converted chains)
	out := m.TransactionCounts()

	converters, err := m.buildTimelockConverters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to build timelock converters: %w", err)
	}

	// Convert the proposal with the provided converters
	prop, _, err := m.Convert(ctx, converters)
	if err != nil {
		return nil, fmt.Errorf("failed to convert proposal: %w", err)
	}

	// Count converted ops per chain
	convCounts := make(map[types.ChainSelector]uint64)
	for _, op := range prop.Operations {
		convCounts[op.ChainSelector]++
	}

	// Overlay converted counts only for chains we attempted to convert
	for sel := range converters {
		if n, ok := convCounts[sel]; ok {
			out[sel] = n
		}
	}

	return out, nil
}

// GetOpCount queries the on-chain MCMS contract for the current op count of the given chain.
func (m *TimelockProposal) GetOpCount(
	ctx context.Context,
	chains chainwrappers.ChainAccessor,
	chainSelector types.ChainSelector,
	opts ...GetOpCountOption,
) (uint64, error) {
	if m == nil {
		return 0, errors.New("nil proposal")
	}

	metadata, ok := m.ChainMetadata[chainSelector]
	if !ok {
		return 0, fmt.Errorf("missing chain metadata for selector %d", chainSelector)
	}

	options := getOpCountOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	inspector := options.inspector
	if inspector == nil {
		var err error
		inspector, err = chainwrappers.BuildInspector(chains, chainSelector, m.Action, metadata)
		if err != nil {
			return 0, err
		}
	}

	return inspector.GetOpCount(ctx, metadata.MCMAddress)
}

type getOpCountOptions struct {
	inspector sdk.Inspector
}

type GetOpCountOption func(*getOpCountOptions)

// WithInspector overrides the default inspector (useful for tests).
func WithInspector(inspector sdk.Inspector) GetOpCountOption {
	return func(o *getOpCountOptions) {
		o.inspector = inspector
	}
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
