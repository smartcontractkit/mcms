package ton

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	"github.com/ethereum/go-ethereum/common"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/timelock"
)

var _ sdk.TimelockExecutor = (*timelockExecutor)(nil)

// sdk.TimelockExecutor implementation for TON chains for accessing the RBACTimelock contract
type timelockExecutor struct {
	sdk.TimelockInspector
	wallet *wallet.Wallet

	// Transaction opts
	amount tlb.Coins
}

// NewTimelockExecutor creates a new TimelockExecutor
func NewTimelockExecutor(client ton.APIClientWrapped, w *wallet.Wallet, amount tlb.Coins) (sdk.TimelockExecutor, error) {
	if IsNil(client) {
		return nil, errors.New("failed to create sdk.Executor - client (ton.APIClientWrapped) is nil")
	}

	return &timelockExecutor{
		TimelockInspector: NewTimelockInspector(client),
		wallet:            w,
		amount:            amount,
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

	qID, err := RandomQueryID()
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to generate random query ID: %w", err)
	}

	body, err := tlb.ToCell(timelock.ExecuteBatch{
		QueryID: qID,

		Calls:       calls,
		Predecessor: predecessor.Big(),
		Salt:        salt.Big(),
	})
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to encode ExecuteBatch body: %w", err)
	}

	msg := &wallet.Message{
		Mode: wallet.PayGasSeparately | wallet.IgnoreErrors,
		InternalMessage: &tlb.InternalMessage{
			IHRDisabled: true,
			Bounce:      true,
			DstAddr:     dstAddr,
			Amount:      e.amount,
			Body:        body,
		},
	}

	tx, _, err := e.wallet.SendWaitTransaction(ctx, msg)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to execute batch: %w", err)
	}

	return types.TransactionResult{
		Hash:        hex.EncodeToString(tx.Hash),
		ChainFamily: chain_selectors.FamilyTon,
		RawData:     tx,
	}, nil
}
