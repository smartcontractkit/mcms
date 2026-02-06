package ton

import (
	"context"
	"fmt"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tlbe"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"

	"github.com/smartcontractkit/mcms/sdk"

	"github.com/smartcontractkit/mcms/types"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/wallet"
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

	groupQuorum, groupParents, signerAddresses, _signerGroups, err := sdk.ExtractSetConfigInputs(cfg)
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
	gqDict, err := tlbe.NewDictFromSlice[uint8](groupQuorum[:])
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to create group quorum dict: %w", err)
	}

	gpDict, err := tlbe.NewDictFromSlice[uint8](groupParents[:])
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to create group parents dict: %w", err)
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

	// check if we should plan this as an MCMS op instead of executing now
	if c.skipSend {
		var tx types.Transaction
		tx, err := NewTransaction(dstAddr, body.ToBuilder().ToSlice(), c.amount.Nano(), "MCMS", []string{"SetConfig"})
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("error encoding transaction: %w", err)
		}

		return types.TransactionResult{
			Hash:        "", // Returning no hash since the transaction hasn't been sent yet.
			ChainFamily: chainsel.FamilyTon,
			RawData:     tx, // will be of type types.Transaction
		}, nil
	}

	return SendTx(ctx, TxOpts{
		Wallet:  c.wallet,
		DstAddr: dstAddr,
		Amount:  c.amount,
		Body:    body,
	})
}
