package ton

import (
	"context"
	"fmt"
	"math/big"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tlbe"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

var _ sdk.Configurer = &configurer{}

type configurer struct {
	wallet *wallet.Wallet

	// Transaction opts
	amount tlb.Coins

	skipSend bool
}

// NewConfigurer creates a new Configurer for TON chains.
//
// options:
//
//	WithDoNotSendInstructionsOnChain: when selected, the Configurer instance will not
//			send the TON instructions to the blockchain.
func NewConfigurer(w *wallet.Wallet, amount tlb.Coins, opts ...ConfigurerOption) (sdk.Configurer, error) {
	c := configurer{w, amount, false}

	for _, o := range opts {
		o(&c)
	}

	return c, nil
}

type ConfigurerOption func(*configurer)

// WithDoNotSendInstructionsOnChain sets the configurer to not sign and send the configuration transaction
// but rather make it return a prepared MCMS types.Transaction instead.
// If set, the Hash field in the result will be empty.
func WithDoNotSendInstructionsOnChain() ConfigurerOption {
	return func(c *configurer) {
		c.skipSend = true
	}
}

func (c configurer) SetConfig(ctx context.Context, mcmsAddr string, cfg *types.Config, clearRoot bool) (types.TransactionResult, error) {
	// Map to Ton Address type
	dstAddr, err := address.ParseAddr(mcmsAddr)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("invalid mcms address: %w", err)
	}

	groupQuorum, groupParents, signerAddresses, _signerGroups, err := evm.ExtractSetConfigInputs(cfg)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to extract set config inputs: %w", err)
	}

	signerKeys := make([]mcms.SignerAddress, len(signerAddresses))
	for i, addr := range signerAddresses {
		signerKeys[i] = mcms.SignerAddress{Val: tlbe.NewUint160(addr.Big())}
	}

	signerGroups := make([]mcms.SignerGroup, len(_signerGroups))
	for i, g := range _signerGroups {
		signerGroups[i] = mcms.SignerGroup{Val: g}
	}

	// Encode SetConfig message
	sz := uint(tvm.SizeUINT8)
	gqDict := cell.NewDict(sz)
	for i, g := range groupQuorum {
		err = gqDict.SetIntKey(big.NewInt(int64(i)), cell.BeginCell().MustStoreUInt(uint64(g), sz).EndCell())
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("unable to dict.set group quorum %d: %w", i, err)
		}
	}

	gpDict := cell.NewDict(sz)
	for i, g := range groupParents {
		err = gpDict.SetIntKey(big.NewInt(int64(i)), cell.BeginCell().MustStoreUInt(uint64(g), sz).EndCell())
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("unable to dict.set group parent %d: %w", i, err)
		}
	}

	qID, err := tvm.RandomQueryID()
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to generate random query ID: %w", err)
	}

	body, err := tlb.ToCell(mcms.SetConfig{
		QueryID: qID,

		SignerAddresses: signerKeys,
		SignerGroups:    signerGroups,
		GroupQuorums:    gqDict,
		GroupParents:    gpDict,

		ClearRoot: clearRoot,
	})
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to encode SetConfig body: %w", err)
	}

	return SendTx(ctx, TxOpts{
		Wallet:   c.wallet,
		DstAddr:  dstAddr,
		Amount:   c.amount,
		Body:     body,
		SkipSend: c.skipSend,
	})
}
