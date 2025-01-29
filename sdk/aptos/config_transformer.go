package aptos

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

type ConfigTransformer struct {
	evmTransformer evm.ConfigTransformer
}

func NewConfigTransformer() *ConfigTransformer { return &ConfigTransformer{} }

// The Aptos API returns theses as snake_case which isn't recognized by mapstructure library
// TODO: Use custom MatchName field in DecoderConfig to re-use existing structs?

type ManyChainMultiSigConfig struct {
	GroupParents [32]uint8 `mapstructure:"group_parents"`
	GroupQuorums [32]uint8 `mapstructure:"group_quorums"`
	Signers      []struct {
		Address common.Address `mapstructure:"addr"`
		Group   uint8
		Index   uint8
	}
}

type ManyChainMultiSigRootMetadata struct {
	ChainID              uint64 `mapstructure:"chain_id"`
	Multisig             string `mapstructure:"multisig"`
	OverridePreviousRoot bool   `mapstructure:"override_previous_root"`
	PostOpCount          uint64 `mapstructure:"post_op_count"`
	PreOpCount           uint64 `mapstructure:"pre_op_count"`
}

func (e *ConfigTransformer) ToConfig(config ManyChainMultiSigConfig) (*types.Config, error) {
	// Re-using the EVM implementation here, but need to convert input first
	evmConfig := bindings.ManyChainMultiSigConfig{
		Signers:      nil,
		GroupQuorums: config.GroupQuorums,
		GroupParents: config.GroupParents,
	}

	for _, signer := range config.Signers {
		evmConfig.Signers = append(evmConfig.Signers, bindings.ManyChainMultiSigSigner{
			Addr:  signer.Address,
			Index: signer.Index,
			Group: signer.Group,
		})
	}
	return e.evmTransformer.ToConfig(evmConfig)
}
