package mcms

import (
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	c "github.com/smartcontractkit/mcms/pkg/config"
	"github.com/smartcontractkit/mcms/pkg/errors"
	"github.com/smartcontractkit/mcms/pkg/gethwrappers"
)

type Executor struct {
	*Signable
	MetadataExecutors  map[ChainIdentifier]MetadataExecutor
	OperationExecutors map[ChainIdentifier]OperationExecutor
}

// NewProposalExecutor constructs a new Executor from a Proposal.
// The executor has all the relevant metadata for onchain execution.
// The sim flag indicates that this will be executed against a simulated chain (
// which has a chainID of 1337).
func NewProposalExecutor(signable *Signable, sim bool) (*Executor, error) {
	return &Executor{
		Signable:           signable,
		MetadataExecutors:  make(map[ChainIdentifier]MetadataExecutor),
		OperationExecutors: make(map[ChainIdentifier]OperationExecutor),
	}, nil
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
	sortedChains := e.Proposal.SortedChainIdentifiers()
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
	proof, err := e.Tree.GetProof(e.MetadataHashes[chain])
	if err != nil {
		return err
	}

	hash, err := e.SigningHash()
	if err != nil {
		return err
	}

	// Sort signatures by recovered address
	sortedSignatures := e.Proposal.Signatures
	sort.Slice(sortedSignatures, func(i, j int) bool {
		recoveredSignerA, _ := sortedSignatures[i].Recover(hash)
		recoveredSignerB, _ := sortedSignatures[j].Recover(hash)

		return recoveredSignerA.Cmp(recoveredSignerB) < 0
	})

	err = e.MetadataExecutors[chain].Execute(e.Proposal.ChainMetadata[chain], proof, e.Tree.Root, e.Proposal.ValidUntil, e.Proposal.Signatures)
	if err != nil {
		return err
	}

	return nil
}

func (e *Executor) ExecuteOnChain(idx int) error {
	mcmOperation := e.Proposal.Transactions[idx]

	proof, err := e.Tree.GetProof(e.OperationHashes[idx])
	if err != nil {
		return err
	}

	err = e.OperationExecutors[mcmOperation.ChainID].Execute(e.TxNonces[idx], proof, mcmOperation)
	if err != nil {
		return err
	}

	return nil
}
