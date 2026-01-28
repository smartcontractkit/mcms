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
		ChainFamily: cselectors.FamilyTon,
		RawData:     tx, // *tlb.Transaction
	}, nil
}
