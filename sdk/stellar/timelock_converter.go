package stellar

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"
	"github.com/stellar/go-stellar-sdk/xdr"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.TimelockConverter = (*TimelockConverter)(nil)

const timelockDomain = "RBAC_TIMELOCK_DOMAIN_SEPARATOR_BATCH_STELLAR"

type TimelockConverter struct{}

func NewTimelockConverter() *TimelockConverter { return &TimelockConverter{} }

func (t *TimelockConverter) ConvertBatchToChainOperations(_ context.Context, _ types.ChainMetadata, batch types.BatchOperation, timelockAddress, mcmAddress string, delay types.Duration, action types.TimelockAction, predecessor, salt common.Hash) ([]types.Operation, common.Hash, error) {
	if len(batch.Transactions) == 0 {
		return nil, common.Hash{}, fmt.Errorf("empty Stellar timelock batch")
	}
	calls := make([]xdr.ScVal, 0, len(batch.Transactions))
	callHashes := make([]common.Hash, 0, len(batch.Transactions))
	for _, tx := range batch.Transactions {
		payload, err := DecodeSorobanInvokePayload(tx.Data)
		if err != nil {
			return nil, common.Hash{}, err
		}
		target, err := parseContractID(tx.To)
		if err != nil {
			return nil, common.Hash{}, err
		}
		call, err := scval.BuildStructScVal(map[string]xdr.ScVal{"target": scval.AddressToScVal(tx.To), "function": scval.SymbolToScVal(payload.Function), "args_xdr": scval.BytesToScVal(payload.ArgsXDR)})
		if err != nil {
			return nil, common.Hash{}, err
		}
		calls = append(calls, call)
		callHashes = append(callHashes, hashTimelockCall([32]byte(target), payload.Function, payload.ArgsXDR))
	}
	operationID := hashTimelockBatch(callHashes, predecessor, salt)
	callsVal, err := scval.BuildStructScVal(map[string]xdr.ScVal{"inner": scval.VecToScVal(calls)})
	if err != nil {
		return nil, common.Hash{}, err
	}
	var fn string
	var args []xdr.ScVal
	switch action {
	case types.TimelockActionSchedule:
		fn = "schedule_batch"
		args = []xdr.ScVal{scval.AddressToScVal(mcmAddress), callsVal, scval.Bytes32ToScVal([32]byte(predecessor)), scval.Bytes32ToScVal([32]byte(salt)), scval.Uint64ToScVal(uint64(delay.Seconds()))}
	case types.TimelockActionCancel:
		fn = "cancel"
		args = []xdr.ScVal{scval.AddressToScVal(mcmAddress), scval.Bytes32ToScVal([32]byte(operationID))}
	case types.TimelockActionBypass:
		fn = "bypasser_execute_batch"
		args = []xdr.ScVal{scval.AddressToScVal(mcmAddress), callsVal}
	default:
		return nil, common.Hash{}, fmt.Errorf("unsupported Stellar timelock action %s", action)
	}
	tx, err := NewTransaction(timelockAddress, fn, args, "RBACTimelock", nil)
	if err != nil {
		return nil, common.Hash{}, err
	}

	return []types.Operation{{ChainSelector: batch.ChainSelector, Transaction: tx}}, operationID, nil
}

func hashTimelockCall(target [32]byte, function string, args []byte) common.Hash {
	f, _ := scval.SymbolToScVal(function).MarshalBinary()
	b := make([]byte, 0, 32+4+len(f)+4+len(args))
	b = append(b, target[:]...)
	var n [4]byte
	binary.BigEndian.PutUint32(n[:], uint32(len(f)))
	b = append(b, n[:]...)
	b = append(b, f...)
	binary.BigEndian.PutUint32(n[:], uint32(len(args)))
	b = append(b, n[:]...)
	b = append(b, args...)

	return crypto.Keccak256Hash(b)
}

func hashTimelockBatch(calls []common.Hash, predecessor, salt common.Hash) common.Hash {
	domain := crypto.Keccak256([]byte(timelockDomain))
	b := append([]byte{}, domain...)
	var n [4]byte
	binary.BigEndian.PutUint32(n[:], uint32(len(calls)))
	b = append(b, n[:]...)
	for _, h := range calls {
		b = append(b, h[:]...)
	}
	b = append(b, predecessor[:]...)
	b = append(b, salt[:]...)

	return crypto.Keccak256Hash(b)
}

func OperationID(batch types.BatchOperation, _ types.TimelockAction, predecessor, salt common.Hash) (common.Hash, error) {
	if len(batch.Transactions) == 0 {
		return common.Hash{}, fmt.Errorf("empty Stellar timelock batch")
	}
	hashes := make([]common.Hash, 0, len(batch.Transactions))
	for _, tx := range batch.Transactions {
		p, err := DecodeSorobanInvokePayload(tx.Data)
		if err != nil {
			return common.Hash{}, err
		}
		target, err := parseContractID(tx.To)
		if err != nil {
			return common.Hash{}, err
		}
		hashes = append(hashes, hashTimelockCall([32]byte(target), p.Function, p.ArgsXDR))
	}

	return hashTimelockBatch(hashes, predecessor, salt), nil
}
