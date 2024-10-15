package config

import (
	"fmt"

	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"

	"github.com/spf13/cast"

	"github.com/smartcontractkit/mcms/pkg/errors"
	"github.com/smartcontractkit/mcms/pkg/gethwrappers"
)

const maxUint8Value = 255

// Config is a struct that holds all the configuration for the owner contracts
type Config struct {
	Quorum uint8 `json:"quorum"`

	// TODO: how should this change as we expand to other non-EVM chains?
	Signers      []common.Address `json:"signers"`
	GroupSigners []Config         `json:"groupSigners"`
}

func NewConfig(quorum uint8, signers []common.Address, groupSigners []Config) (*Config, error) {
	config := Config{
		Quorum:       quorum,
		Signers:      signers,
		GroupSigners: groupSigners,
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

func NewConfigFromRaw(rawConfig gethwrappers.ManyChainMultiSigConfig) (*Config, error) {
	groupToSigners := make([][]common.Address, len(rawConfig.GroupQuorums))
	for _, signer := range rawConfig.Signers {
		groupToSigners[signer.Group] = append(groupToSigners[signer.Group], signer.Addr)
	}

	groups := make([]Config, len(rawConfig.GroupQuorums))
	for i, quorum := range rawConfig.GroupQuorums {
		signers := groupToSigners[i]
		if signers == nil {
			signers = []common.Address{}
		}

		groups[i] = Config{
			Signers:      signers,
			GroupSigners: []Config{},
			Quorum:       quorum,
		}
	}

	for i, parent := range rawConfig.GroupParents {
		if i > 0 && groups[i].Quorum > 0 {
			groups[parent].GroupSigners = append(groups[parent].GroupSigners, groups[i])
		}
	}

	if errValidate := groups[0].Validate(); errValidate != nil {
		return nil, errValidate
	}

	return &groups[0], nil
}

func (c *Config) Validate() error {
	if c.Quorum == 0 {
		return &errors.InvalidMCMSConfigError{
			Reason: "Quorum must be greater than 0",
		}
	}

	if len(c.Signers) == 0 && len(c.GroupSigners) == 0 {
		return &errors.InvalidMCMSConfigError{
			Reason: "Config must have at least one signer or group",
		}
	}

	if (len(c.Signers) + len(c.GroupSigners)) < int(c.Quorum) {
		return &errors.InvalidMCMSConfigError{
			Reason: "Quorum must be less than or equal to the number of signers and groups",
		}
	}

	for _, groupSigner := range c.GroupSigners {
		if err := groupSigner.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) ToRawConfig() (gethwrappers.ManyChainMultiSigConfig, error) {
	groupQuorums, groupParents, signerAddresses, signerGroups, errSetConfig := c.ExtractSetConfigInputs()
	if errSetConfig != nil {
		return gethwrappers.ManyChainMultiSigConfig{}, errSetConfig
	}
	// Check the length of signerAddresses up-front
	if len(signerAddresses) > maxUint8Value+1 {
		return gethwrappers.ManyChainMultiSigConfig{}, &errors.TooManySignersError{NumSigners: uint64(len(signerAddresses))}
	}
	// convert to gethwrappers types
	signers := make([]gethwrappers.ManyChainMultiSigSigner, len(signerAddresses))
	idx := uint8(0)
	for i, signer := range signerAddresses {
		signers[i] = gethwrappers.ManyChainMultiSigSigner{
			Addr:  signer,
			Group: signerGroups[i],
			Index: idx,
		}
		idx += 1
	}

	return gethwrappers.ManyChainMultiSigConfig{
		GroupQuorums: groupQuorums,
		GroupParents: groupParents,
		Signers:      signers,
	}, nil
}

func (c *Config) Equals(other *Config) bool {
	if c.Quorum != other.Quorum {
		return false
	}

	if len(c.Signers) != len(other.Signers) {
		return false
	}

	// Compare signers (order doesn't matter)
	if !unorderedArrayEquals(c.Signers, other.Signers) {
		return false
	}

	if len(c.GroupSigners) != len(other.GroupSigners) {
		return false
	}

	// Compare all group signers in first exist in second (order doesn't matter)
	// the reverse is not necessary because the lengths are already checked
	for i := range c.GroupSigners {
		found := false
		for j := range other.GroupSigners {
			if c.GroupSigners[i].Equals(&other.GroupSigners[j]) {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

func (c *Config) ExtractSetConfigInputs() ([32]uint8, [32]uint8, []common.Address, []uint8, error) {
	var groupQuorums, groupParents, signerGroups = []uint8{}, []uint8{}, []uint8{}
	var signers = []common.Address{}

	errExtract := extractGroupsAndSigners(c, 0, &groupQuorums, &groupParents, &signers, &signerGroups)
	if errExtract != nil {
		return [32]uint8{}, [32]uint8{}, []common.Address{}, []uint8{}, errExtract
	}
	// fill the rest of the arrays with 0s
	for i := len(groupQuorums); i < 32; i++ {
		groupQuorums = append(groupQuorums, 0)
		groupParents = append(groupParents, 0)
	}

	// Combine SignerAddresses and SignerGroups into a slice of Signer structs
	signerObjs := make([]gethwrappers.ManyChainMultiSigSigner, len(signers))
	for i := range signers {
		signerObjs[i] = gethwrappers.ManyChainMultiSigSigner{
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

func extractGroupsAndSigners(group *Config, parentIdx uint8, groupQuorums *[]uint8, groupParents *[]uint8, signers *[]common.Address, signerGroups *[]uint8) error {
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

func unorderedArrayEquals[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}

	aMap := make(map[T]struct{})
	bMap := make(map[T]struct{})

	for _, i := range a {
		aMap[i] = struct{}{}
	}

	for _, i := range b {
		bMap[i] = struct{}{}
	}

	for _, i := range a {
		if _, ok := bMap[i]; !ok {
			return false
		}
	}

	for _, i := range b {
		if _, ok := aMap[i]; !ok {
			return false
		}
	}

	return true
}
