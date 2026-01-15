package ton

import (
	"context"
	"encoding/hex"
	"fmt"

	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/mcms/types"
)

type TxOpts struct {
	Wallet   *wallet.Wallet
	DstAddr  *address.Address
	Amount   tlb.Coins
	Body     *cell.Cell
	SkipSend bool
}

func SendTx(ctx context.Context, opts TxOpts) (types.TransactionResult, error) {
	if opts.SkipSend {
		var tx types.Transaction
		tx, err := NewTransaction(opts.DstAddr, opts.Body.ToBuilder().ToSlice(), opts.Amount.Nano(), "", nil)
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("error encoding transaction: %w", err)
		}

		return types.TransactionResult{
			Hash:        "", // Returning no hash since the transaction hasn't been sent yet.
			ChainFamily: cselectors.FamilyTon,
			RawData:     tx, // will be of type types.Transaction
		}, nil
	}

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
		ChainFamily: cselectors.FamilyTon,
		RawData:     tx, // *tlb.Transaction
	}, nil
}
