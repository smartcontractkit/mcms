package solana

import (
	"github.com/ethereum/go-ethereum/common"
	bindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"

	"github.com/smartcontractkit/mcms/types"
)

const maxUint8Value = 255

type ConfigTransformer struct{}

func NewConfigTransformer() *ConfigTransformer {
	return &ConfigTransformer{}
}

// ToConfig converts an EVM ManyChainMultiSigConfig to a chain-agnostic types.Config
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

	for i, parent := range bindConfig.GroupParents {
		if i > 0 && groups[i].Quorum > 0 {
			groups[parent].GroupSigners = append(groups[parent].GroupSigners, groups[i])
		}
	}

	if err := groups[0].Validate(); err != nil {
		return nil, err
	}

	return &groups[0], nil
}

// ToChainConfig converts a chain-agnostic types.Config to an EVM ManyChainMultiSigConfig
func (e *ConfigTransformer) ToChainConfig(
	cfg types.Config,
) (bindings.MultisigConfig, error) {
	panic("implement me")
}
