package ton

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

type TxOpts struct {
	Wallet  *wallet.Wallet
	DstAddr *address.Address
	Amount  tlb.Coins
	Body    *cell.Cell
}

func SendTx(ctx context.Context, opts TxOpts) (types.TransactionResult, error) {
	msg := &wallet.Message{
		Mode: wallet.PayGasSeparately | wallet.IgnoreErrors,
		InternalMessage: &tlb.InternalMessage{
			IHRDisabled: true,
			Bounce:      true,
			DstAddr:     opts.DstAddr,
			Amount:      opts.Amount,
			Body:        opts.Body,
		},
	}

	tx, _, err := opts.Wallet.SendWaitTransaction(ctx, msg)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to send transaction: %w", err)
	}

	return types.TransactionResult{
		Hash:        hex.EncodeToString(tx.Hash),
		ChainFamily: chainsel.FamilyTon,
		RawData:     tx, // *tlb.Transaction
	}, nil
}

func SendTxAfter(ctx context.Context, opts TxOpts, now uint64, validAfter uint64, buffer time.Duration, wait ...bool) (types.TransactionResult, error) {
	var z types.TransactionResult // zero value

	// If the operation is valid to be executed, send the transaction immediately
	if now >= validAfter {
		return SendTx(ctx, opts)
	}

	// Waits by default, unless specified otherwise
	if len(wait) > 0 && !wait[0] {
		return z, fmt.Errorf("pending operation not final: operation is not yet valid to be executed (valid after %d, now %d)", validAfter, now)
	}

	// add a buffer to wait to ensure validity
	waitDuration := time.Duration(validAfter-now)*time.Second + buffer //nolint:gosec // ok to convert here
	tick := time.NewTicker(waitDuration)

	logger := sdk.LoggerFrom(ctx)
	logger.Infof("waiting to send transaction", "validAfter", validAfter, "now", now, "waitDuration", waitDuration)
	for {
		select {
		case <-tick.C:
			nowApprox := now + uint64(waitDuration.Seconds())
			logger.Infof("sending delayed transaction", "validAfter", validAfter, "nowApprox", nowApprox)

			return SendTx(ctx, opts)
		case <-ctx.Done():
			return z, fmt.Errorf("context done while waiting to send transaction: %w", ctx.Err())
		}
	}
}
