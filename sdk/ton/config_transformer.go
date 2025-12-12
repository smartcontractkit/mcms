package ton

import (
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tlbe"

	"github.com/smartcontractkit/mcms/sdk"

	sdkerrors "github.com/smartcontractkit/mcms/sdk/errors"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

type ConfigTransformer = sdk.ConfigTransformer[mcms.Config, any]

var _ ConfigTransformer = &configTransformer{}

type configTransformer struct {
	evmTransformer evm.ConfigTransformer
}

func NewConfigTransformer() ConfigTransformer { return &configTransformer{} }

// ToChainConfig converts the chain agnostic config to the chain-specific config
func (e *configTransformer) ToChainConfig(cfg types.Config, _ any) (mcms.Config, error) {
	groupQuorum, groupParents, signerAddrs, signerGroups, err := evm.ExtractSetConfigInputs(&cfg)
	if err != nil {
		return mcms.Config{}, fmt.Errorf("unable to extract set config inputs: %w", err)
	}

	// Check the length of signerAddresses up-front
	if len(signerAddrs) > math.MaxUint8 {
		return mcms.Config{}, sdkerrors.NewTooManySignersError(uint64(len(signerAddrs)))
	}

	// Figure out the number of groups
	var groupMax uint8
	for _, v := range signerGroups {
		if v > groupMax {
			groupMax = v
		}
	}

	// Convert to the binding config
	signers := make([]mcms.Signer, len(signerAddrs))
	idx := uint8(0)
	for i, signerAddr := range signerAddrs {
		signers[i] = mcms.Signer{
			Address: tlbe.NewUint160(signerAddr.Big()), // represented as big.Int on TON
			Group:   signerGroups[i],
			Index:   idx,
		}
		idx++
	}

	keySz := uint(SizeUINT8)
	signersDict := cell.NewDict(keySz)
	for i, s := range signers {
		var sc *cell.Cell
		sc, err = tlb.ToCell(s)
		if err != nil {
			return mcms.Config{}, fmt.Errorf("unable to encode signer %d: %w", i, err)
		}

		err = signersDict.SetIntKey(big.NewInt(int64(i)), sc)
		if err != nil {
			return mcms.Config{}, fmt.Errorf("unable to dict.set signer %d: %w", i, err)
		}
	}

	sz := uint(SizeUINT8)
	gqDict := cell.NewDict(keySz)
	for i, g := range groupQuorum {
		//nolint:gosec // G115 conversion safe, max 32 groups
		if uint8(i) <= groupMax { // don't set unnecessary groups
			v := cell.BeginCell().MustStoreUInt(uint64(g), sz).EndCell()
			err = gqDict.SetIntKey(big.NewInt(int64(i)), v)
			if err != nil {
				return mcms.Config{}, fmt.Errorf("unable to dict.set group quorum %d: %w", i, err)
			}
		}
	}

	gpDict := cell.NewDict(keySz)
	for i, g := range groupParents {
		//nolint:gosec // G115 conversion safe, max 32 groups
		if uint8(i) <= groupMax { // don't set unnecessary groups
			v := cell.BeginCell().MustStoreUInt(uint64(g), sz).EndCell()
			err = gpDict.SetIntKey(big.NewInt(int64(i)), v)
			if err != nil {
				return mcms.Config{}, fmt.Errorf("unable to dict.set group parent %d: %w", i, err)
			}
		}
	}

	return mcms.Config{
		Signers:      signersDict,
		GroupQuorums: gqDict,
		GroupParents: gpDict,
	}, nil
}

// ToConfig Maps the chain-specific config to the chain-agnostic config
func (e *configTransformer) ToConfig(config mcms.Config) (*types.Config, error) {
	kvSigners, err := config.Signers.LoadAll()
	if err != nil {
		return nil, fmt.Errorf("unable to load signers: %w", err)
	}

	// Re-using the EVM implementation here, but need to convert input first
	evmConfig := bindings.ManyChainMultiSigConfig{
		Signers:      make([]bindings.ManyChainMultiSigSigner, len(kvSigners)),
		GroupQuorums: [32]uint8{},
		GroupParents: [32]uint8{},
	}

	for i, kvSigner := range kvSigners {
		var signer mcms.Signer
		err = tlb.LoadFromCell(&signer, kvSigner.Value)
		if err != nil {
			return nil, fmt.Errorf("unable to decode signer: %w", err)
		}

		addrBytes := make([]byte, common.AddressLength)
		signer.Address.Value().FillBytes(addrBytes) // TODO: tvm.KeyUINT160

		evmConfig.Signers[i] = bindings.ManyChainMultiSigSigner{
			Addr:  common.Address(addrBytes),
			Index: signer.Index,
			Group: signer.Group,
		}
	}

	kvGroupQuorums, err := config.GroupQuorums.LoadAll()
	if err != nil {
		return nil, fmt.Errorf("unable to load all group quorums: %w", err)
	}

	for i, kvGroupQuorum := range kvGroupQuorums {
		var val uint64
		val, err = kvGroupQuorum.Value.LoadUInt(SizeUINT8)
		if err != nil {
			return nil, fmt.Errorf("unable to load group quorum value: %w", err)
		}

		//nolint:gosec // G115 conversion safe
		evmConfig.GroupQuorums[i] = uint8(val)
	}

	kvGroupParents, err := config.GroupParents.LoadAll()
	if err != nil {
		return nil, fmt.Errorf("unable to load group parents: %w", err)
	}

	for i, kvGroupParent := range kvGroupParents {
		var val uint64
		val, err = kvGroupParent.Value.LoadUInt(SizeUINT8)
		if err != nil {
			return nil, fmt.Errorf("unable to load group parent value: %w", err)
		}

		//nolint:gosec // G115 conversion safe
		evmConfig.GroupParents[i] = uint8(val)
	}

	return e.evmTransformer.ToConfig(evmConfig)
}
