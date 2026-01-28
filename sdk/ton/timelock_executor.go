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
	wallet *wallet.Wallet

	// Transaction opts
	amount tlb.Coins
}

type TimelockExecutorOpts struct {
	Client ton.APIClientWrapped
	Wallet *wallet.Wallet
	Amount tlb.Coins
}

// NewTimelockExecutor creates a new TimelockExecutor
func NewTimelockExecutor(opts TimelockExecutorOpts) (sdk.TimelockExecutor, error) {
	if lo.IsNil(opts.Client) {
		return nil, errors.New("failed to create sdk.Executor - client (ton.APIClientWrapped) is nil")
	}

	if opts.Wallet == nil {
		return nil, errors.New("failed to create sdk.Executor - wallet (*wallet.Wallet) is nil")
	}

	return &timelockExecutor{
		TimelockInspector: NewTimelockInspector(opts.Client),
		wallet:            opts.Wallet,
		amount:            opts.Amount,
	}, nil
}

func (e *timelockExecutor) Execute(
	ctx context.Context, bop types.BatchOperation, timelockAddress string, predecessor common.Hash, salt common.Hash,
) (types.TransactionResult, error) {
	// Map to Ton Address type
	dstAddr, err := address.ParseAddr(timelockAddress)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("invalid timelock address: %w", err)
	}

	calls, err := ConvertBatchToCalls(bop)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to convert batch to calls: %w", err)
	}

	qID, err := tvm.RandomQueryID()
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to generate random query ID: %w", err)
	}

	body, err := tlb.ToCell(timelock.ExecuteBatch{
		QueryID: qID,

		Calls:       calls,
		Predecessor: tlbe.NewUint256(predecessor.Big()),
		Salt:        tlbe.NewUint256(salt.Big()),
	})
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to encode ExecuteBatch body: %w", err)
	}

	return SendTx(ctx, TxOpts{
		Wallet:  e.wallet,
		DstAddr: dstAddr,
		Amount:  e.amount,
		Body:    body,
	})
}
