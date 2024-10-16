package mcms

import (
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	c "github.com/smartcontractkit/mcms/pkg/config"
	"github.com/smartcontractkit/mcms/pkg/errors"
	"github.com/smartcontractkit/mcms/pkg/gethwrappers"
	"github.com/smartcontractkit/mcms/pkg/merkle"
)

type Executor struct {
	Proposal           *MCMSProposal
	Tree               *merkle.MerkleTree
	MetadataEncoders   map[ChainIdentifier]MetadataEncoder
	OperationEncoders  map[ChainIdentifier]OperationEncoder
	MetadataExecutors  map[ChainIdentifier]MetadataExecutor
	OperationExecutors map[ChainIdentifier]OperationExecutor
	TxCounts           map[ChainIdentifier]uint64
	TxToChainNonce     map[int]uint32 // Maps transaction index to chain nonce
}

// NewProposalExecutor constructs a new Executor from a Proposal.
// The executor has all the relevant metadata for onchain execution.
// The sim flag indicates that this will be executed against a simulated chain (
// which has a chainID of 1337).
func NewProposalExecutor(proposal *MCMSProposal, sim bool) (*Executor, error) {
	txCounts := calculateTransactionCounts(proposal.Transactions)
	metadataEncoders, opEncoders, err := buildEncoders(proposal.ChainMetadata, sim)
	if err != nil {
		return nil, err
	}

	chainIdentifiers := sortedChainIdentifiers(proposal.ChainMetadata)
	tree, err := buildMerkleTree(
		chainIdentifiers,
		txCounts,
		metadataEncoders,
		opEncoders,
		proposal.ChainMetadata,
		proposal.Transactions,
		proposal.OverridePreviousRoot,
	)

	return &Executor{
		Proposal:          proposal,
		Tree:              tree,
		MetadataEncoders:  metadataEncoders,
		OperationEncoders: opEncoders,
		TxCounts:          txCounts,
	}, err
}

func (e *Executor) SigningHash() (common.Hash, error) {
	// Convert validUntil to [32]byte
	var validUntilBytes [32]byte
	binary.BigEndian.PutUint32(validUntilBytes[28:], e.Proposal.ValidUntil) // Place the uint32 in the last 4 bytes

	hashToSign := crypto.Keccak256Hash(e.Tree.Root.Bytes(), validUntilBytes[:])

	return toEthSignedMessageHash(hashToSign), nil
}

func (e *Executor) SigningMessage() ([]byte, error) {
	return ABIEncode(`[{"type":"bytes32"},{"type":"uint32"}]`, e.Tree.Root, e.Proposal.ValidUntil)
}

func toEthSignedMessageHash(messageHash common.Hash) common.Hash {
	// Add the Ethereum signed message prefix
	prefix := []byte("\x19Ethereum Signed Message:\n32")
	data := append(prefix, messageHash.Bytes()...)

	// Hash the prefixed message
	return crypto.Keccak256Hash(data)
}

func (e *Executor) ValidateMCMSConfigs(clients map[ChainIdentifier]ContractDeployBackend) error {
	configs, err := e.GetConfigs(clients)
	if err != nil {
		return err
	}

	wrappedConfigs, err := transformMCMSConfigs(configs)
	if err != nil {
		return err
	}

	// Validate that all configs are equivalent
	sortedChains := sortedChainIdentifiers(e.Proposal.ChainMetadata)
	for i, chain := range sortedChains {
		if i == 0 {
			continue
		}

		if !wrappedConfigs[chain].Equals(wrappedConfigs[sortedChains[i-1]]) {
			return &errors.ErrInconsistentConfigs{
				ChainIdentifierA: uint64(chain),
				ChainIdentifierB: uint64(sortedChains[i-1]),
			}
		}
	}

	return nil
}

func (e *Executor) GetCurrentOpCounts(clients map[ChainIdentifier]ContractDeployBackend) (map[ChainIdentifier]big.Int, error) {
	opCounts := make(map[ChainIdentifier]big.Int)

	callers, err := e.getMCMSCallers(clients)
	if err != nil {
		return nil, err
	}

	for chain, wrapper := range callers {
		opCount, err := wrapper.GetOpCount(&bind.CallOpts{})
		if err != nil {
			return nil, err
		}

		opCounts[chain] = *opCount
	}

	return opCounts, nil
}

func (e *Executor) GetConfigs(clients map[ChainIdentifier]ContractDeployBackend) (map[ChainIdentifier]gethwrappers.ManyChainMultiSigConfig, error) {
	configs := make(map[ChainIdentifier]gethwrappers.ManyChainMultiSigConfig)

	callers, err := e.getMCMSCallers(clients)
	if err != nil {
		return nil, err
	}

	for chain, wrapper := range callers {
		config, err := wrapper.GetConfig(&bind.CallOpts{})
		if err != nil {
			return nil, err
		}

		configs[chain] = config
	}

	return configs, nil
}

func (e *Executor) getMCMSCallers(clients map[ChainIdentifier]ContractDeployBackend) (map[ChainIdentifier]*gethwrappers.ManyChainMultiSig, error) {
	mcms := transformMCMAddresses(e.Proposal.ChainMetadata)
	mcmsWrappers := make(map[ChainIdentifier]*gethwrappers.ManyChainMultiSig)
	for chain, mcmAddress := range mcms {
		client, ok := clients[chain]
		if !ok {
			return nil, &errors.ErrMissingChainClient{
				ChainIdentifier: uint64(chain),
			}
		}

		mcms, err := gethwrappers.NewManyChainMultiSig(mcmAddress, client)
		if err != nil {
			return nil, err
		}

		mcmsWrappers[chain] = mcms
	}

	return mcmsWrappers, nil
}

func (e *Executor) CheckQuorum(client bind.ContractBackend, chain ChainIdentifier) (bool, error) {
	hash, err := e.SigningHash()
	if err != nil {
		return false, err
	}

	recoveredSigners := make([]common.Address, len(e.Proposal.Signatures))
	for i, sig := range e.Proposal.Signatures {
		recoveredAddr, rerr := sig.Recover(hash)
		if rerr != nil {
			return false, rerr
		}

		recoveredSigners[i] = recoveredAddr
	}

	// mcm, err := gethwrappers.NewManyChainMultiSig(e.RootMetadatas[chain].MultiSig, client)
	// if err != nil {
	// 	return false, err
	// }

	// config, err := mcm.GetConfig(&bind.CallOpts{})
	// if err != nil {
	// 	return false, err
	// }

	// spread the signers to get address from the configuration
	// contractSigners := make([]common.Address, 0, len(config.Signers))
	// for _, s := range config.Signers {
	// 	contractSigners = append(contractSigners, s.Addr)
	// }

	// Validate that all signers are valid
	// for _, signer := range recoveredSigners {
	// 	if !slices.Contains(contractSigners, signer) {
	// 		return false, &errors.ErrInvalidSignature{
	// 			ChainIdentifier:  uint64(chain),
	// 			MCMSAddress:      e.RootMetadatas[chain].MultiSig,
	// 			RecoveredAddress: signer,
	// 		}
	// 	}
	// }

	// Validate if the quorum is met

	// newConfig, err := c.NewConfigFromRaw(config)
	// if err != nil {
	// 	return false, err
	// }

	// if !isReadyToSetRoot(*newConfig, recoveredSigners) {
	// 	return false, &errors.ErrQuorumNotMet{
	// 		ChainIdentifier: uint64(chain),
	// 	}
	// }

	return true, nil
}

func (e *Executor) ValidateSignatures(clients map[ChainIdentifier]ContractDeployBackend) (bool, error) {
	hash, err := e.SigningHash()
	if err != nil {
		return false, err
	}

	recoveredSigners := make([]common.Address, len(e.Proposal.Signatures))
	for i, sig := range e.Proposal.Signatures {
		recoveredAddr, rerr := sig.Recover(hash)
		if rerr != nil {
			return false, rerr
		}

		recoveredSigners[i] = recoveredAddr
	}

	configs, err := e.GetConfigs(clients)
	if err != nil {
		return false, err
	}

	// Validate that all signers are valid
	for chain, config := range configs {
		for _, signer := range recoveredSigners {
			found := false
			for _, mcmsSigner := range config.Signers {
				if mcmsSigner.Addr == signer {
					found = true
					break
				}
			}

			if !found {
				return false, &errors.ErrInvalidSignature{
					ChainIdentifier:  uint64(chain),
					RecoveredAddress: signer,
				}
			}
		}
	}

	// Validate if the quorum is met
	wrappedConfigs, err := transformMCMSConfigs(configs)
	if err != nil {
		return false, err
	}

	for chain, config := range wrappedConfigs {
		if !isReadyToSetRoot(*config, recoveredSigners) {
			return false, &errors.ErrQuorumNotMet{
				ChainIdentifier: uint64(chain),
			}
		}
	}

	return true, nil
}

func isReadyToSetRoot(rootGroup c.Config, recoveredSigners []common.Address) bool {
	return isGroupAtConsensus(rootGroup, recoveredSigners)
}

func isGroupAtConsensus(group c.Config, recoveredSigners []common.Address) bool {
	signerApprovalsInGroup := 0
	for _, signer := range group.Signers {
		for _, recoveredSigner := range recoveredSigners {
			if signer == recoveredSigner {
				signerApprovalsInGroup++
				break
			}
		}
	}

	groupApprovals := 0
	for _, groupSigner := range group.GroupSigners {
		if isGroupAtConsensus(groupSigner, recoveredSigners) {
			groupApprovals++
		}
	}

	return (signerApprovalsInGroup + groupApprovals) >= int(group.Quorum)
}

func (e *Executor) SetRootOnChain(chain ChainIdentifier) error {
	metadata := e.Proposal.ChainMetadata[chain]

	metadataEncoder := e.MetadataEncoders[chain]
	encodedMetadata, err := metadataEncoder.Hash(metadata, e.TxCounts[chain], e.Proposal.OverridePreviousRoot)
	if err != nil {
		return err
	}

	proof, err := e.Tree.GetProof(encodedMetadata)
	if err != nil {
		return err
	}

	hash, err := e.SigningHash()
	if err != nil {
		return err
	}

	err = e.MetadataExecutors[chain].Execute(metadata, e.TxCounts[chain], e.Proposal.OverridePreviousRoot, e.Tree.Root, e.Proposal.ValidUntil, hash, e.Proposal.Signatures, proof)
	if err != nil {
		return err
	}

	return nil
}

func (e *Executor) ExecuteOnChain(idx int) error {
	mcmOperation := e.Proposal.Transactions[idx]
	hash, err := e.OperationEncoders[mcmOperation.ChainID].Hash(mcmOperation, e.TxToChainNonce[idx])
	if err != nil {
		return err
	}

	proof, err := e.Tree.GetProof(hash)
	if err != nil {
		return err
	}

	err = e.OperationExecutors[mcmOperation.ChainID].Execute(mcmOperation, e.TxToChainNonce[idx], proof)
	if err != nil {
		return err
	}

	return nil
}
