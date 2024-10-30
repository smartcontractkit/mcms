package evm

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/mcms/internal/core"
	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
	"github.com/smartcontractkit/mcms/internal/core/proposal/timelock"
	"github.com/smartcontractkit/mcms/internal/evm/bindings"
)

var ZERO_HASH = common.Hash{}

type TimelockConverterEVM struct{}

func (t *TimelockConverterEVM) ConvertBatchToChainOperation(
	txn timelock.BatchChainOperation,
	timelockAddress common.Address,
	minDelay string,
	operation timelock.TimelockOperationType,
	predecessor common.Hash,
) (mcms.ChainOperation, common.Hash, error) {
	// Create the list of RBACTimelockCall (batch of calls) and tags for the operations
	calls := make([]bindings.RBACTimelockCall, 0)
	tags := make([]string, 0)
	for _, op := range txn.Batch {
		// Unmarshal the additional fields
		var additionalFields EVMAdditionalFields
		if err := json.Unmarshal(op.AdditionalFields, &additionalFields); err != nil {
			return mcms.ChainOperation{}, common.Hash{}, err
		}

		calls = append(calls, bindings.RBACTimelockCall{
			Target: common.HexToAddress(op.To),
			Data:   op.Data,
			Value:  additionalFields.Value,
		})
		tags = append(tags, op.Tags...)
	}

	salt := ZERO_HASH
	delay, _ := time.ParseDuration(minDelay)

	abi, errAbi := bindings.RBACTimelockMetaData.GetAbi()
	if errAbi != nil {
		return mcms.ChainOperation{}, common.Hash{}, errAbi
	}

	operationId, errHash := hashOperationBatch(calls, predecessor, salt)
	if errHash != nil {
		return mcms.ChainOperation{}, common.Hash{}, errHash
	}

	// Encode the data based on the operation
	var data []byte
	var err error
	switch operation {
	case timelock.Schedule:
		data, err = abi.Pack("scheduleBatch", calls, predecessor, salt, big.NewInt(int64(delay.Seconds())))
	case timelock.Cancel:
		data, err = abi.Pack("cancel", operationId)
	case timelock.Bypass:
		data, err = abi.Pack("bypasserExecuteBatch", calls)
	default:
		return mcms.ChainOperation{}, common.Hash{}, &core.InvalidTimelockOperationError{
			ReceivedTimelockOperation: string(operation),
		}
	}

	if err != nil {
		return mcms.ChainOperation{}, common.Hash{}, err
	}

	chainOperation := mcms.ChainOperation{
		ChainSelector: txn.ChainSelector,
		Operation: NewEVMOperation(
			timelockAddress,
			data,
			big.NewInt(0),
			"RBACTimelock",
			tags,
		),
	}

	return chainOperation, operationId, nil
}

// hashOperationBatch replicates the hash calculation from Solidity
// TODO: see if there's an easier way to do this using the gethwrappers
func hashOperationBatch(calls []bindings.RBACTimelockCall, predecessor, salt [32]byte) (common.Hash, error) {
	const abi = `[{"components":[{"internalType":"address","name":"target","type":"address"},{"internalType":"uint256","name":"value","type":"uint256"},{"internalType":"bytes","name":"data","type":"bytes"}],"internalType":"struct Call[]","name":"calls","type":"tuple[]"},{"internalType":"bytes32","name":"predecessor","type":"bytes32"},{"internalType":"bytes32","name":"salt","type":"bytes32"}]`
	encoded, err := ABIEncode(abi, calls, predecessor, salt)
	if err != nil {
		return common.Hash{}, err
	}

	// Return the hash as a [32]byte array
	return crypto.Keccak256Hash(encoded), nil
}
