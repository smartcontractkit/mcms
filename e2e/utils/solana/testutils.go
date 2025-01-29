package solana

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/stretchr/testify/require"
)

const timeout = 30 * time.Second
const pollInterval = 50 * time.Millisecond

// FundAccounts funds the given accounts with 100 SOL each.
// It waits for the transactions to be confirmed.
func FundAccounts(
	t *testing.T,
	ctx context.Context, accounts []solana.PublicKey, solAmount uint64, solanaGoClient *rpc.Client) {
	t.Helper()

	var sigs = make([]solana.Signature, 0, len(accounts))
	for _, v := range accounts {
		sig, err := solanaGoClient.RequestAirdrop(ctx, v, solAmount*solana.LAMPORTS_PER_SOL, rpc.CommitmentConfirmed)
		require.NoError(t, err)
		sigs = append(sigs, sig)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	remaining := len(sigs)
	for remaining > 0 {
		select {
		case <-timeoutCtx.Done():
			require.NoError(t, fmt.Errorf("unable to find transaction within timeout"))
		case <-ticker.C:
			statusRes, sigErr := solanaGoClient.GetSignatureStatuses(ctx, true, sigs...)
			require.NoError(t, sigErr)
			require.NotNil(t, statusRes)
			require.NotNil(t, statusRes.Value)

			unconfirmedTxCount := 0
			for _, res := range statusRes.Value {
				if res == nil || res.ConfirmationStatus == rpc.ConfirmationStatusProcessed {
					unconfirmedTxCount++
				}
			}
			remaining = unconfirmedTxCount
		}
	}
}
