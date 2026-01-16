package evm

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/sdk"
	sdkerrors "github.com/smartcontractkit/mcms/sdk/errors"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.ConfigTransformer[bindings.ManyChainMultiSigConfig, any] = (*ConfigTransformer)(nil)

const maxUint8Value = 255

type ConfigTransformer struct{}

func NewConfigTransformer() *ConfigTransformer {
	return &ConfigTransformer{}
}

// ToConfig converts an EVM ManyChainMultiSigConfig to a chain-agnostic types.Config
func (e *ConfigTransformer) ToConfig(
	bindConfig bindings.ManyChainMultiSigConfig,
) (*types.Config, error) {
	groupToSigners := make([][]common.Address, len(bindConfig.GroupQuorums))
	for _, signer := range bindConfig.Signers {
		groupToSigners[signer.Group] = append(groupToSigners[signer.Group], signer.Addr)
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

// ToChainConfig converts a chain-agnostic types.Config to an EVM ManyChainMultiSigConfig
func (e *ConfigTransformer) ToChainConfig(
	cfg types.Config,
	_ any,
) (bindings.ManyChainMultiSigConfig, error) {
	var bindConfig bindings.ManyChainMultiSigConfig

	groupQuorums, groupParents, signerAddrs, signerGroups, err := sdk.ExtractSetConfigInputs(&cfg)
	if err != nil {
		return bindConfig, err
	}

	// Check the length of signerAddresses up-front
	if len(signerAddrs) > maxUint8Value {
		return bindConfig, sdkerrors.NewTooManySignersError(uint64(len(signerAddrs)))
	}

	// Convert to the binding config
	bindSigners := make([]bindings.ManyChainMultiSigSigner, len(signerAddrs))
	idx := uint8(0)
	for i, signerAddr := range signerAddrs {
		bindSigners[i] = bindings.ManyChainMultiSigSigner{
			Addr:  signerAddr,
			Group: signerGroups[i],
			Index: idx,
		}
		idx++
	}

	return bindings.ManyChainMultiSigConfig{
		GroupQuorums: groupQuorums,
		GroupParents: groupParents,
		Signers:      bindSigners,
	}, nil
}
