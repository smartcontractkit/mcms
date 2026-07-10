package solana

import (
	"context"

	"github.com/gagliardetto/solana-go"

	"github.com/smartcontractkit/mcms/types"
)

// executePayersKey is the unexported context key for the expected execute payers.
type executePayersKey struct{}

// ExecutePayers maps a chain selector to the public key that will pay for (and
// therefore sign) the MCM execute transaction on that chain.
type ExecutePayers map[types.ChainSelector]solana.PublicKey

// WithExecutePayers returns a copy of ctx carrying the expected execute payer
// per chain. This is consumed by the Solana TimelockConverter when converting
// bypass operations: if the payer also appears in an operation's remaining
// accounts, it must be hashed as a signer in the off-chain Merkle root so that
// on-chain proof verification (which always sees the fee payer as a signer)
// succeeds. See ConvertBatchToChainOperations.
func WithExecutePayers(ctx context.Context, payers ExecutePayers) context.Context {
	return context.WithValue(ctx, executePayersKey{}, payers)
}

// ExecutePayersFrom returns the ExecutePayers stored in ctx, if any.
func ExecutePayersFrom(ctx context.Context) (ExecutePayers, bool) {
	payers, ok := ctx.Value(executePayersKey{}).(ExecutePayers)
	return payers, ok
}
