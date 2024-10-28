package evm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/mcms/pkg/errors"
	owner "github.com/smartcontractkit/mcms/pkg/gethwrappers"
	"github.com/smartcontractkit/mcms/pkg/proposal/mcms"
	"github.com/smartcontractkit/mcms/pkg/proposal/timelock"
	"math/big"
	"time"
)

var ZERO_HASH = common.Hash{}

type TimelockConverterEVM struct{}

func (t *TimelockConverterEVM) ConvertBatchToChainOperation(
	txn timelock.BatchChainOperation,
	timelockAddress common.Address,
	minDelay string,
	operation timelock.TimelockOperation,
	predecessor common.Hash,
) (mcms.ChainOperation, common.Hash, error) {

	// Create the list of RBACTimelockCall (batch of calls) and tags for the operations
	calls := make([]owner.RBACTimelockCall, 0)
	tags := make([]string, 0)
	for _, op := range txn.Batch {
		calls = append(calls, owner.RBACTimelockCall{
			Target: op.To,
			Data:   op.Data,
			Value:  op.Value,
		})
		tags = append(tags, op.Tags...)
	}

	salt := ZERO_HASH
	delay, _ := time.ParseDuration(minDelay)

	abi, errAbi := owner.RBACTimelockMetaData.GetAbi()
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
		return mcms.ChainOperation{}, common.Hash{}, &errors.InvalidTimelockOperationError{
			ReceivedTimelockOperation: string(operation),
		}
	}

	if err != nil {
		return mcms.ChainOperation{}, common.Hash{}, err
	}

	chainOperation := mcms.ChainOperation{
		ChainIdentifier: txn.ChainIdentifier,
		Operation: mcms.Operation{
			To:           timelockAddress,
			Data:         data,
			Value:        big.NewInt(0), // TODO: is this right?
			ContractType: "RBACTimelock",
			Tags:         tags,
		},
	}

	return chainOperation, operationId, nil
}

// hashOperationBatch replicates the hash calculation from Solidity
// TODO: see if there's an easier way to do this using the gethwrappers
func hashOperationBatch(calls []owner.RBACTimelockCall, predecessor, salt [32]byte) (common.Hash, error) {
	const abi = `[{"components":[{"internalType":"address","name":"target","type":"address"},{"internalType":"uint256","name":"value","type":"uint256"},{"internalType":"bytes","name":"data","type":"bytes"}],"internalType":"struct Call[]","name":"calls","type":"tuple[]"},{"internalType":"bytes32","name":"predecessor","type":"bytes32"},{"internalType":"bytes32","name":"salt","type":"bytes32"}]`
	encoded, err := mcms.ABIEncode(abi, calls, predecessor, salt)
	if err != nil {
		return common.Hash{}, err
	}

	// Return the hash as a [32]byte array
	return crypto.Keccak256Hash(encoded), nil
}
