package ton

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/mcms/sdk"
	sdkerrors "github.com/smartcontractkit/mcms/sdk/errors"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
)

func AsUnsigned(v *big.Int, sz uint) *big.Int {
	if sz == 0 {
		return new(big.Int)
	}
	mask := new(big.Int).Lsh(big.NewInt(1), sz)
	mask.Sub(mask, big.NewInt(1))
	return new(big.Int).And(v, mask) // interpret as uint sz
}

const maxUint8Value = 255

type ConfigTransformer = sdk.ConfigTransformer[mcms.Config, any]

var _ ConfigTransformer = &configTransformer{}

type configTransformer struct {
	evmTransformer evm.ConfigTransformer
}

func NewConfigTransformer() ConfigTransformer { return &configTransformer{} }

// ToChainConfig converts the chain agnostic config to the chain-specific config
func (e *configTransformer) ToChainConfig(cfg types.Config, _ any) (mcms.Config, error) {
	// Note: for TON, we will get the signer keys (public keys) instead of addresses
	// Re-using the EVM implementation here, we first need to map a set of signer keys to addresses
	// (by taking the first 20 bytes of the public key)

	// Extract the set config inputs using the EVM implementation
	keysMap := ConfigRemapSignerKeys(&cfg)
	groupQuorum, groupParents, signerAddrs, signerGroups, err := evm.ExtractSetConfigInputs(&cfg)
	if err != nil {
		return mcms.Config{}, fmt.Errorf("unable to extract set config inputs: %w", err)
	}

	// Check the length of signerAddresses up-front
	if len(signerAddrs) > maxUint8Value {
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
		// retrieve the public key corresponding to the address
		key := keysMap[signerAddr]
		signers[i] = mcms.Signer{
			Key:   new(big.Int).SetBytes(key),
			Group: signerGroups[i],
			Index: idx,
		}
		idx += 1
	}

	keySz := uint(8)
	signersDict := cell.NewDict(keySz)
	for i, s := range signers {
		sc, err := tlb.ToCell(s)
		if err != nil {
			return mcms.Config{}, fmt.Errorf("unable to encode signer %d: %w", i, err)
		}

		signersDict.SetIntKey(big.NewInt(int64(i)), sc)
	}

	sz := uint(8)
	gqDict := cell.NewDict(keySz)
	for i, g := range groupQuorum {
		if uint8(i) <= groupMax { // don't set unnecessary groups
			v := cell.BeginCell().MustStoreUInt(uint64(g), sz).EndCell()
			gqDict.SetIntKey(big.NewInt(int64(i)), v)
		}
	}

	gpDict := cell.NewDict(keySz)
	for i, g := range groupParents {
		if uint8(i) <= groupMax { // don't set unnecessary groups
			v := cell.BeginCell().MustStoreUInt(uint64(g), sz).EndCell()
			gpDict.SetIntKey(big.NewInt(int64(i)), v)
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

	// Note: for TON, we will get the signer keys (public keys) instead of addresses
	// Re-using the EVM implementation here, we first need to map a set of signer keys to addresses
	// (by taking the first 20 bytes of the public key)
	keysMap := make(map[common.Address][]byte)

	for i, kvSigner := range kvSigners {
		var signer mcms.Signer
		err = tlb.LoadFromCell(&signer, kvSigner.Value)
		if err != nil {
			return nil, fmt.Errorf("unable to decode signer: %w", err)
		}

		// TODO: big.Int loading doesn't work for me
		key := AsUnsigned(signer.Key, 256).Bytes()

		addr := common.Address(key[0:20])
		keysMap[addr] = key

		evmConfig.Signers[i] = bindings.ManyChainMultiSigSigner{
			Addr:  addr,
			Index: signer.Index,
			Group: signer.Group,
		}
	}

	kvGroupQuorums, err := config.GroupQuorums.LoadAll()
	if err != nil {
		return nil, fmt.Errorf("unable to laaoad group aa quorums: %w", err)
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

	outc, err := e.evmTransformer.ToConfig(evmConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to convert to SDK config type: %w", err)
	}

	// recursively map group signers' keys as well
	ConfigRemapSigners(outc, keysMap)

	return outc, nil
}

// Recursively remaps the SignerKeys field in the config into Signers by taking the first 20 bytes of each key
func ConfigRemapSignerKeys(cfg *types.Config) map[common.Address][]byte {
	var _remap func(cfg *types.Config)
	keysMap := make(map[common.Address][]byte)

	_remap = func(cfg *types.Config) {
		if cfg == nil {
			return
		}

		cfg.Signers = make([]common.Address, len(cfg.SignerKeys))
		for i, key := range cfg.SignerKeys {
			if len(key) < 20 {
				// pad zeros if key is less than 20 bytes
				paddedKey := make([]byte, 20)
				copy(paddedKey[20-len(key):], key)
				key = paddedKey
			}

			cfg.Signers[i] = common.Address(key[0:20])
			keysMap[cfg.Signers[i]] = key
		}

		for i := range cfg.GroupSigners {
			_remap(&cfg.GroupSigners[i])
		}
	}

	_remap(cfg)
	return keysMap
}

// Recursively remaps the Signers field in the config into SignerKeys based on the provided keysMap
func ConfigRemapSigners(cfg *types.Config, keysMap map[common.Address][]byte) {
	// recursively map group signers' keys as well
	var _remap func(cfg *types.Config)
	_remap = func(cfg *types.Config) {
		if cfg == nil {
			return
		}

		cfg.SignerKeys = make([][]byte, len(cfg.Signers))
		for i, signerAddr := range cfg.Signers {
			cfg.SignerKeys[i] = keysMap[signerAddr]
		}

		for i := range cfg.GroupSigners {
			_remap(&cfg.GroupSigners[i])
		}
	}
	_remap(cfg)
}
