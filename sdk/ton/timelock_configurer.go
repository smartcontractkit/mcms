package ton

import (
	"context"
	"fmt"
	"math"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings"
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

	skipSend bool
}

// NewTimelockConfigurer creates a new TimelockConfigurer for TON chains.
func NewTimelockConfigurer(w *wallet.Wallet, amount tlb.Coins, opts ...TimelockConfigurerOption) *TimelockConfigurer {
	c := &TimelockConfigurer{wallet: w, amount: amount}
	for _, opt := range opts {
		opt(c)
	}

	return c
}

type TimelockConfigurerOption func(*TimelockConfigurer)

func WithDoNotSendTimelockInstructionsOnChain() TimelockConfigurerOption {
	return func(c *TimelockConfigurer) {
		c.skipSend = true
	}
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

	msg := timelock.UpdateDelay{
		NewDelay: uint32(newDelay),
	}
	var body *cell.Cell
	if c.skipSend {
		body, err = tlb.ToCell(msg)
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("failed to encode UpdateDelay body: %w", err)
		}

		msg.QueryID = deterministicPreparedQueryID(dstAddr, "RBACTimelock:UpdateDelay", body)
	} else {
		msg.QueryID, err = tvm.RandomQueryID()
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("failed to generate random query ID: %w", err)
		}
	}

	body, err = tlb.ToCell(msg)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to encode UpdateDelay body: %w", err)
	}

	if c.skipSend {
		tx, err := NewTransaction(dstAddr, body.ToBuilder().ToSlice(), c.amount.Nano(), bindings.ShortTimelock, string(bindings.TypeTimelock)+" 0.0.0", []string{"UpdateDelay"})
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("error encoding transaction: %w", err)
		}

		return types.TransactionResult{
			Hash:        "",
			ChainFamily: chainsel.FamilyTon,
			RawData:     tx,
		}, nil
	}

	return SendTx(ctx, TxOpts{
		Wallet:  c.wallet,
		DstAddr: dstAddr,
		Amount:  c.amount,
		Body:    body,
	})
}
