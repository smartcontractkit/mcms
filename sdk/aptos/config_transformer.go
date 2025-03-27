package aptos

import (
	"github.com/ethereum/go-ethereum/common"

	module_mcms "github.com/smartcontractkit/chainlink-aptos/bindings/mcms/mcms"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

type ConfigTransformer struct {
	evmTransformer evm.ConfigTransformer
}

func NewConfigTransformer() *ConfigTransformer { return &ConfigTransformer{} }

func (e *ConfigTransformer) ToConfig(config module_mcms.Config) (*types.Config, error) {
	// Re-using the EVM implementation here, but need to convert input first
	evmConfig := bindings.ManyChainMultiSigConfig{
		Signers:      nil,
		GroupQuorums: [32]uint8(config.GroupQuorums),
		GroupParents: [32]uint8(config.GroupParents),
	}

	for _, signer := range config.Signers {
		evmConfig.Signers = append(evmConfig.Signers, bindings.ManyChainMultiSigSigner{
			Addr:  common.BytesToAddress(signer.Addr),
			Index: signer.Index,
			Group: signer.Group,
		})
	}

	return e.evmTransformer.ToConfig(evmConfig)
}
