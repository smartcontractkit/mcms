package mcms

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/internal/core/merkle"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var (
	ErrInspectorsNotProvided = errors.New("inspectors not provided")
	ErrSimulatorsNotProvided = errors.New("simulators not provided")
)

// Signable provides signing functionality for an Proposal. It contains all the necessary
// information required to validate, sign, and check the quorum of a proposal.

// Signable contains the proposal itself, a Merkle tree representation of the proposal, encoders for
// different chains to perform the signing, while the inspectors are used for retrieving contract
// configurations and operational counts on chain.
type Signable struct {
	proposal   *Proposal
	tree       *merkle.Tree
	txNonces   []uint64
	encoders   map[types.ChainSelector]sdk.Encoder
	inspectors map[types.ChainSelector]sdk.Inspector
	simulators map[types.ChainSelector]sdk.Simulator
}

// NewSignable creates a new Signable from a proposal and inspectors, and initializes the encoders
// and merkle tree.
func NewSignable(
	proposal *Proposal,
	inspectors map[types.ChainSelector]sdk.Inspector,
) (*Signable, error) {
	encoders, err := proposal.GetEncoders()
	if err != nil {
		return nil, err
	}

	tree, err := proposal.MerkleTree()
	if err != nil {
		return nil, err
	}

	// Generate the tx nonces from the proposal
	txNonces, err := proposal.TransactionNonces()
	if err != nil {
		return nil, err
	}

	return &Signable{
		proposal:   proposal,
		tree:       tree,
		encoders:   encoders,
		inspectors: inspectors,
		txNonces:   txNonces,
	}, nil
}

// Sign signs the root of the proposal's Merkle tree with the provided signer.
func (s *Signable) Sign(signer signer) (sig types.Signature, err error) {
	// Validate proposal
	if err = s.proposal.Validate(); err != nil {
		return sig, err
	}

	// Get the signing hash
	payload, err := s.proposal.SigningHash()
	if err != nil {
		return sig, err
	}

	// Sign the payload
	sigB, err := signer.Sign(payload.Bytes())
	if err != nil {
		return sig, err
	}

	return types.NewSignatureFromBytes(sigB)
}

// SignAndAppend signs the proposal using the provided signer and appends the resulting signature
// to the proposal's list of signatures.
//
// This function modifies the proposal in place by adding the new signature to its Signatures
// slice.
func (s *Signable) SignAndAppend(signer signer) (types.Signature, error) {
	// Sign the proposal
	sig, err := s.Sign(signer)
	if err != nil {
		return types.Signature{}, err
	}

	// Add the signature to the proposal
	s.proposal.AppendSignature(sig)

	return sig, nil
}

// Simulate simulates the proposal on the given chain using the provided simulator.
func (s *Signable) Simulate() error {
	if s.simulators == nil {
		return ErrSimulatorsNotProvided
	}

	for _, op := range s.proposal.Operations {
		simulator, ok := s.simulators[op.ChainSelector]
		if !ok {
			fmt.Printf("simulator not found for chain %d", op.ChainSelector)
			return fmt.Errorf("simulator not found for chain %d", op.ChainSelector)
		}
		fmt.Printf("Simulating operation on chain %d", op.ChainSelector)

		// TODO: should we fail on the first error or aggregate all simulation errors?
		err := simulator.SimulateOperation(s.proposal.ChainMetadata[op.ChainSelector], op)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetConfigs retrieves the MCMS contract configurations for each chain in the proposal.
func (s *Signable) GetConfigs() (map[types.ChainSelector]*types.Config, error) {
	if s.inspectors == nil {
		return nil, ErrInspectorsNotProvided
	}

	configs := make(map[types.ChainSelector]*types.Config)
	for chain, metadata := range s.proposal.ChainMetadata {
		inspector, ok := s.inspectors[chain]
		if !ok {
			return nil, fmt.Errorf("inspector not found for chain %d", chain)
		}

		configuration, err := inspector.GetConfig(metadata.MCMAddress)
		if err != nil {
			return nil, err
		}

		configs[chain] = configuration
	}

	return configs, nil
}

// CheckQuorum checks if the quorum for the proposal on the given chain has been reached. This will
// fetch the current configuration for the chain and check if the recovered signers from the
// proposal's signatures can set the root.
func (s *Signable) CheckQuorum(chain types.ChainSelector) (bool, error) {
	if s.inspectors == nil {
		return false, ErrInspectorsNotProvided
	}

	inspector, ok := s.inspectors[chain]
	if !ok {
		return false, errors.New("inspector not found for chain " + strconv.FormatUint(uint64(chain), 10))
	}

	hash, err := s.proposal.SigningHash()
	if err != nil {
		return false, err
	}

	recoveredSigners := make([]common.Address, len(s.proposal.Signatures))
	for i, sig := range s.proposal.Signatures {
		recoveredAddr, rerr := sig.Recover(hash)
		if rerr != nil {
			return false, rerr
		}

		recoveredSigners[i] = recoveredAddr
	}

	configuration, err := inspector.GetConfig(s.proposal.ChainMetadata[chain].MCMAddress)
	if err != nil {
		return false, err
	}

	return configuration.CanSetRoot(recoveredSigners)
}

// ValidateSignatures checks if the quorum for the proposal has been reached on the MCM contracts
// across all chains in the proposal.
func (s *Signable) ValidateSignatures() (bool, error) {
	for chain := range s.proposal.ChainMetadata {
		checkQuorum, err := s.CheckQuorum(chain)
		if err != nil {
			return false, err
		}

		if !checkQuorum {
			return false, NewQuorumNotReachedError(chain)
		}
	}

	return true, nil
}

// ValidateConfigs checks the MCMS contract configurations for each chain in the proposal for
// consistency.
//
// We expect that the configurations for each chain are the same so that the same quorum can be
// reached across all chains in the proposal.
func (s *Signable) ValidateConfigs() error {
	configs, err := s.GetConfigs()
	if err != nil {
		return err
	}

	for i, sel := range s.proposal.ChainSelectors() {
		if i == 0 {
			continue
		}

		if !configs[sel].Equals(configs[s.proposal.ChainSelectors()[i-1]]) {
			return &InconsistentConfigsError{
				ChainSelectorA: sel,
				ChainSelectorB: s.proposal.ChainSelectors()[i-1],
			}
		}
	}

	return nil
}

// getCurrentOpCounts returns the current op counts for the MCM contract on each chain in the
// proposal. This data is fetched from the contract on the chain using the provided inspectors.
//
// Note: This function is currently not used but left for potential future use.
func (s *Signable) getCurrentOpCounts() (map[types.ChainSelector]uint64, error) {
	if s.inspectors == nil {
		return nil, ErrInspectorsNotProvided
	}

	opCounts := make(map[types.ChainSelector]uint64)
	for sel, metadata := range s.proposal.ChainMetadata {
		inspector, ok := s.inspectors[sel]
		if !ok {
			return nil, fmt.Errorf("inspector not found for chain %d", sel)
		}

		opCount, err := inspector.GetOpCount(metadata.MCMAddress)
		if err != nil {
			return nil, err
		}

		opCounts[sel] = opCount
	}

	return opCounts, nil
}
