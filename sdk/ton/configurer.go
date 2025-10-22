package ton

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"math/rand/v2"

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
}

func NewConfigurer(wallet *wallet.Wallet, amount tlb.Coins) (configurer, error) {
	return configurer{wallet, amount}, nil
}

func (c configurer) SetConfig(ctx context.Context, mcmsAddr string, cfg *types.Config, clearRoot bool) (types.TransactionResult, error) {
	// Map to Ton Address type
	dstAddr, err := address.ParseAddr(mcmsAddr)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("invalid mcms address: %w", err)
	}

	groupQuorum, groupParents, signerAddresses, signerGroups, err := evm.ExtractSetConfigInputs(cfg)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to extract set config inputs: %w", err)
	}

	signerKeys := make([]mcms.SignerKey, len(signerAddresses))
	for i, addr := range signerAddresses {
		signerKeys[i] = mcms.SignerKey{
			Value: addr.Big(),
		}
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
		SignerGroups: common.SnakeData[uint8](signerGroups),
		GroupQuorums: gqDict,
		GroupParents: gpDict,

		ClearRoot: clearRoot,
	})
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to encode SetConfig body: %w", err)
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
		RawData:     tx,
	}, nil
}
