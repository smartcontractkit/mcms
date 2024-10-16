package mcms

import (
	"errors"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/common"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
	mcm_errors "github.com/smartcontractkit/mcms/pkg/errors"
	"github.com/smartcontractkit/mcms/pkg/merkle"
)

// MCMSProposal is a struct where the target contract is an MCMS contract
// with no forwarder contracts. This type does not support any type of atomic contract
// call batching, as the MCMS contract natively doesn't support batching
type MCMSProposal struct {
	Version              string      `json:"version"`
	ValidUntil           uint32      `json:"validUntil"`
	Signatures           []Signature `json:"signatures"`
	OverridePreviousRoot bool        `json:"overridePreviousRoot"`

	// Map of chain identifier to chain metadata
	ChainMetadata map[ChainIdentifier]ChainMetadata `json:"chainMetadata"`

	// This is intended to be displayed as-is to signers, to give them
	// context for the change. File authors should templatize strings for
	// this purpose in their pipelines.
	Description string `json:"description"`

	// Operations to be executed
	Transactions []ChainOperation `json:"transactions"`
}

func NewProposal(
	version string,
	validUntil uint32,
	signatures []Signature,
	overridePreviousRoot bool,
	chainMetadata map[ChainIdentifier]ChainMetadata,
	description string,
	transactions []ChainOperation,
) (*MCMSProposal, error) {
	proposal := MCMSProposal{
		Version:              version,
		ValidUntil:           validUntil,
		Signatures:           signatures,
		OverridePreviousRoot: overridePreviousRoot,
		ChainMetadata:        chainMetadata,
		Description:          description,
		Transactions:         transactions,
	}

	err := proposal.Validate()
	if err != nil {
		return nil, err
	}

	return &proposal, nil
}

func NewProposalFromFile(filePath string) (*MCMSProposal, error) {
	var out MCMSProposal
	err := FromFile(filePath, &out)
	if err != nil {
		return nil, err
	}

	return &out, nil
}

func (m *MCMSProposal) Validate() error {
	if m.Version == "" {
		return &mcm_errors.ErrInvalidVersion{
			ReceivedVersion: m.Version,
		}
	}

	// Get the current Unix timestamp as an int64
	currentTime := time.Now().Unix()

	if m.ValidUntil <= uint32(currentTime) {
		// ValidUntil is a Unix timestamp, so it should be greater than the current time
		return &mcm_errors.ErrInvalidValidUntil{
			ReceivedValidUntil: m.ValidUntil,
		}
	}

	if len(m.ChainMetadata) == 0 {
		return &mcm_errors.ErrNoChainMetadata{}
	}

	if len(m.Transactions) == 0 {
		return &mcm_errors.ErrNoTransactions{}
	}

	if m.Description == "" {
		return &mcm_errors.ErrInvalidDescription{
			ReceivedDescription: m.Description,
		}
	}

	// Validate all chains in transactions have an entry in chain metadata
	for _, t := range m.Transactions {
		if _, ok := m.ChainMetadata[t.ChainID]; !ok {
			return &mcm_errors.ErrMissingChainDetails{
				ChainIdentifier: uint64(t.ChainID),
				Parameter:       "chain metadata",
			}
		}
	}

	return nil
}

func (m *MCMSProposal) TransactionCounts() map[ChainIdentifier]uint64 {
	txCounts := make(map[ChainIdentifier]uint64)
	for _, tx := range m.Transactions {
		txCounts[tx.ChainID]++
	}

	return txCounts
}

func (m *MCMSProposal) SortedChainIdentifiers() []ChainIdentifier {
	chainIdentifiers := make([]ChainIdentifier, 0, len(m.ChainMetadata))
	for chainID := range m.ChainMetadata {
		chainIdentifiers = append(chainIdentifiers, chainID)
	}
	sort.Slice(chainIdentifiers, func(i, j int) bool { return chainIdentifiers[i] < chainIdentifiers[j] })

	return chainIdentifiers
}

func (m *MCMSProposal) GetEncoders(isSim bool) (map[ChainIdentifier]MetadataEncoder, map[ChainIdentifier]OperationEncoder, error) {
	txCounts := m.TransactionCounts()
	metadataEncoders := make(map[ChainIdentifier]MetadataEncoder)
	chainOpEncoders := make(map[ChainIdentifier]OperationEncoder)
	for chainID, metadata := range m.ChainMetadata {
		chain, exists := chain_selectors.ChainBySelector(uint64(chainID))
		if !exists {
			return nil, nil, &mcm_errors.ErrInvalidChainID{
				ReceivedChainID: uint64(chainID),
			}
		}

		// Simulated chains always have block.chainid = 1337
		// So for setRoot to execute (not throw WrongChainId) we must
		// override the evmChainID to be 1337.
		if isSim {
			chain.EvmChainID = 1337
		}

		family, err := chain_selectors.GetSelectorFamily(uint64(chainID))
		if err != nil {
			return nil, nil, errors.New("unknown chain family")
		}

		switch family {
		case chain_selectors.FamilyEVM:
			metadataEncoders[chainID] = NewEVMMetadataEncoder(
				chain.EvmChainID,
				txCounts[chainID],
				m.OverridePreviousRoot,
			)

			chainOpEncoders[chainID] = NewEVMOperationEncoder(
				chain.EvmChainID,
				common.HexToAddress(metadata.MCMAddress),
			)
		case chain_selectors.FamilySolana:
			return nil, nil, errors.New("solana not supported")
		case chain_selectors.FamilyStarknet:
			return nil, nil, errors.New("starknet not supported")
		default:
			return nil, nil, errors.New("unsupported chain family")
		}
	}

	return metadataEncoders, chainOpEncoders, nil
}

func (m *MCMSProposal) ToSignable(sim bool) (*Signable, error) {
	sortedChainIdentifiers := m.SortedChainIdentifiers()
	metadataEncoders, operationsEncoders, err := m.GetEncoders(sim)
	if err != nil {
		return nil, err
	}

	hashLeaves := make([]common.Hash, 0)                               // represents all leaves in the tree
	chainIdx := make(map[ChainIdentifier]uint64, len(m.ChainMetadata)) // tracks the current nonce for a given chain

	txNonces := make([]uint64, len(m.Transactions))                               // stores the chain nonce for each tx
	metadataHashes := make(map[ChainIdentifier]common.Hash, len(m.ChainMetadata)) // stores the hash for the metadata of each chain
	opHashes := make([]common.Hash, len(m.Transactions))                          // stores the hash for each operation at index i

	for _, chainID := range sortedChainIdentifiers {
		encodedRootMetadata, err := metadataEncoders[chainID].Hash(m.ChainMetadata[chainID])
		if err != nil {
			return nil, err
		}
		hashLeaves = append(hashLeaves, encodedRootMetadata)
		metadataHashes[chainID] = encodedRootMetadata

		// Set the initial chainIdx to the starting nonce in the metadata
		chainIdx[chainID] = m.ChainMetadata[chainID].StartingOpCount
	}

	for i, op := range m.Transactions {
		encodedOp, err := operationsEncoders[op.ChainID].Hash(chainIdx[op.ChainID], op)
		if err != nil {
			return nil, err
		}
		hashLeaves = append(hashLeaves, encodedOp)
		opHashes[i] = encodedOp
		txNonces[i] = chainIdx[op.ChainID]

		// Increment chain idx
		chainIdx[op.ChainID]++
	}

	// sort the hashes and sort the pairs
	sort.Slice(hashLeaves, func(i, j int) bool {
		return hashLeaves[i].String() < hashLeaves[j].String()
	})

	tree := merkle.NewMerkleTree(hashLeaves)

	return NewSignable(m, tree, metadataHashes, opHashes, txNonces), nil
}

func (m *MCMSProposal) ToExecutor(sim bool) (*Executor, error) {
	signable, err := m.ToSignable(sim)
	if err != nil {
		return nil, err
	}

	// Create a new executor
	executor, err := NewProposalExecutor(signable, sim)
	if err != nil {
		return nil, err
	}

	return executor, nil
}

func (m *MCMSProposal) AddSignature(signature Signature) {
	m.Signatures = append(m.Signatures, signature)
}
