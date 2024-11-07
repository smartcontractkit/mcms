package mcms

import (
	"encoding/binary"
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
	"github.com/smartcontractkit/mcms/types"
)

// BaseProposal is the base struct for all MCMS proposals, contains shared fields for all proposal types.
type BaseProposal struct {
	Version              string                                      `json:"version" validate:"required"`
	ValidUntil           uint32                                      `json:"validUntil" validate:"required"`
	Signatures           []types.Signature                           `json:"signatures" validate:"omitempty,dive,required"`
	OverridePreviousRoot bool                                        `json:"overridePreviousRoot"`
	ChainMetadata        map[types.ChainSelector]types.ChainMetadata `json:"chainMetadata" validate:"required,min=1"`
	Description          string                                      `json:"description"`

	// This field is passed to SDK implementations to indicate whether the proposal is being run
	// against a simulated environment. This is only used for testing purposes.
	useSimulatedBackend bool `json:"-"`
}

// Proposal is a struct where the target contract is an MCMS contract
// with no forwarder contracts. This type does not support any type of atomic contract
// call batching, as the MCMS contract natively doesn't support batching
type Proposal struct {
	BaseProposal

	Transactions []types.ChainOperation `json:"transactions" validate:"required,min=1"`
}

func NewProposal(reader io.Reader) (*Proposal, error) {
	var out Proposal
	err := json.NewDecoder(reader).Decode(&out)
	if err != nil {
		return nil, err
	}

	if err := out.Validate(); err != nil {
		return nil, err
	}

	return &out, nil
}

// MarshalJSON marshals the proposal to JSON
func (m *Proposal) MarshalJSON() ([]byte, error) {
	// First, check the proposal is valid
	if err := m.Validate(); err != nil {
		return nil, err
	}

	// Let the default JSON marshaller handle everything
	type Alias Proposal

	return json.Marshal((*Alias)(m))
}

// UnmarshalJSON unmarshals the JSON to a proposal
func (m *Proposal) UnmarshalJSON(data []byte) error {
	// Unmarshal all fields using the default unmarshaller
	type Alias Proposal
	if err := json.Unmarshal(data, (*Alias)(m)); err != nil {
		return err
	}

	// Validate the proposal after unmarshalling
	if err := m.Validate(); err != nil {
		return err
	}

	return nil
}

func (m *Proposal) Validate() error {
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
func (m *Proposal) UseSimulatedBackend(b bool) {
	m.useSimulatedBackend = b
}

// ChainSelectors returns a sorted list of chain selectors from the chains' metadata
func (m *Proposal) ChainSelectors() []types.ChainSelector {
	return slices.Sorted(maps.Keys(m.ChainMetadata))
}

// MerkleTree generates a merkle tree from the proposal's chain metadata and transactions.
func (m *Proposal) MerkleTree() (*merkle.Tree, error) {
	encoders, err := m.GetEncoders()
	if err != nil {
		return nil, wrapTreeGenErr(err)
	}

	hashLeaves := make([]common.Hash, 0)
	for _, sel := range m.ChainSelectors() {
		// Since we create encoders from the list of chain selectors provided in the ChainMetadata,
		// we can be sure the encoder exists, and don't need to check for existence.
		encoder := encoders[sel]

		// Similarly, we can be sure the metadata exists, as we iterate over the chain selectors,
		// since the chain selectors are keys in the ChainMetadata map.
		metadata := m.ChainMetadata[sel]

		encodedRootMetadata, encerr := encoder.HashMetadata(metadata)
		if encerr != nil {
			return nil, wrapTreeGenErr(encerr)
		}

		hashLeaves = append(hashLeaves, encodedRootMetadata)
	}

	for i, tx := range m.Transactions {
		txNonces, txerr := m.TransactionNonces()
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
			m.ChainMetadata[tx.ChainSelector],
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

func (m *Proposal) SigningHash() (common.Hash, error) {
	tree, err := m.MerkleTree()
	if err != nil {
		return common.Hash{}, err
	}

	// Convert validUntil to [32]byte
	var validUntilBytes [32]byte
	binary.BigEndian.PutUint32(validUntilBytes[28:], m.ValidUntil) // Place the uint32 in the last 4 bytes

	hashToSign := crypto.Keccak256Hash(tree.Root.Bytes(), validUntilBytes[:])

	return toEthSignedMessageHash(hashToSign), nil
}

// We may need to put this back in
// func (e *Executor) SigningMessage() ([]byte, error) {
// 	return ABIEncode(`[{"type":"bytes32"},{"type":"uint32"}]`, s.Tree.Root, s.Proposal.ValidUntil)
// }

// TransactionCounts returns a map of chain selectors to the number of transactions for that chain
func (m *Proposal) TransactionCounts() map[types.ChainSelector]uint64 {
	txCounts := make(map[types.ChainSelector]uint64)
	for _, tx := range m.Transactions {
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
func (m *Proposal) TransactionNonces() ([]uint64, error) {
	// Map to keep track of local index counts for each ChainSelector
	chainIndexMap := make(map[types.ChainSelector]uint64, len(m.ChainMetadata))

	txNonces := make([]uint64, len(m.Transactions))
	for i, tx := range m.Transactions {
		// Get the current local index for this ChainSelector
		localIndex := chainIndexMap[tx.ChainSelector]

		// Lookup the StartingOpCount for this ChainSelector from cmMap
		md, ok := m.ChainMetadata[tx.ChainSelector]
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

// AppendSignature appends a signature to the proposal's signature list.
func (m *Proposal) AppendSignature(signature types.Signature) {
	m.Signatures = append(m.Signatures, signature)
}

// GetEncoders generates encoders for each chain in the proposal's chain metadata.
func (m *Proposal) GetEncoders() (map[types.ChainSelector]sdk.Encoder, error) {
	txCounts := m.TransactionCounts()
	encoders := make(map[types.ChainSelector]sdk.Encoder)
	for chainSelector := range m.ChainMetadata {
		encoder, err := newEncoder(chainSelector, txCounts[chainSelector], m.OverridePreviousRoot, m.useSimulatedBackend)
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
