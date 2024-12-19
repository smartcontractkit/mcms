package mcms

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

// BatchToChainOperation converts a batch of chain operations to a single types.ChainOperation for
// different chains
func BatchToChainOperation(
	bops types.BatchOperation,
	timelockAddr string,
	delay types.Duration,
	action types.TimelockAction,
	predecessor common.Hash,
) (types.Operation, common.Hash, error) {
	chainFamily, err := types.GetChainSelectorFamily(bops.ChainSelector)
	if err != nil {
		return types.Operation{}, common.Hash{}, err
	}

	var converter sdk.TimelockConverter
	switch chainFamily {
	case cselectors.FamilyEVM:
		converter = &evm.TimelockConverter{}
	}

	return converter.ConvertBatchToChainOperation(
		bops, timelockAddr, delay, action, predecessor,
	)
}

// Applies the EIP191 prefix to the payload and hashes it.
func toEthSignedMessageHash(payload []byte) common.Hash {
	// Add the Ethereum signed message prefix
	prefix := []byte("\x19Ethereum Signed Message:\n32")
	data := append(prefix, payload...)

	// Hash the prefixed message
	return crypto.Keccak256Hash(data)
}
