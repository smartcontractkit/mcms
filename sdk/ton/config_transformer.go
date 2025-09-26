package ton

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"

	"github.com/smartcontractkit/mcms/sdk"

	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

// TODO: move to github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms#Signer
// Signer information
type Signer struct {
	Key   *big.Int `tlb:"## 256"` // The public key of the signer.
	Index uint8    `tlb:"## 8"`   // The index of the signer in data.config.signers
	Group uint8    `tlb:"## 8"`   // 0 <= group < NUM_GROUPS. Each signer can only be in one group.
}

type ConfigTransformer = sdk.ConfigTransformer[mcms.Config, any]

var _ ConfigTransformer = &configTransformer{}

type configTransformer struct {
	evmTransformer evm.ConfigTransformer
}

func NewConfigTransformer() *configTransformer { return &configTransformer{} }

// ToChainConfig converts the chain agnostic config to the chain-specific config
func (e *configTransformer) ToChainConfig(cfg types.Config, _ any) (mcms.Config, error) {
	groupQuorum, groupParents, signerAddrs, signerGroups, err := evm.ExtractSetConfigInputs(&cfg)
	if err != nil {
		return mcms.Config{}, fmt.Errorf("unable to extract set config inputs: %w", err)
	}

	// Convert to the binding config
	signers := make([]Signer, len(signerAddrs))
	idx := uint8(0)
	for i, signerAddr := range signerAddrs {
		signers[i] = Signer{
			Key:   signerAddr.Big(),
			Group: signerGroups[i],
			Index: idx,
		}
		idx += 1
	}

	szSigner := uint(256 + 8 + 8)
	signersDict := cell.NewDict(szSigner)
	for i, s := range groupQuorum {
		sc, err := tlb.ToCell(s)
		if err != nil {
			return mcms.Config{}, fmt.Errorf("unable to encode signer %d: %w", i, err)
		}

		signersDict.SetIntKey(big.NewInt(int64(i)), sc)
	}

	sz := uint(8)
	gqDict := cell.NewDict(sz)
	for i, g := range groupQuorum {
		gqDict.SetIntKey(big.NewInt(int64(i)), cell.BeginCell().MustStoreUInt(uint64(g), sz).EndCell())
	}

	gpDict := cell.NewDict(sz)
	for i, g := range groupParents {
		gpDict.SetIntKey(big.NewInt(int64(i)), cell.BeginCell().MustStoreUInt(uint64(g), sz).EndCell())
	}

	return mcms.Config{
		Signers:      signersDict,
		GroupQuorums: gqDict,
		GroupParents: gpDict,
	}, nil
}

// ToConfig Maps the chain-specific config to the chain-agnostic config
func (e *configTransformer) ToConfig(config mcms.Config) (*types.Config, error) {
	// Re-using the EVM implementation here, but need to convert input first
	evmConfig := bindings.ManyChainMultiSigConfig{
		Signers:      make([]bindings.ManyChainMultiSigSigner, 0),
		GroupQuorums: [32]uint8{},
		GroupParents: [32]uint8{},
	}

	kvSigners, err := config.Signers.LoadAll()
	if err != nil {
		return nil, fmt.Errorf("unable to load signers: %w", err)
	}

	for _, kvSigner := range kvSigners {
		var signer Signer
		err = tlb.LoadFromCell(&signer, kvSigner.Value)
		if err != nil {
			return nil, fmt.Errorf("unable to decode signer: %w", err)
		}

		evmConfig.Signers = append(evmConfig.Signers, bindings.ManyChainMultiSigSigner{
			Addr:  common.BytesToAddress(signer.Key.Bytes()),
			Index: signer.Index,
			Group: signer.Group,
		})
	}

	kvGroupQuorums, err := config.GroupQuorums.LoadAll()
	if err != nil {
		return nil, fmt.Errorf("unable to load group quorums: %w", err)
	}

	for i, kvGroupQuorum := range kvGroupQuorums {
		val, err := kvGroupQuorum.Value.LoadUInt(8)
		if err != nil {
			return nil, fmt.Errorf("unable to load group quorum value: %w", err)
		}
		evmConfig.GroupQuorums[i] = uint8(val)
	}

	kvGroupParents, err := config.GroupParents.LoadAll()
	if err != nil {
		return nil, fmt.Errorf("unable to load group parents: %w", err)
	}

	for i, kvGroupParent := range kvGroupParents {
		val, err := kvGroupParent.Value.LoadUInt(8)
		if err != nil {
			return nil, fmt.Errorf("unable to load group parent value: %w", err)
		}
		evmConfig.GroupParents[i] = uint8(val)
	}

	return e.evmTransformer.ToConfig(evmConfig)
}
