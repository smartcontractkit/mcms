package ton

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand/v2"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	"github.com/ethereum/go-ethereum/common"
	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/timelock"
	commonton "github.com/smartcontractkit/chainlink-ton/pkg/ccip/bindings/common"
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
func NewTimelockExecutor(client *ton.APIClient, wallet *wallet.Wallet, amount tlb.Coins) sdk.TimelockExecutor {
	return &timelockExecutor{
		TimelockInspector: NewTimelockInspector(client),
		wallet:            wallet,
		amount:            amount,
	}
}

func (t *timelockExecutor) Execute(
	ctx context.Context, bop types.BatchOperation, timelockAddress string, predecessor common.Hash, salt common.Hash,
) (types.TransactionResult, error) {
	// Map to Ton Address type
	dstAddr, err := address.ParseAddr(timelockAddress)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("invalid timelock address: %w", err)
	}

	calls := make([]timelock.Call, len(bop.Transactions))
	for i, tx := range bop.Transactions {
		// Unmarshal the AdditionalFields from the operation
		var additionalFields AdditionalFields
		if err := json.Unmarshal(tx.AdditionalFields, &additionalFields); err != nil {
			return types.TransactionResult{}, err
		}

		// Map to Ton Address type
		to, err := address.ParseAddr(tx.To)
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("invalid target address: %w", err)
		}

		datac, err := cell.FromBOC(tx.Data)
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("invalid cell BOC data: %w", err)
		}

		calls[i] = timelock.Call{
			Target: *to,
			Data:   datac,
			Value:  additionalFields.Value,
		}
	}

	body, err := tlb.ToCell(timelock.ExecuteBatch{
		QueryID: rand.Uint64(),

		Calls:       commonton.SnakeData[timelock.Call](calls),
		Predecessor: predecessor.Big(),
		Salt:        salt.Big(),
	})
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to encode ExecuteBatch body: %w", err)
	}

	msg := &wallet.Message{
		Mode: 1,
		InternalMessage: &tlb.InternalMessage{
			IHRDisabled: true,
			Bounce:      true,
			DstAddr:     dstAddr,
			Amount:      t.amount,
			Body:        body,
		},
	}

	// TODO: do we wait for execution trace?
	tx, _, err := t.wallet.SendWaitTransaction(ctx, msg)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to set config: %w", err)
	}

	return types.TransactionResult{
		Hash:        hex.EncodeToString(tx.Hash),
		ChainFamily: chain_selectors.FamilyTon,
		RawData:     tx,
	}, nil
}
