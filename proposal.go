package mcms

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/go-playground/validator/v10"

	"github.com/smartcontractkit/mcms/internal/core/merkle"
	"github.com/smartcontractkit/mcms/internal/utils/abi"
	"github.com/smartcontractkit/mcms/internal/utils/safecast"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

const SignMsgABI = `[{"type":"bytes32"},{"type":"uint32"}]`

type ProposalInterface interface {
	AppendSignature(signature types.Signature)
	TransactionCounts() map[types.ChainSelector]uint64
	ChainMetadatas() map[types.ChainSelector]types.ChainMetadata
	setChainMetadata(chainSelector types.ChainSelector, metadata types.ChainMetadata)
	Validate() error
}

func LoadProposal(proposalType types.ProposalKind, filePath string) (ProposalInterface, error) {
	switch proposalType {
	case types.KindProposal:
		// Open the file
		file, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}

		// Ensure the file is closed when done
		defer file.Close()

		return NewProposal(file)
	case types.KindTimelockProposal:
		// Open the file
		file, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}

		// Ensure the file is closed when done
		defer file.Close()

		return NewTimelockProposal(file)
	default:
		return nil, errors.New("unknown proposal type")
	}
}

// BaseProposal is the base struct for all MCMS proposals, contains shared fields for all proposal types.
type BaseProposal struct {
	Version              string                                      `json:"version" validate:"required,oneof=v1"`
	Kind                 types.ProposalKind                          `json:"kind" validate:"required,oneof=Proposal TimelockProposal"`
	ValidUntil           uint32                                      `json:"validUntil" validate:"required"`
	Signatures           []types.Signature                           `json:"signatures" validate:"omitempty,dive,required"`
	OverridePreviousRoot bool                                        `json:"overridePreviousRoot"`
	ChainMetadata        map[types.ChainSelector]types.ChainMetadata `json:"chainMetadata" validate:"required,min=1"`
	Description          string                                      `json:"description"`

	// This field is passed to SDK implementations to indicate whether the proposal is being run
	// against a simulated environment. This is only used for testing purposes.
	useSimulatedBackend bool `json:"-"`
}

// AppendSignature appends a signature to the proposal's signature list.
func (p *BaseProposal) AppendSignature(signature types.Signature) {
	p.Signatures = append(p.Signatures, signature)
}

// ChainMetadata returns the chain metadata for the proposal.
func (p *BaseProposal) ChainMetadatas() map[types.ChainSelector]types.ChainMetadata {
	cmCopy := make(map[types.ChainSelector]types.ChainMetadata, len(p.ChainMetadata))
	for k, v := range p.ChainMetadata {
		cmCopy[k] = v
	}

	return cmCopy
}

// SetChainMetadata sets the chain metadata for a given chain selector.
func (p *BaseProposal) setChainMetadata(chainSelector types.ChainSelector, metadata types.ChainMetadata) {
	p.ChainMetadata[chainSelector] = metadata
}

// Proposal is a struct where the target contract is an MCMS contract
// with no forwarder contracts. This type does not support any type of atomic contract
// call batching, as the MCMS contract natively doesn't support batching
type Proposal struct {
	BaseProposal

	Operations []types.Operation `json:"operations" validate:"required,min=1,dive"`
}

var _ ProposalInterface = (*Proposal)(nil)

type ProposalOption func(*proposalOptions)

type proposalOptions struct {
	predecessors []io.Reader
}

// WithPredecessors is an option that allows the user to specify a list of
// that contain the predecessors for the proposal for configuring operations counts, which makes the following
// assumptions:
//   - The order of the predecessors array is the order in which the proposals are
//     intended to be executed.
//   - The op counts for the first proposal are meant to be the starting op for the
//     full set of proposals.
//   - The op counts for all other proposals except the first are ignored
//   - all proposals are configured correctly and need no additional modifications
func WithPredecessors(predecessors []io.Reader) ProposalOption {
	return func(opts *proposalOptions) {
		opts.predecessors = predecessors
		if opts.predecessors == nil {
			opts.predecessors = []io.Reader{}
		}
	}
}

// NewProposal unmarshals data from the reader to JSON and returns a new Proposal.
func NewProposal(reader io.Reader, opts ...ProposalOption) (*Proposal, error) {
	options := &proposalOptions{}
	for _, opt := range opts {
		opt(options)
	}

	return newProposal[*Proposal](reader, options.predecessors)
}

// WriteProposal marshals the proposal to JSON and writes it to the provided writer.
func WriteProposal(w io.Writer, proposal *Proposal) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	return enc.Encode(proposal)
}

func (p *Proposal) Validate() error {
	// Run tag-based validation
	var validate = validator.New()
	if err := validate.Struct(p); err != nil {
		return err
	}

	// Validate all chains in operations have an entry in chain metadata
	for _, op := range p.Operations {
		if _, ok := p.ChainMetadata[op.ChainSelector]; !ok {
			return NewChainMetadataNotFoundError(op.ChainSelector)
		}
	}

	for _, op := range p.Operations {
		// Chain specific validations.
		if err := ValidateAdditionalFields(op.Transaction.AdditionalFields, op.ChainSelector); err != nil {
			return err
		}
	}

	if err := proposalValidateBasic(*p); err != nil {
		return err
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
func (p *Proposal) UseSimulatedBackend(b bool) {
	p.useSimulatedBackend = b
}

// ChainSelectors returns a sorted list of chain selectors from the chains' metadata
func (p *Proposal) ChainSelectors() []types.ChainSelector {
	return slices.Sorted(maps.Keys(p.ChainMetadata))
}

// MerkleTree generates a merkle tree from the proposal's chain metadata and transactions.
func (p *Proposal) MerkleTree() (*merkle.Tree, error) {
	encoders, err := p.GetEncoders()
	if err != nil {
		return nil, wrapTreeGenErr(err)
	}

	hashLeaves := make([]common.Hash, 0)
	for _, sel := range p.ChainSelectors() {
		// Since we create encoders from the list of chain selectors provided in the ChainMetadata,
		// we can be sure the encoder exists, and don't need to check for existence.
		encoder := encoders[sel]

		// Similarly, we can be sure the metadata exists, as we iterate over the chain selectors,
		// since the chain selectors are keys in the ChainMetadata map.
		metadata := p.ChainMetadata[sel]

		encodedRootMetadata, encerr := encoder.HashMetadata(metadata)
		if encerr != nil {
			return nil, wrapTreeGenErr(encerr)
		}

		hashLeaves = append(hashLeaves, encodedRootMetadata)
	}

	for i, op := range p.Operations {
		txNonces, txerr := p.TransactionNonces()
		if txerr != nil {
			return nil, wrapTreeGenErr(txerr)
		}

		txNonce, txerr := safecast.Uint64ToUint32(txNonces[i])
		if txerr != nil {
			return nil, wrapTreeGenErr(txerr)
		}

		// This will always exist since encoders are created from the chain selectors in the
		// metadata, and TransactionNonces has validated that the metadata exists for each chain
		// selector defined in the transactions.
		encoder := encoders[op.ChainSelector]

		encodedOp, txerr := encoder.HashOperation(
			txNonce,
			p.ChainMetadata[op.ChainSelector],
			op,
		)
		if txerr != nil {
			return nil, wrapTreeGenErr(txerr)
		}
		hashLeaves = append(hashLeaves, encodedOp)
	}

	// sort the hashes and sort the pairs
	slices.SortFunc(hashLeaves, func(a, b common.Hash) int {
		return strings.Compare(a.String(), b.String())
	})

	return merkle.NewTree(hashLeaves), nil
}

// SigningHash returns the hash of the proposal that should be signed, using the tree root and the valid until timestamp.
func (p *Proposal) SigningHash() (common.Hash, error) {
	msg, err := p.SigningMessage()
	if err != nil {
		return common.Hash{}, err
	}

	return toEthSignedMessageHash(msg.Bytes()), nil
}

// SigningMessage generates a signing message without the EIP191 prefix.
// This function is used for ledger contexts where the ledger itself will apply the EIP191 prefix.
// Corresponds to the input here https://github.com/smartcontractkit/ccip-owner-contracts/blob/main/src/ManyChainMultiSig.sol#L202
func (p *Proposal) SigningMessage() (common.Hash, error) {
	tree, err := p.MerkleTree()
	if err != nil {
		return common.Hash{}, err
	}
	msg, err := abi.ABIEncode(SignMsgABI, tree.Root, p.ValidUntil)
	if err != nil {
		return [32]byte{}, err
	}

	return crypto.Keccak256Hash(msg), nil
}

// TransactionCounts returns a map of chain selectors to the number of transactions for that chain.
//
// Since proposal operations only contains a single transaction, we can count the number of
// operations per chain selector to get the number of transactions.
func (p *Proposal) TransactionCounts() map[types.ChainSelector]uint64 {
	txCounts := make(map[types.ChainSelector]uint64)
	for _, o := range p.Operations {
		txCounts[o.ChainSelector]++
	}

	return txCounts
}

// TransactionNonces calculates and returns a slice of nonces for each transaction based on their
// respective chain selectors and associated metadata.
//
// It returns a slice of nonces, where each nonce corresponds to a transaction in the same order
// as the transactions slice. The nonce is calculated as the local index of the transaction with
// respect to it's chain  selector, plus the starting op count for that chain selector.
func (p *Proposal) TransactionNonces() ([]uint64, error) {
	// Map to keep track of local index counts for each ChainSelector
	chainIndexMap := make(map[types.ChainSelector]uint64, len(p.ChainMetadata))

	txNonces := make([]uint64, len(p.Operations))
	for i, op := range p.Operations {
		// Get the current local index for this ChainSelector
		localIndex := chainIndexMap[op.ChainSelector]

		// Lookup the StartingOpCount for this ChainSelector from cmMap
		md, ok := p.ChainMetadata[op.ChainSelector]
		if !ok {
			return nil, NewChainMetadataNotFoundError(op.ChainSelector)
		}

		// Add the local index to the StartingOpCount to get the final nonce
		txNonces[i] = localIndex + md.StartingOpCount

		// Increment the local index for the current ChainSelector
		chainIndexMap[op.ChainSelector]++
	}

	return txNonces, nil
}

// GetEncoders generates encoders for each chain in the proposal's chain metadata.
func (p *Proposal) GetEncoders() (map[types.ChainSelector]sdk.Encoder, error) {
	txCounts := p.TransactionCounts()
	encoders := make(map[types.ChainSelector]sdk.Encoder)
	for chainSelector := range p.ChainMetadata {
		encoder, err := newEncoder(chainSelector, txCounts[chainSelector], p.OverridePreviousRoot, p.useSimulatedBackend)
		if err != nil {
			return nil, fmt.Errorf("unable to create encoder: %w", err)
		}

		encoders[chainSelector] = encoder
	}

	return encoders, nil
}

// proposalValidateBasic basic validation for an MCMS proposal
func proposalValidateBasic(proposalObj Proposal) error {
	validUntil := time.Unix(int64(proposalObj.ValidUntil), 0)

	if time.Now().After(validUntil) {
		return NewInvalidValidUntilError(proposalObj.ValidUntil)
	}

	return nil
}

// wrapTreeGenErr wraps an error with a message indicating that it occurred during
// merkle tree generation.
func wrapTreeGenErr(err error) error {
	return fmt.Errorf("merkle tree generation error: %w", err)
}
