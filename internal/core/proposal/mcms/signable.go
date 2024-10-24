package mcms

import (
	"encoding/binary"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/mcms/internal/core"
	"github.com/smartcontractkit/mcms/internal/core/config"
	"github.com/smartcontractkit/mcms/internal/core/merkle"
)

type Signable struct {
	*MCMSProposal
	*merkle.Tree

	Encoders   map[ChainSelector]Encoder
	Inspectors *map[ChainSelector]Inspector // optional, skip any inspections
	Simulators *map[ChainSelector]Simulator // optional, skip simulations
	Decoders   *map[ChainSelector]Decoder   // optional, skip decoding
}

func (e *Signable) SigningHash() (common.Hash, error) {
	// Convert validUntil to [32]byte
	var validUntilBytes [32]byte
	binary.BigEndian.PutUint32(validUntilBytes[28:], e.ValidUntil) // Place the uint32 in the last 4 bytes

	hashToSign := crypto.Keccak256Hash(e.Tree.Root.Bytes(), validUntilBytes[:])

	return toEthSignedMessageHash(hashToSign), nil
}

// func (e *Executor) SigningMessage() ([]byte, error) {
// 	return ABIEncode(`[{"type":"bytes32"},{"type":"uint32"}]`, e.Tree.Root, e.Proposal.ValidUntil)
// }

func toEthSignedMessageHash(messageHash common.Hash) common.Hash {
	// Add the Ethereum signed message prefix
	prefix := []byte("\x19Ethereum Signed Message:\n32")
	data := append(prefix, messageHash.Bytes()...)

	// Hash the prefixed message
	return crypto.Keccak256Hash(data)
}

func (e *Signable) GetCurrentOpCounts() (map[ChainSelector]uint64, error) {
	if e.Inspectors == nil {
		return nil, errors.New("inspectors not provided")
	}

	opCounts := make(map[ChainSelector]uint64)
	for chain, metadata := range e.ChainMetadata {
		inspector, ok := (*e.Inspectors)[chain]
		if !ok {
			return nil, errors.New("inspector not found for chain " + string(chain))
		}

		opCount, err := inspector.GetOpCount(metadata.MCMAddress)
		if err != nil {
			return nil, err
		}

		opCounts[chain] = opCount
	}

	return opCounts, nil
}

func (e *Signable) GetConfigs() (map[ChainSelector]*config.Config, error) {
	if e.Inspectors == nil {
		return nil, errors.New("inspectors not provided")
	}

	configs := make(map[ChainSelector]*config.Config)
	for chain, metadata := range e.ChainMetadata {
		inspector, ok := (*e.Inspectors)[chain]
		if !ok {
			return nil, errors.New("inspector not found for chain " + string(chain))
		}

		config, err := inspector.GetConfig(metadata.MCMAddress)
		if err != nil {
			return nil, err
		}

		configs[chain] = config
	}

	return configs, nil
}

func (e *Signable) CheckQuorum(chain ChainSelector) (bool, error) {
	if e.Inspectors == nil {
		return false, errors.New("inspectors not provided")
	}

	inspector, ok := (*e.Inspectors)[chain]
	if !ok {
		return false, errors.New("inspector not found for chain " + string(chain))
	}

	hash, err := e.SigningHash()
	if err != nil {
		return false, err
	}

	recoveredSigners := make([]common.Address, len(e.Signatures))
	for i, sig := range e.Signatures {
		recoveredAddr, rerr := sig.Recover(hash)
		if rerr != nil {
			return false, rerr
		}

		recoveredSigners[i] = recoveredAddr
	}

	config, err := inspector.GetConfig(e.ChainMetadata[chain].MCMAddress)
	if err != nil {
		return false, err
	}

	return config.CanSetRoot(recoveredSigners)
}

func (e *Signable) ValidateSignatures() (bool, error) {
	for chain := range e.ChainMetadata {
		checkQuorum, err := e.CheckQuorum(chain)
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

func (e *Signable) ValidateConfigs() error {
	configs, err := e.GetConfigs()
	if err != nil {
		return err
	}

	for i, chain := range e.ChainIdentifiers() {
		if i == 0 {
			continue
		}

		if !configs[chain].Equals(configs[e.ChainIdentifiers()[i-1]]) {
			return &core.InconsistentConfigsError{
				ChainIdentifierA: uint64(chain),
				ChainIdentifierB: uint64(e.ChainIdentifiers()[i-1]),
			}
		}
	}

	return nil
}
