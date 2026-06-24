package sdk

import (
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/internal/utils/safecast"
	"github.com/smartcontractkit/mcms/types"
)

// ExtractSetConfigInputs flattens a nested `*types.Config` into:
//  1. groupQuorums: [32]uint8 where each index i holds the quorum for group i (zero-padded).
//  2. groupParents: [32]uint8 where each index i holds the parent group’s index (or a sentinel).
//  3. orderedSignerAddresses: a sorted slice of all signer addresses.
//  4. orderedSignerGroups: a parallel slice of group indices for each signer.
//
// Returns an error if the structure cannot be flattened (e.g., too many groups).
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

	type signerWithGroup struct {
		addr  common.Address
		group uint8
	}

	// Combine signer addresses and groups so they can be sorted together.
	signers := make([]signerWithGroup, len(signerAddrs))
	for i := range signerAddrs {
		signers[i] = signerWithGroup{
			addr:  signerAddrs[i],
			group: signerGroups[i],
		}
	}

	// Sort signers by their addresses in ascending order
	slices.SortFunc(signers, func(i, j signerWithGroup) int {
		addressA := new(big.Int).SetBytes(i.addr.Bytes())
		addressB := new(big.Int).SetBytes(j.addr.Bytes())

		return addressA.Cmp(addressB)
	})

	// Extract the ordered addresses and groups after sorting
	orderedSignerAddresses := make([]common.Address, len(signerAddrs))
	orderedSignerGroups := make([]uint8, len(signerAddrs))
	for i, signer := range signers {
		orderedSignerAddresses[i] = signer.addr
		orderedSignerGroups[i] = signer.group
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
