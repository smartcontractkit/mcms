package ton

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand/v2"

	"github.com/smartcontractkit/mcms/sdk"
	sdkerrors "github.com/smartcontractkit/mcms/sdk/errors"
	"github.com/smartcontractkit/mcms/types"

	"github.com/ethereum/go-ethereum/common"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/timelock"
	toncommon "github.com/smartcontractkit/chainlink-ton/pkg/ccip/bindings/common"
)

var _ sdk.TimelockConverter = (*timelockConverter)(nil)

type timelockConverter struct{}

// NewTimelockConverter creates a new TimelockConverter
func NewTimelockConverter() sdk.TimelockConverter {
	return &timelockConverter{}
}

func (t *timelockConverter) ConvertBatchToChainOperations(
	_ context.Context,
	metadata types.ChainMetadata,
	bop types.BatchOperation,
	timelockAddress string,
	mcmAddress string,
	delay types.Duration,
	action types.TimelockAction,
	predecessor common.Hash,
	salt common.Hash,
) ([]types.Operation, common.Hash, error) {
	// Create the list of RBACTimelockCall (batch of calls) and tags for the operations
	calls := make([]timelock.Call, 0)
	tags := make([]string, 0)
	for _, tx := range bop.Transactions {
		// TODO: duplicate code, refactor? (@see sdk/ton/timelock_executor.go)
		// Unmarshal the AdditionalFields from the operation
		var additionalFields AdditionalFields
		if err := json.Unmarshal(tx.AdditionalFields, &additionalFields); err != nil {
			return []types.Operation{}, common.Hash{}, fmt.Errorf("failed to unmarshal additional fields: %w", err)
		}

		// Map to Ton Address type
		to, err := address.ParseAddr(tx.To)
		if err != nil {
			return []types.Operation{}, common.Hash{}, fmt.Errorf("invalid target address: %w", err)
		}

		datac, err := cell.FromBOC(tx.Data)
		if err != nil {
			return []types.Operation{}, common.Hash{}, fmt.Errorf("invalid cell BOC data: %w", err)
		}

		calls = append(calls, timelock.Call{
			Target: to,
			Data:   datac,
			Value:  additionalFields.Value,
		})

		tags = append(tags, tx.Tags...)
	}

	operationId, errHash := HashOperationBatch(calls, predecessor, salt)
	if errHash != nil {
		return []types.Operation{}, common.Hash{}, errHash
	}

	// Encode the data based on the operation
	var data *cell.Cell
	var err error
	switch action {
	case types.TimelockActionSchedule:
		data, err = tlb.ToCell(timelock.ScheduleBatch{
			QueryID: rand.Uint64(),

			Calls:       toncommon.SnakeRef[timelock.Call](calls),
			Predecessor: predecessor.Big(),
			Salt:        salt.Big(),
			Delay:       uint32(delay.Seconds()),
		})
	case types.TimelockActionCancel:
		data, err = tlb.ToCell(timelock.Cancel{
			QueryID: rand.Uint64(),

			ID: operationId.Big(),
		})
	case types.TimelockActionBypass:
		data, err = tlb.ToCell(timelock.BypasserExecuteBatch{
			QueryID: rand.Uint64(),

			Calls: toncommon.SnakeRef[timelock.Call](calls),
		})
	default:
		return []types.Operation{}, common.Hash{}, sdkerrors.NewInvalidTimelockOperationError(string(action))
	}

	if err != nil {
		return []types.Operation{}, common.Hash{}, err
	}

	// Map to Ton Address type
	dstAddr, err := address.ParseAddr(timelockAddress)
	if err != nil {
		return []types.Operation{}, common.Hash{}, fmt.Errorf("invalid timelock address: %w", err)
	}

	// TODO: remove hardcoded value
	tx, err := NewTransaction(dstAddr, data.BeginParse(), big.NewInt(0), "RBACTimelock", tags)
	if err != nil {
		return []types.Operation{}, common.Hash{}, fmt.Errorf("failed to create transaction: %w", err)
	}

	op := types.Operation{
		ChainSelector: bop.ChainSelector,
		Transaction:   tx,
	}

	return []types.Operation{op}, operationId, nil
}

// HashOperationBatch replicates the hash calculation from Solidity
func HashOperationBatch(calls []timelock.Call, predecessor, salt common.Hash) (common.Hash, error) {
	ob, err := tlb.ToCell(timelock.OperationBatch{
		Calls:       toncommon.SnakeRef[timelock.Call](calls),
		Predecessor: predecessor.Big(),
		Salt:        salt.Big(),
	})
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to encode OperationBatch: %w", err)
	}

	// Return the hash as a [32]byte array
	return common.BytesToHash(ob.Hash()), nil
}
