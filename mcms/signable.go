package mcms

import (
	"encoding/binary"
	"errors"
	"github.com/smartcontractkit/mcms/sdk"
	"sort"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartcontractkit/mcms/internal/core"
	"github.com/smartcontractkit/mcms/internal/core/config"
	"github.com/smartcontractkit/mcms/internal/core/merkle"
	coreProposal "github.com/smartcontractkit/mcms/internal/core/proposal"
	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
	"github.com/smartcontractkit/mcms/internal/utils/safecast"
	"github.com/smartcontractkit/mcms/types"
)

type Signable struct {
	*MCMSProposal
	*merkle.Tree

	// Map of operation to chain index where tx i is the ChainNonce[i]th
	// operation of chain Transaction[i].ChainSelector
	ChainNonces []uint64

	Encoders   map[types.ChainSelector]sdk.Encoder
	Inspectors map[types.ChainSelector]sdk.Inspector  // optional, skip any inspections
	Simulators map[types.ChainSelector]mcms.Simulator // optional, skip simulations
	Decoders   map[types.ChainSelector]mcms.Decoder   // optional, skip decoding
}

var _ coreProposal.Signable = (*Signable)(nil)

func NewSignable(
	proposal *MCMSProposal,
	encoders map[types.ChainSelector]sdk.Encoder,
	inspectors map[types.ChainSelector]sdk.Inspector,
) (*Signable, error) {
	hashLeaves := make([]common.Hash, 0)
	chainIdx := make(map[types.ChainSelector]uint64, len(proposal.ChainMetadata))

	for _, chain := range proposal.ChainIdentifiers() {
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

		encodedOp, err := encoders[op.ChainSelector].HashOperation(
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

func (s *Signable) SigningHash() (common.Hash, error) {
	// Convert validUntil to [32]byte
	var validUntilBytes [32]byte
	binary.BigEndian.PutUint32(validUntilBytes[28:], s.ValidUntil) // Place the uint32 in the last 4 bytes

	hashToSign := crypto.Keccak256Hash(s.Tree.Root.Bytes(), validUntilBytes[:])

	return toEthSignedMessageHash(hashToSign), nil
}

// func (e *Executor) SigningMessage() ([]byte, error) {
// 	return ABIEncode(`[{"type":"bytes32"},{"type":"uint32"}]`, s.Tree.Root, s.Proposal.ValidUntil)
// }

func toEthSignedMessageHash(messageHash common.Hash) common.Hash {
	// Add the Ethereum signed message prefix
	prefix := []byte("\x19Ethereum Signed Message:\n32")
	data := append(prefix, messageHash.Bytes()...)

	// Hash the prefixed message
	return crypto.Keccak256Hash(data)
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

func (s *Signable) GetConfigs() (map[types.ChainSelector]*config.Config, error) {
	if s.Inspectors == nil {
		return nil, errors.New("inspectors not provided")
	}

	configs := make(map[types.ChainSelector]*config.Config)
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
			return false, &core.QuorumNotMetError{
				ChainIdentifier: uint64(chain),
			}
		}
	}

	return true, nil
}

func (s *Signable) ValidateConfigs() error {
	configs, err := s.GetConfigs()
	if err != nil {
		return err
	}

	for i, chain := range s.ChainIdentifiers() {
		if i == 0 {
			continue
		}

		if !configs[chain].Equals(configs[s.ChainIdentifiers()[i-1]]) {
			return &core.InconsistentConfigsError{
				ChainIdentifierA: uint64(chain),
				ChainIdentifierB: uint64(s.ChainIdentifiers()[i-1]),
			}
		}
	}

	return nil
}
