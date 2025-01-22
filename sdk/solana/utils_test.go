package solana

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/solana/mocks"
	"github.com/smartcontractkit/mcms/types"
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

func mockGetBlockTime(
	t *testing.T, client *mocks.JSONRPCClient, block uint64,
	blockTime *solana.UnixTimeSeconds, mockBlockHeightError error,
	mockBlockTimeError error,
) {
	t.Helper()

	// mock getBlockHeight
	client.EXPECT().CallForInto(anyContext, mock.Anything, "getBlockHeight",
		[]any{rpc.M{"commitment": rpc.CommitmentConfirmed}},
	).RunAndReturn(func(_ context.Context, output any, _ string, _ []any) error {
		result, ok := output.(*uint64)
		require.True(t, ok)

		*result = block // set block height as 1

		return mockBlockHeightError
	}).Once()

	if mockBlockHeightError != nil {
		return
	}

	client.EXPECT().CallForInto(
		anyContext, mock.Anything, "getBlockTime", []any{block},
	).RunAndReturn(func(_ context.Context, output any, _ string, _ []any) error {
		result, ok := output.(**solana.UnixTimeSeconds)
		require.True(t, ok)

		*result = blockTime

		return mockBlockTimeError
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

func mockSolanaSimulateTransaction(
	t *testing.T, client *mocks.JSONRPCClient, lastBlockHeight uint64, mockBlockHashRPCError error,
	mockSimulateTransactionError error,
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

		return mockBlockHashRPCError
	}).Once()
	if mockBlockHashRPCError != nil {
		return
	}

	client.EXPECT().CallForInto(
		anyContext, mock.Anything, "simulateTransaction", mock.Anything,
	).RunAndReturn(func(_ context.Context, output any, _ string, _ []any) error {
		result, ok := output.(**rpc.SimulateTransactionResponse)
		require.True(t, ok)

		*result = &rpc.SimulateTransactionResponse{
			Value: &rpc.SimulateTransactionResult{
				Err:  mockSimulateTransactionError,
				Logs: []string{"Transaction simulation successful"},
			},
		}

		return nil
	})
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

func generateSignatures(t *testing.T, numSignatures int) []types.Signature {
	t.Helper()

	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	signatures := make([]types.Signature, numSignatures)
	for i := range signatures {
		payload := []byte(fmt.Sprintf("\x19Ethereum Signed Message:\n320x%d", i))
		hash := crypto.Keccak256Hash(payload)

		sigBytes, err := crypto.Sign(hash[:], privateKey)
		require.NoError(t, err)

		signatures[i], err = types.NewSignatureFromBytes(sigBytes)
		require.NoError(t, err)
	}

	slices.SortFunc(signatures, func(a, b types.Signature) int { return bytes.Compare(a.ToBytes(), b.ToBytes()) })

	return signatures
}

func ptrTo[T any](value T) *T { return &value }
