package solana

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/solana/mocks"
)

var anyContext = mock.MatchedBy(func(_ context.Context) bool { return true })

func mockGetAccountInfo(
	t *testing.T, mockJSONRPCClient *mocks.JSONRPCClient, account solana.PublicKey, accountInfo any,
	mockError error,
) {
	t.Helper()

	mockJSONRPCClient.EXPECT().CallForInto(anyContext, mock.Anything, "getAccountInfo", []any{
		account, rpc.M{"commitment": rpc.CommitmentConfirmed, "encoding": solana.EncodingBase64},
	},
	).RunAndReturn(func(_ context.Context, output any, _ string, _ []any) error {
		result, ok := output.(**rpc.GetAccountInfoResult)
		require.True(t, ok)

		marshaledConfig, err := bin.MarshalBorsh(accountInfo)

		require.NoError(t, err)

		*result = &rpc.GetAccountInfoResult{Value: &rpc.Account{Data: rpc.DataBytesOrJSONFromBytes(marshaledConfig)}}

		return mockError
	}).Once()
}

func mockSolanaTransaction(
	t *testing.T, client *mocks.JSONRPCClient, lastBlockHeight uint64, slot uint64, signature string, mockError error,
) {
	t.Helper()

	client.EXPECT().CallForInto(
		anyContext, mock.Anything, "getLatestBlockhash", []any{rpc.M{"commitment": rpc.CommitmentFinalized}},
	).RunAndReturn(func(_ context.Context, output any, _ string, _ []any) error {
		result, ok := output.(**rpc.GetLatestBlockhashResult)
		require.True(t, ok)

		*result = &rpc.GetLatestBlockhashResult{Value: &rpc.LatestBlockhashResult{
			Blockhash:            solana.MustHashFromBase58(randomPublicKey(t).String()),
			LastValidBlockHeight: lastBlockHeight,
		}}

		return mockError
	}).Once()
	if mockError != nil {
		return
	}

	client.EXPECT().CallForInto(
		anyContext, mock.Anything, "sendTransaction", sendTransactionParams(t),
	).RunAndReturn(func(_ context.Context, output any, _ string, _ []any) error {
		result, ok := output.(*solana.Signature)
		require.True(t, ok)
		*result = solana.MustSignatureFromBase58("3Kp5n9Ye69MNAeUEiw77QCMR2c5csEUxr3opSUzFJM7dFRf5jUYNufbb4B1caQehD1wGrP3yGCo5N7V9W96CQzAH")

		return nil
	}).Once()

	client.EXPECT().CallForInto(
		anyContext, mock.Anything, "getSignatureStatuses", mock.Anything,
	).RunAndReturn(func(_ context.Context, output any, _ string, _ []any) error {
		result, ok := output.(**rpc.GetSignatureStatusesResult)
		require.True(t, ok)
		*result = &rpc.GetSignatureStatusesResult{
			Value: []*rpc.SignatureStatusesResult{{
				Slot:               slot,
				Confirmations:      ptrTo(uint64(2)),
				ConfirmationStatus: rpc.ConfirmationStatusConfirmed,
			}},
		}

		return nil
	}).Once()

	client.EXPECT().CallForInto(
		anyContext, mock.Anything, "getTransaction", mock.Anything,
	).RunAndReturn(func(_ context.Context, output any, _ string, _ []any) error {
		result, ok := output.(**rpc.GetTransactionResult)
		require.True(t, ok)

		var transactionEnvelope rpc.TransactionResultEnvelope
		err := transactionEnvelope.UnmarshalJSON([]byte(`{
			"signatures": ["` + signature + `"],
			"message": {}
		}`))
		require.NoError(t, err)

		*result = &rpc.GetTransactionResult{
			Version:     1,
			Slot:        slot,
			BlockTime:   ptrTo(solana.UnixTimeSeconds(time.Now().Unix())),
			Transaction: &transactionEnvelope,
			Meta:        &rpc.TransactionMeta{},
		}

		return nil
	}).Once()
}

var sendTransactionParams = func(t *testing.T) any {
	t.Helper()

	return mock.MatchedBy(func(args []any) bool {
		if len(args) == 1 {
			_, isMap := args[0].(rpc.M)

			return isMap
		}
		if len(args) == 2 {
			_, isString := args[0].(string)
			_, isMap := args[1].(rpc.M)

			return isString && isMap
		}

		return false
	})
}

func randomPublicKey(t *testing.T) solana.PublicKey {
	t.Helper()
	privKey, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	return privKey.PublicKey()
}

func generateSigners(t *testing.T, numSigners int) []common.Address {
	t.Helper()

	signers := make([]common.Address, numSigners)
	for i := range signers {
		signers[i] = common.HexToAddress(fmt.Sprintf("0x%x", i))
	}
	slices.SortFunc(signers, func(a, b common.Address) int { return a.Cmp(b) })

	return signers
}

func ptrTo[T any](value T) *T { return &value }
