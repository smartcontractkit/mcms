package ton

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"math/rand/v2"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
	"github.com/smartcontractkit/chainlink-ton/pkg/ccip/bindings/common"

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
func NewConfigurer(wallet *wallet.Wallet, amount tlb.Coins, opts ...configurerOption) (sdk.Configurer, error) {
	c := configurer{wallet, amount, false}

	for _, o := range opts {
		o(&c)
	}

	return c, nil
}

type configurerOption func(*configurer)

// WithDoNotSendInstructionsOnChain sets the configurer to not sign and send the configuration transaction
// but rather make it return a prepared MCMS types.Transaction instead.
// If set, the Hash field in the result will be empty.
func WithDoNotSendInstructionsOnChain() configurerOption {
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

	signerKeys := make([]mcms.SignerKey, len(signerAddresses))
	for i, addr := range signerAddresses {
		signerKeys[i] = mcms.SignerKey{Value: addr.Big()}
	}

	signerGroups := make([]mcms.SignerGroup, len(_signerGroups))
	for i, g := range _signerGroups {
		signerGroups[i] = mcms.SignerGroup{Value: g}
	}

	// Encode SetConfig message
	sz := uint(8)
	gqDict := cell.NewDict(sz)
	for i, g := range groupQuorum {
		gqDict.SetIntKey(big.NewInt(int64(i)), cell.BeginCell().MustStoreUInt(uint64(g), sz).EndCell())
	}

	gpDict := cell.NewDict(sz)
	for i, g := range groupParents {
		gpDict.SetIntKey(big.NewInt(int64(i)), cell.BeginCell().MustStoreUInt(uint64(g), sz).EndCell())
	}

	body, err := tlb.ToCell(mcms.SetConfig{
		QueryID: rand.Uint64(),

		SignerKeys:   common.SnakeData[mcms.SignerKey](signerKeys),
		SignerGroups: common.SnakeData[mcms.SignerGroup](signerGroups),
		GroupQuorums: gqDict,
		GroupParents: gpDict,

		ClearRoot: clearRoot,
	})
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to encode SetConfig body: %w", err)
	}

	if c.skipSend {
		tx, err := NewTransaction(dstAddr, body.ToBuilder().ToSlice(), c.amount.Nano(), "", nil)
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("error encoding mcms setConfig transaction: %w", err)
		}

		return types.TransactionResult{
			Hash:        "", // Returning no hash since the transaction hasn't been sent yet.
			ChainFamily: chain_selectors.FamilyTon,
			RawData:     tx, // will be of type types.Transaction
		}, nil
	}

	msg := &wallet.Message{
		Mode: wallet.PayGasSeparately,
		InternalMessage: &tlb.InternalMessage{
			IHRDisabled: true,
			Bounce:      true,
			DstAddr:     dstAddr,
			Amount:      c.amount,
			Body:        body,
		},
	}

	// TODO: do we wait for execution trace?
	tx, _, err := c.wallet.SendWaitTransaction(ctx, msg)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to set config: %w", err)
	}

	return types.TransactionResult{
		Hash:        hex.EncodeToString(tx.Hash),
		ChainFamily: cselectors.FamilyTon,
		RawData:     tx, // *tlb.Transaction
	}, nil
}
