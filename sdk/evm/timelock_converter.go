package evm

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	abiUtils "github.com/smartcontractkit/mcms/internal/utils/abi"
	"github.com/smartcontractkit/mcms/sdk"
	sdkerrors "github.com/smartcontractkit/mcms/sdk/errors"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

var ZERO_HASH = common.Hash{}

var _ sdk.TimelockConverter = (*TimelockConverter)(nil)

type TimelockConverter struct{}

func (t *TimelockConverter) ConvertBatchToChainOperation(
	bop types.BatchOperation,
	timelockAddress string,
	delay types.Duration,
	action types.TimelockAction,
	predecessor common.Hash,
) (types.Operation, common.Hash, error) {
	// Create the list of RBACTimelockCall (batch of calls) and tags for the operations
	calls := make([]bindings.RBACTimelockCall, 0)
	tags := make([]string, 0)
	for _, tx := range bop.Transactions {
		// Unmarshal the additional fields
		var additionalFields AdditionalFields
		if err := json.Unmarshal(tx.AdditionalFields, &additionalFields); err != nil {
			return types.Operation{}, common.Hash{}, err
		}

		calls = append(calls, bindings.RBACTimelockCall{
			Target: common.HexToAddress(tx.To),
			Data:   tx.Data,
			Value:  additionalFields.Value,
		})
		tags = append(tags, tx.Tags...)
	}

	salt := ZERO_HASH

	abi, errAbi := bindings.RBACTimelockMetaData.GetAbi()
	if errAbi != nil {
		return types.Operation{}, common.Hash{}, errAbi
	}

	operationId, errHash := HashOperationBatch(calls, predecessor, salt)
	if errHash != nil {
		return types.Operation{}, common.Hash{}, errHash
	}

	// Encode the data based on the operation
	var data []byte
	var err error
	switch action {
	case types.TimelockActionSchedule:
		data, err = abi.Pack("scheduleBatch", calls, predecessor, salt, big.NewInt(int64(delay.Seconds())))
	case types.TimelockActionCancel:
		data, err = abi.Pack("cancel", operationId)
	case types.TimelockActionBypass:
		data, err = abi.Pack("bypasserExecuteBatch", calls)
	default:
		return types.Operation{}, common.Hash{}, sdkerrors.NewInvalidTimelockOperationError(string(action))
	}

	if err != nil {
		return types.Operation{}, common.Hash{}, err
	}

	op := types.Operation{
		ChainSelector: bop.ChainSelector,
		Transaction: NewOperation(
			common.HexToAddress(timelockAddress),
			data,
			big.NewInt(0),
			"RBACTimelock",
			tags,
		),
	}

	return op, operationId, nil
}

// HashOperationBatch replicates the hash calculation from Solidity
// TODO: see if there's an easier way to do this using the gethwrappers
func HashOperationBatch(calls []bindings.RBACTimelockCall, predecessor, salt [32]byte) (common.Hash, error) {
	const abi = `[{"components":[{"internalType":"address","name":"target","type":"address"},{"internalType":"uint256","name":"value","type":"uint256"},{"internalType":"bytes","name":"data","type":"bytes"}],"internalType":"struct Call[]","name":"calls","type":"tuple[]"},{"internalType":"bytes32","name":"predecessor","type":"bytes32"},{"internalType":"bytes32","name":"salt","type":"bytes32"}]`
	encoded, err := abiUtils.ABIEncode(abi, calls, predecessor, salt)
	if err != nil {
		return common.Hash{}, err
	}

	// Return the hash as a [32]byte array
	return crypto.Keccak256Hash(encoded), nil
}
