package ton

import (
	"context"
	"fmt"
	"math"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/wallet"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/timelock"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.TimelockConfigurer = (*TimelockConfigurer)(nil)

// TimelockConfigurer configures timelock parameters on TON chains.
type TimelockConfigurer struct {
	wallet *wallet.Wallet
	// amount is attached to the internal message.
	amount tlb.Coins
}

// NewTimelockConfigurer creates a new TimelockConfigurer for TON chains.
func NewTimelockConfigurer(w *wallet.Wallet, amount tlb.Coins) *TimelockConfigurer {
	return &TimelockConfigurer{wallet: w, amount: amount}
}

// UpdateDelay sends the RBACTimelock UpdateDelay message to the given timelock
// address. TON's timelock accepts a uint32 delay.
func (c *TimelockConfigurer) UpdateDelay(
	ctx context.Context, timelockAddress string, newDelay uint64,
) (types.TransactionResult, error) {
	if newDelay > math.MaxUint32 {
		return types.TransactionResult{}, fmt.Errorf("newDelay %d exceeds uint32 range supported by TON timelock", newDelay)
	}

	dstAddr, err := address.ParseAddr(timelockAddress)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("invalid timelock address: %w", err)
	}

	qID, err := tvm.RandomQueryID()
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to generate random query ID: %w", err)
	}

	body, err := tlb.ToCell(timelock.UpdateDelay{
		QueryID:  qID,
		NewDelay: uint32(newDelay),
	})
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to encode UpdateDelay body: %w", err)
	}

	return SendTx(ctx, TxOpts{
		Wallet:  c.wallet,
		DstAddr: dstAddr,
		Amount:  c.amount,
		Body:    body,
	})
}
