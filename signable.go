package mcms

import (
	"errors"
	"sort"
	"strconv"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/internal/core/merkle"
	coreProposal "github.com/smartcontractkit/mcms/internal/core/proposal"
	"github.com/smartcontractkit/mcms/internal/utils/safecast"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

type Signable struct {
	*MCMSProposal
	*merkle.Tree

	// Map of operation to chain index where tx i is the ChainNonce[i]th
	// operation of chain Transaction[i].ChainSelector
	ChainNonces []uint64

	Encoders   map[types.ChainSelector]sdk.Encoder
	Inspectors map[types.ChainSelector]sdk.Inspector // optional, skip any inspections
	Simulators map[types.ChainSelector]sdk.Simulator // optional, skip simulations
	Decoders   map[types.ChainSelector]sdk.Decoder   // optional, skip decoding
}

var _ coreProposal.Signable = (*Signable)(nil)

func NewSignable(
	proposal *MCMSProposal,
	encoders map[types.ChainSelector]sdk.Encoder,
	inspectors map[types.ChainSelector]sdk.Inspector,
) (*Signable, error) {
	hashLeaves := make([]common.Hash, 0)
	chainIdx := make(map[types.ChainSelector]uint64, len(proposal.ChainMetadata))

	for _, chain := range proposal.ChainSelectors() {
		encoder, ok := encoders[chain]
		if !ok {
			return nil, errors.New("encoder not provided for chain " + strconv.FormatUint(uint64(chain), 10))
		}

		metadata, ok := proposal.ChainMetadata[chain]
		if !ok {
			return nil, errors.New("metadata not provided for chain " + strconv.FormatUint(uint64(chain), 10))
		}

		encodedRootMetadata, err := encoder.HashMetadata(metadata)
		if err != nil {
			return nil, err
		}
		hashLeaves = append(hashLeaves, encodedRootMetadata)

		// Set the initial chainIdx to the starting nonce in the metadata
		chainIdx[chain] = metadata.StartingOpCount
	}

	chainNonces := make([]uint64, len(proposal.Transactions))
	for i, op := range proposal.Transactions {
		chainNonce, err := safecast.Uint64ToUint32(chainIdx[op.ChainSelector])
		if err != nil {
			return nil, err
		}

		encoder, ok := encoders[op.ChainSelector]
		if !ok {
			return nil, errors.New("encoder not provided for chain " + strconv.FormatUint(uint64(op.ChainSelector), 10))
		}

		encodedOp, err := encoder.HashOperation(
			chainNonce,
			proposal.ChainMetadata[op.ChainSelector],
			op,
		)
		if err != nil {
			return nil, err
		}
		hashLeaves = append(hashLeaves, encodedOp)

		// Increment chain idx
		chainNonces[i] = chainIdx[op.ChainSelector]
		chainIdx[op.ChainSelector]++
	}

	// sort the hashes and sort the pairs
	sort.Slice(hashLeaves, func(i, j int) bool {
		return hashLeaves[i].String() < hashLeaves[j].String()
	})

	return &Signable{
		MCMSProposal: proposal,
		Tree:         merkle.NewTree(hashLeaves),
		Encoders:     encoders,
		Inspectors:   inspectors,
		ChainNonces:  chainNonces,
	}, nil
}

func (s *Signable) GetTree() *merkle.Tree {
	return s.Tree
}

func (s *Signable) ChainNonce(index int) uint64 {
	return s.ChainNonces[index]
}

func (s *Signable) GetCurrentOpCounts() (map[types.ChainSelector]uint64, error) {
	if s.Inspectors == nil {
		return nil, errors.New("inspectors not provided")
	}

	opCounts := make(map[types.ChainSelector]uint64)
	for chain, metadata := range s.ChainMetadata {
		inspector, ok := s.Inspectors[chain]
		if !ok {
			return nil, errors.New("inspector not found for chain " + strconv.FormatUint(uint64(chain), 10))
		}

		opCount, err := inspector.GetOpCount(metadata.MCMAddress)
		if err != nil {
			return nil, err
		}

		opCounts[chain] = opCount
	}

	return opCounts, nil
}

func (s *Signable) GetConfigs() (map[types.ChainSelector]*types.Config, error) {
	if s.Inspectors == nil {
		return nil, errors.New("inspectors not provided")
	}

	configs := make(map[types.ChainSelector]*types.Config)
	for chain, metadata := range s.ChainMetadata {
		inspector, ok := s.Inspectors[chain]
		if !ok {
			return nil, errors.New("inspector not found for chain " + strconv.FormatUint(uint64(chain), 10))
		}

		configuration, err := inspector.GetConfig(metadata.MCMAddress)
		if err != nil {
			return nil, err
		}

		configs[chain] = configuration
	}

	return configs, nil
}

func (s *Signable) CheckQuorum(chain types.ChainSelector) (bool, error) {
	if s.Inspectors == nil {
		return false, errors.New("inspectors not provided")
	}

	inspector, ok := s.Inspectors[chain]
	if !ok {
		return false, errors.New("inspector not found for chain " + strconv.FormatUint(uint64(chain), 10))
	}

	hash, err := s.SigningHash()
	if err != nil {
		return false, err
	}

	recoveredSigners := make([]common.Address, len(s.Signatures))
	for i, sig := range s.Signatures {
		recoveredAddr, rerr := sig.Recover(hash)
		if rerr != nil {
			return false, rerr
		}

		recoveredSigners[i] = recoveredAddr
	}

	configuration, err := inspector.GetConfig(s.ChainMetadata[chain].MCMAddress)
	if err != nil {
		return false, err
	}

	return configuration.CanSetRoot(recoveredSigners)
}

func (s *Signable) ValidateSignatures() (bool, error) {
	for chain := range s.ChainMetadata {
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

func (s *Signable) ValidateConfigs() error {
	configs, err := s.GetConfigs()
	if err != nil {
		return err
	}

	for i, chain := range s.ChainSelectors() {
		if i == 0 {
			continue
		}

		if !configs[chain].Equals(configs[s.ChainSelectors()[i-1]]) {
			return &InconsistentConfigsError{
				ChainSelectorA: chain,
				ChainSelectorB: s.ChainSelectors()[i-1],
			}
		}
	}

	return nil
}
