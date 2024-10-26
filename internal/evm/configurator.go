package evm

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cast"

	"github.com/smartcontractkit/mcms/internal/core"
	"github.com/smartcontractkit/mcms/internal/core/config"
	"github.com/smartcontractkit/mcms/internal/evm/bindings"
)

const maxUint8Value = 255

type EVMConfigurator struct{}

func (e *EVMConfigurator) ToConfig(onchainConfig bindings.ManyChainMultiSigConfig) (*config.Config, error) {
	groupToSigners := make([][]common.Address, len(onchainConfig.GroupQuorums))
	for _, signer := range onchainConfig.Signers {
		groupToSigners[signer.Group] = append(groupToSigners[signer.Group], signer.Addr)
	}

	groups := make([]config.Config, len(onchainConfig.GroupQuorums))
	for i, quorum := range onchainConfig.GroupQuorums {
		signers := groupToSigners[i]
		if signers == nil {
			signers = []common.Address{}
		}

		groups[i] = config.Config{
			Signers:      signers,
			GroupSigners: []config.Config{},
			Quorum:       quorum,
		}
	}

	for i, parent := range onchainConfig.GroupParents {
		if i > 0 && groups[i].Quorum > 0 {
			groups[parent].GroupSigners = append(groups[parent].GroupSigners, groups[i])
		}
	}

	if errValidate := groups[0].Validate(); errValidate != nil {
		return nil, errValidate
	}

	return &groups[0], nil
}

func (e *EVMConfigurator) SetConfigInputs(configuration config.Config) (bindings.ManyChainMultiSigConfig, error) {
	groupQuorums, groupParents, signerAddresses, signerGroups, errSetConfig := ExtractSetConfigInputs(&configuration)
	if errSetConfig != nil {
		return bindings.ManyChainMultiSigConfig{}, errSetConfig
	}
	// Check the length of signerAddresses up-front
	if len(signerAddresses) > maxUint8Value+1 {
		return bindings.ManyChainMultiSigConfig{}, &core.TooManySignersError{NumSigners: uint64(len(signerAddresses))}
	}
	// convert to bindings types
	signers := make([]bindings.ManyChainMultiSigSigner, len(signerAddresses))
	idx := uint8(0)
	for i, signer := range signerAddresses {
		signers[i] = bindings.ManyChainMultiSigSigner{
			Addr:  signer,
			Group: signerGroups[i],
			Index: idx,
		}
		idx += 1
	}

	return bindings.ManyChainMultiSigConfig{
		GroupQuorums: groupQuorums,
		GroupParents: groupParents,
		Signers:      signers,
	}, nil
}

func ExtractSetConfigInputs(group *config.Config) ([32]uint8, [32]uint8, []common.Address, []uint8, error) {
	var groupQuorums, groupParents, signerGroups = []uint8{}, []uint8{}, []uint8{}
	var signers = []common.Address{}

	errExtract := extractGroupsAndSigners(group, 0, &groupQuorums, &groupParents, &signers, &signerGroups)
	if errExtract != nil {
		return [32]uint8{}, [32]uint8{}, []common.Address{}, []uint8{}, errExtract
	}
	// fill the rest of the arrays with 0s
	for i := len(groupQuorums); i < 32; i++ {
		groupQuorums = append(groupQuorums, 0)
		groupParents = append(groupParents, 0)
	}

	// Combine SignerAddresses and SignerGroups into a slice of Signer structs
	signerObjs := make([]bindings.ManyChainMultiSigSigner, len(signers))
	for i := range signers {
		signerObjs[i] = bindings.ManyChainMultiSigSigner{
			Addr:  signers[i],
			Group: signerGroups[i],
		}
	}

	// Sort signers by their addresses in ascending order
	sort.Slice(signerObjs, func(i, j int) bool {
		addressA := new(big.Int).SetBytes(signerObjs[i].Addr.Bytes())
		addressB := new(big.Int).SetBytes(signerObjs[j].Addr.Bytes())

		return addressA.Cmp(addressB) < 0
	})

	// Extract the ordered addresses and groups after sorting
	orderedSignerAddresses := make([]common.Address, len(signers))
	orderedSignerGroups := make([]uint8, len(signers))
	for i, signer := range signerObjs {
		orderedSignerAddresses[i] = signer.Addr
		orderedSignerGroups[i] = signer.Group
	}

	return [32]uint8(groupQuorums), [32]uint8(groupParents), orderedSignerAddresses, orderedSignerGroups, nil
}

func extractGroupsAndSigners(group *config.Config, parentIdx uint8, groupQuorums *[]uint8, groupParents *[]uint8, signers *[]common.Address, signerGroups *[]uint8) error {
	// Append the group's quorum and parent index to the respective slices
	*groupQuorums = append(*groupQuorums, group.Quorum)
	*groupParents = append(*groupParents, parentIdx)

	// Assign the current group index
	currentGroupIdx := len(*groupQuorums) - 1

	// Check if currentGroupIdx is within the uint8 range
	if currentGroupIdx > int(maxUint8Value) {
		return fmt.Errorf("group index %d exceeds uint8 range", currentGroupIdx)
	}

	// Safe to cast currentGroupIdx to uint8
	currentGroupIdxUint8 := cast.ToUint8(currentGroupIdx)

	// For each string signer, append the signer and its group index
	for _, signer := range group.Signers {
		*signers = append(*signers, signer)
		*signerGroups = append(*signerGroups, currentGroupIdxUint8)
	}

	// Recursively handle the nested multisig groups
	for _, groupSigner := range group.GroupSigners {
		if err := extractGroupsAndSigners(&groupSigner, currentGroupIdxUint8, groupQuorums, groupParents, signers, signerGroups); err != nil {
			return err
		}
	}

	return nil
}
