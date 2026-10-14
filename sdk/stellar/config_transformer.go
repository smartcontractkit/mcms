package stellar

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	mcmsbindings "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/mcms"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.ConfigTransformer[*mcmsbindings.Config, any] = (*ConfigTransformer)(nil)

type ConfigTransformer struct{}

func NewConfigTransformer() *ConfigTransformer {
	return &ConfigTransformer{}
}

func (e *ConfigTransformer) ToConfig(cfg *mcmsbindings.Config) (*types.Config, error) {
	return toSDKConfig(cfg)
}

func (e *ConfigTransformer) ToChainConfig(cfg types.Config, _ any) (*mcmsbindings.Config, error) {
	signerAddresses, signerGroups, groupQuorums, groupParents, err := stellarSetConfigInputs(&cfg)
	if err != nil {
		return nil, err
	}

	signers := make([]mcmsbindings.Signer, len(signerAddresses.Inner))
	for i := range signerAddresses.Inner {
		signers[i] = mcmsbindings.Signer{
			Addr:  signerAddresses.Inner[i],
			Group: signerGroups.Inner[i],
			Index: uint32(i),
		}
	}

	return &mcmsbindings.Config{
		Signers:      signers,
		GroupQuorums: groupQuorums,
		GroupParents: groupParents,
	}, nil
}

// toSDKConfig converts the chainlink-stellar bindings Config type to the mcms SDK types.Config.
//
// On-chain Soroban stores: Vec<Signer{addr: [32]byte, group: u32, index: u32}>,
// group_quorums: [32]byte, group_parents: [32]byte. This mirrors the Solana flat layout
// and is converted to the hierarchical SDK types.Config tree the same way.
func toSDKConfig(cfg *mcmsbindings.Config) (*types.Config, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil on-chain config")
	}
	if len(cfg.Signers) == 0 {
		return nil, fmt.Errorf("on-chain config has no signers")
	}

	// Determine number of active groups (entries with non-zero quorum).
	numGroups := 0
	for i := range 32 {
		if cfg.GroupQuorums[i] != 0 {
			numGroups = i + 1
		}
	}
	if numGroups == 0 || cfg.GroupQuorums[0] == 0 {
		return nil, fmt.Errorf("on-chain config has no active root group")
	}

	// Build group→signers mapping.
	groupToSigners := make([][]common.Address, numGroups)
	for i := range groupToSigners {
		groupToSigners[i] = []common.Address{}
	}
	for _, s := range cfg.Signers {
		if int(s.Group) >= numGroups {
			return nil, fmt.Errorf("signer index %d belongs to inactive group %d", s.Index, s.Group)
		}
		if cfg.GroupQuorums[s.Group] == 0 {
			return nil, fmt.Errorf("signer index %d belongs to disabled group %d", s.Index, s.Group)
		}
		for _, prefix := range s.Addr[:12] {
			if prefix != 0 {
				return nil, fmt.Errorf("signer index %d is not a padded EVM address", s.Index)
			}
		}

		var addr common.Address
		copy(addr[:], s.Addr[12:]) // EVM addresses are 20 bytes padded to 32
		groupToSigners[s.Group] = append(groupToSigners[s.Group], addr)
	}

	// Build SDK group configs.
	groups := make([]types.Config, numGroups)
	for i := range numGroups {
		signers := groupToSigners[i]
		if signers == nil {
			signers = []common.Address{}
		}
		groups[i] = types.Config{
			Signers:      signers,
			GroupSigners: []types.Config{},
			Quorum:       cfg.GroupQuorums[i],
		}
	}

	// Link groups by parent index (reverse order so children exist when referenced).
	for i := numGroups - 1; i > 0; i-- {
		if groups[i].Quorum == 0 {
			continue
		}

		parent := cfg.GroupParents[i]
		if int(parent) >= i || groups[parent].Quorum == 0 {
			return nil, fmt.Errorf("active group %d has invalid parent %d", i, parent)
		}
		groups[parent].GroupSigners = append([]types.Config{groups[i]}, groups[parent].GroupSigners...)
	}

	if err := groups[0].Validate(); err != nil {
		return nil, fmt.Errorf("invalid on-chain config: %w", err)
	}

	return &groups[0], nil
}
