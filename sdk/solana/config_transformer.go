package solana

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	bindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"

	sdkerrors "github.com/smartcontractkit/mcms/sdk/errors"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

type AdditionalConfig struct {
	ChainID    uint64
	MultisigID [32]uint8
	// Current Owner of the multisig ID
	Owner solana.PublicKey
	// Proposed Owner of the program when calling transfer_ownership
	ProposedOwner solana.PublicKey
}

const maxUint8Value = 255

type ConfigTransformer struct{}

func NewConfigTransformer() *ConfigTransformer {
	return &ConfigTransformer{}
}

// ToConfig converts an Solana ManyChainMultiSigConfig to a chain-agnostic types.Config
func (e *ConfigTransformer) ToConfig(
	bindConfig *bindings.MultisigConfig,
) (*types.Config, error) {
	groupToSigners := make([][]common.Address, len(bindConfig.GroupQuorums))
	for _, signer := range bindConfig.Signers {
		groupToSigners[signer.Group] = append(groupToSigners[signer.Group], signer.EvmAddress)
	}

	groups := make([]types.Config, len(bindConfig.GroupQuorums))
	for i, quorum := range bindConfig.GroupQuorums {
		signers := groupToSigners[i]
		if signers == nil {
			signers = []common.Address{}
		}

		groups[i] = types.Config{
			Signers:      signers,
			GroupSigners: []types.Config{},
			Quorum:       quorum,
		}
	}

	// link the group signers; this assumes a group's parent always has a lower index
	for i := 31; i >= 0; i-- {
		parent := bindConfig.GroupParents[i]
		if i > 0 && groups[i].Quorum > 0 {
			groups[parent].GroupSigners = append([]types.Config{groups[i]}, groups[parent].GroupSigners...)
		}
	}

	if err := groups[0].Validate(); err != nil {
		return nil, err
	}

	return &groups[0], nil
}

// ToChainConfig converts a chain-agnostic types.Config to an Solana ManyChainMultiSigConfig
func (e *ConfigTransformer) ToChainConfig(cfg types.Config, solanaConfig AdditionalConfig) (bindings.MultisigConfig, error) {
	// Populate additional Configs
	result := bindings.MultisigConfig{
		ChainId:       solanaConfig.ChainID,
		MultisigId:    solanaConfig.MultisigID,
		Owner:         solanaConfig.Owner,
		ProposedOwner: solanaConfig.ProposedOwner,
	}
	// Populate the signers: we can reuse the evm implementation here as the signers structure is the same
	groupQuorums, groupParents, signerAddrs, signerGroups, err := evm.ExtractSetConfigInputs(&cfg)
	if err != nil {
		return bindings.MultisigConfig{}, err
	}
	// Check the length of signerAddresses up-front
	if len(signerAddrs) > maxUint8Value {
		return bindings.MultisigConfig{}, sdkerrors.NewTooManySignersError(uint64(len(signerAddrs)))
	}
	// Set the signers
	bindSigners := make([]bindings.McmSigner, len(signerAddrs))
	idx := uint8(0)
	for i, signerAddr := range signerAddrs {
		bindSigners[i] = bindings.McmSigner{
			EvmAddress: signerAddr,
			Group:      signerGroups[i],
			Index:      idx,
		}
		idx += 1
	}
	result.Signers = bindSigners
	// Set group quorums and group parents.
	result.GroupQuorums = groupQuorums
	result.GroupParents = groupParents

	return result, nil
}
