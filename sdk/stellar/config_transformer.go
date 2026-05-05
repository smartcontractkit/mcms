package stellar

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/sdk"
	sdkerrors "github.com/smartcontractkit/mcms/sdk/errors"
	"github.com/smartcontractkit/mcms/types"

	stellarmcms "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/mcms"
)

var _ sdk.ConfigTransformer[*stellarmcms.Config, any] = (*ConfigTransformer)(nil)

const maxUint8Value = 255

// ConfigTransformer maps Stellar MCMS on-chain config (get_config) to chain-agnostic types.Config.
type ConfigTransformer struct{}

// NewConfigTransformer returns a new Stellar config transformer.
func NewConfigTransformer() *ConfigTransformer {
	return &ConfigTransformer{}
}

// ToConfig converts a Stellar ManyChainMultiSig-style config to chain-agnostic types.Config.
func (e *ConfigTransformer) ToConfig(onchainConfig *stellarmcms.Config) (*types.Config, error) {
	if onchainConfig == nil {
		return nil, fmt.Errorf("nil config")
	}

	bindConfig := onchainConfig

	groupToSigners := make([][]common.Address, len(bindConfig.GroupQuorums))
	for _, signer := range bindConfig.Signers {
		addr := paddedBytes32ToCommonAddress(signer.Addr)
		groupToSigners[signer.Group] = append(groupToSigners[signer.Group], addr)
	}

	groups := make([]types.Config, len(bindConfig.GroupQuorums))
	for i := range bindConfig.GroupQuorums {
		quorum := bindConfig.GroupQuorums[i]

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

	// Link nested groups; assumes each group's parent index is lower than the child index.
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

// ToChainConfig converts chain-agnostic types.Config into the Stellar contract config shape.
func (e *ConfigTransformer) ToChainConfig(cfg types.Config, _ any) (*stellarmcms.Config, error) {
	groupQuorums, groupParents, signerAddrs, signerGroups, err := sdk.ExtractSetConfigInputs(&cfg)
	if err != nil {
		return nil, err
	}

	if len(signerAddrs) > maxUint8Value {
		return nil, sdkerrors.NewTooManySignersError(uint64(len(signerAddrs)))
	}

	out := &stellarmcms.Config{}
	copy(out.GroupQuorums[:], groupQuorums[:])
	copy(out.GroupParents[:], groupParents[:])

	out.Signers = make([]stellarmcms.Signer, len(signerAddrs))

	var idx uint32

	for i, signerAddr := range signerAddrs {
		out.Signers[i] = stellarmcms.Signer{
			Addr:  commonAddressToPaddedBytes32(signerAddr),
			Group: uint32(signerGroups[i]),
			Index: idx,
		}

		idx++
	}

	return out, nil
}

func commonAddressToPaddedBytes32(a common.Address) [32]byte {
	var out [32]byte
	copy(out[evmAddressABIWordLeadingZeroBytes:], a[:])

	return out
}

func paddedBytes32ToCommonAddress(b [32]byte) common.Address {
	var a common.Address
	copy(a[:], b[evmAddressABIWordLeadingZeroBytes:])

	return a
}
