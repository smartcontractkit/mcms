package evm

import (
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/internal/utils/safecast"
	sdkerrors "github.com/smartcontractkit/mcms/sdk/errors"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

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
) (bindings.ManyChainMultiSigConfig, error) {
	var bindConfig bindings.ManyChainMultiSigConfig

	groupQuorums, groupParents, signerAddrs, signerGroups, err := ExtractSetConfigInputs(&cfg)
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
		idx += 1
	}

	return bindings.ManyChainMultiSigConfig{
		GroupQuorums: groupQuorums,
		GroupParents: groupParents,
		Signers:      bindSigners,
	}, nil
}

func ExtractSetConfigInputs(
	group *types.Config,
) ([32]uint8, [32]uint8, []common.Address, []uint8, error) {
	var groupQuorums, groupParents, signerGroups = []uint8{}, []uint8{}, []uint8{}
	var signerAddrs = []common.Address{}

	err := extractGroupsAndSigners(group, 0, &groupQuorums, &groupParents, &signerAddrs, &signerGroups)
	if err != nil {
		return [32]uint8{}, [32]uint8{}, []common.Address{}, []uint8{}, err
	}

	// fill the rest of the arrays with 0s
	for i := len(groupQuorums); i < 32; i++ {
		groupQuorums = append(groupQuorums, 0)
		groupParents = append(groupParents, 0)
	}

	// Combine SignerAddresses and SignerGroups into a slice of Signer structs
	bindSigners := make([]bindings.ManyChainMultiSigSigner, len(signerAddrs))
	for i := range signerAddrs {
		bindSigners[i] = bindings.ManyChainMultiSigSigner{
			Addr:  signerAddrs[i],
			Group: signerGroups[i],
		}
	}

	// Sort signers by their addresses in ascending order
	slices.SortFunc(bindSigners, func(i, j bindings.ManyChainMultiSigSigner) int {
		addressA := new(big.Int).SetBytes(i.Addr.Bytes())
		addressB := new(big.Int).SetBytes(j.Addr.Bytes())

		return addressA.Cmp(addressB)
	})

	// Extract the ordered addresses and groups after sorting
	orderedSignerAddresses := make([]common.Address, len(signerAddrs))
	orderedSignerGroups := make([]uint8, len(signerAddrs))
	for i, signer := range bindSigners {
		orderedSignerAddresses[i] = signer.Addr
		orderedSignerGroups[i] = signer.Group
	}

	return [32]uint8(groupQuorums), [32]uint8(groupParents), orderedSignerAddresses, orderedSignerGroups, nil
}

func extractGroupsAndSigners(
	group *types.Config,
	parentIdx uint8,
	groupQuorums *[]uint8,
	groupParents *[]uint8,
	signerAddrs *[]common.Address,
	signerGroups *[]uint8,
) error {
	// Append the group's quorum and parent index to the respective slices
	*groupQuorums = append(*groupQuorums, group.Quorum)
	*groupParents = append(*groupParents, parentIdx)

	// Assign the current group index
	currentGroupIdx := len(*groupQuorums) - 1

	// Safe to cast currentGroupIdx to uint8
	currentGroupIdxUint8, err := safecast.IntToUint8(currentGroupIdx)
	if err != nil {
		return fmt.Errorf("group index %d exceeds uint8 range", currentGroupIdx)
	}

	// For each string signer, append the signer and its group index
	for _, signerAddr := range group.Signers {
		*signerAddrs = append(*signerAddrs, signerAddr)
		*signerGroups = append(*signerGroups, currentGroupIdxUint8)
	}

	// Recursively handle the nested multisig groups
	for _, groupSigner := range group.GroupSigners {
		if err := extractGroupsAndSigners(&groupSigner, currentGroupIdxUint8, groupQuorums, groupParents, signerAddrs, signerGroups); err != nil {
			return err
		}
	}

	return nil
}
