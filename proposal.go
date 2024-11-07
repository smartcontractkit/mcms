package mcms

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/go-playground/validator/v10"

	"github.com/smartcontractkit/mcms/internal/core"
	"github.com/smartcontractkit/mcms/internal/core/merkle"
	"github.com/smartcontractkit/mcms/internal/utils/safecast"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

const SignMsgABI = `[{"type":"bytes32"},{"type":"uint32"}]`

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

// Proposal is a struct where the target contract is an MCMS contract
// with no forwarder contracts. This type does not support any type of atomic contract
// call batching, as the MCMS contract natively doesn't support batching
type Proposal struct {
	BaseProposal

	Transactions []types.ChainOperation `json:"transactions" validate:"required,min=1"`
}

// NewProposal unmarshal data from the reader to JSON and returns a new Proposal.
func NewProposal(reader io.Reader) (*Proposal, error) {
	var p Proposal
	if err := json.NewDecoder(reader).Decode(&p); err != nil {
		return nil, err
	}

	if err := p.Validate(); err != nil {
		return nil, err
	}

	return &p, nil
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

	if p.Kind != types.KindProposal {
		return NewInvalidProposalKindError(p.Kind, types.KindProposal)
	}

	if err := proposalValidateBasic(*p); err != nil {
		return err
	}

	// Validate all chains in transactions have an entry in chain metadata
	for _, t := range p.Transactions {
		if _, ok := p.ChainMetadata[t.ChainSelector]; !ok {
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

	for i, tx := range p.Transactions {
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
		encoder := encoders[tx.ChainSelector]

		encodedOp, txerr := encoder.HashOperation(
			txNonce,
			p.ChainMetadata[tx.ChainSelector],
			tx,
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

	return toEthSignedMessageHash(msg), nil
}

// SigningMessage generates a signing message without the EIP191 prefix.
// This function is used for ledger contexts where the ledger itself will apply the EIP191 prefix.
// Corresponds to the input here https://github.com/smartcontractkit/ccip-owner-contracts/blob/main/src/ManyChainMultiSig.sol#L202
func (p *Proposal) SigningMessage() ([32]byte, error) {
	tree, err := p.MerkleTree()
	if err != nil {
		return common.Hash{}, err
	}
	msg, err := evm.ABIEncode(SignMsgABI, tree.Root, p.ValidUntil)
	if err != nil {
		return [32]byte{}, err
	}

	return crypto.Keccak256Hash(msg), nil
}

// TransactionCounts returns a map of chain selectors to the number of transactions for that chain
func (p *Proposal) TransactionCounts() map[types.ChainSelector]uint64 {
	txCounts := make(map[types.ChainSelector]uint64)
	for _, tx := range p.Transactions {
		txCounts[tx.ChainSelector]++
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

	txNonces := make([]uint64, len(p.Transactions))
	for i, tx := range p.Transactions {
		// Get the current local index for this ChainSelector
		localIndex := chainIndexMap[tx.ChainSelector]

		// Lookup the StartingOpCount for this ChainSelector from cmMap
		md, ok := p.ChainMetadata[tx.ChainSelector]
		if !ok {
			return nil, NewChainMetadataNotFoundError(tx.ChainSelector)
		}

		// Add the local index to the StartingOpCount to get the final nonce
		txNonces[i] = localIndex + md.StartingOpCount

		// Increment the local index for the current ChainSelector
		chainIndexMap[tx.ChainSelector]++
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

func toEthSignedMessageHash(messageHash common.Hash) common.Hash {
	// Add the Ethereum signed message prefix
	prefix := []byte("\x19Ethereum Signed Message:\n32")
	data := append(prefix, messageHash.Bytes()...)

	// Hash the prefixed message
	return crypto.Keccak256Hash(data)
}

// proposalValidateBasic basic validation for an MCMS proposal
func proposalValidateBasic(proposalObj Proposal) error {
	validUntil := time.Unix(int64(proposalObj.ValidUntil), 0)

	if time.Now().After(validUntil) {
		return &core.InvalidValidUntilError{
			ReceivedValidUntil: proposalObj.ValidUntil,
		}
	}

	return nil
}

// wrapTreeGenErr wraps an error with a message indicating that it occurred during
// merkle tree generation.
func wrapTreeGenErr(err error) error {
	return fmt.Errorf("merkle tree generation error: %w", err)
}
