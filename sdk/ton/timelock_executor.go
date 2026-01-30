package ton

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/timelock"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tlbe"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"
)

var _ sdk.TimelockExecutor = (*timelockExecutor)(nil)

// sdk.TimelockExecutor implementation for TON chains for accessing the RBACTimelock contract
type timelockExecutor struct {
	sdk.TimelockInspector

	client ton.APIClientWrapped
	wallet *wallet.Wallet

	// Transaction options
	amount tlb.Coins

	// Executor options
	wait bool
}

type TimelockExecutorOpts struct {
	Client ton.APIClientWrapped
	Wallet *wallet.Wallet

	// Value to send (to Timelock) with message
	Amount tlb.Coins
	// Whether to wait until the pending operation is finalized before trying to execute
	WaitPending *bool // default: true
}

// NewTimelockExecutor creates a new TimelockExecutor
func NewTimelockExecutor(opts TimelockExecutorOpts) (sdk.TimelockExecutor, error) {
	if lo.IsNil(opts.Client) {
		return nil, errors.New("failed to create sdk.Executor - client (ton.APIClientWrapped) is nil")
	}

	if opts.Wallet == nil {
		return nil, errors.New("failed to create sdk.Executor - wallet (*wallet.Wallet) is nil")
	}

	wait := true // default
	if opts.WaitPending != nil {
		wait = *opts.WaitPending
	}

	return &timelockExecutor{
		TimelockInspector: NewTimelockInspector(opts.Client),
		client:            opts.Client,
		wallet:            opts.Wallet,
		amount:            opts.Amount,
		wait:              wait,
	}, nil
}

func (e *timelockExecutor) Execute(
	ctx context.Context, bop types.BatchOperation, timelockAddress string, predecessor common.Hash, salt common.Hash,
) (types.TransactionResult, error) {
	var z types.TransactionResult // zero value

	// Map to Ton Address type
	dstAddr, err := address.ParseAddr(timelockAddress)
	if err != nil {
		return z, fmt.Errorf("invalid timelock address: %w", err)
	}

	calls, err := ConvertBatchToCalls(bop)
	if err != nil {
		return z, fmt.Errorf("failed to convert batch to calls: %w", err)
	}

	qID, err := tvm.RandomQueryID()
	if err != nil {
		return z, fmt.Errorf("failed to generate random query ID: %w", err)
	}

	body, err := tlb.ToCell(timelock.ExecuteBatch{
		QueryID: qID,

		Calls:       calls,
		Predecessor: tlbe.NewUint256(predecessor.Big()),
		Salt:        tlbe.NewUint256(salt.Big()),
	})
	if err != nil {
		return z, fmt.Errorf("failed to encode ExecuteBatch body: %w", err)
	}

	// Check the status of potential pending operation
	// Get current block
	blockID, err := e.client.CurrentMasterchainInfo(ctx)
	if err != nil {
		return z, fmt.Errorf("failed to get current block: %w", err)
	}

	// Load the full block to get timestamp and hash
	block, err := e.client.GetBlockData(ctx, blockID)
	if err != nil {
		return z, fmt.Errorf("failed to get block data: %w", err)
	}

	// Load the current on-chain time
	now := block.BlockInfo.GenUtime

	info, err := tvm.CallGetter(ctx, e.client, blockID, dstAddr, timelock.GetOpPendingInfo)
	if err != nil {
		return z, fmt.Errorf("failed to call timelock.GetOpPendingInfo getter: %w", err)
	}

	tx := TxOpts{
		Wallet:  e.wallet,
		DstAddr: dstAddr,
		Amount:  e.amount,
		Body:    body,
	}

	return SendTxAfter(ctx, tx, uint64(now), info.ValidAfter, DefaultWaitBuffer, e.wait)
}
