package stellar

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/smartcontractkit/chainlink-stellar/bindings"
	"github.com/stellar/go-stellar-sdk/keypair"
	"github.com/stellar/go-stellar-sdk/strkey"
	"github.com/stellar/go-stellar-sdk/txnbuild"
	"github.com/stellar/go-stellar-sdk/xdr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	protocolrpc "github.com/stellar/go-stellar-sdk/protocols/rpc"

	"github.com/smartcontractkit/mcms/sdk/stellar/mocks"
)

const testNetworkPassphrase = "Test SDF Network ; September 2015"

func newTestSigner(t *testing.T) *mocks.Signer {
	t.Helper()

	kp, err := keypair.Random()
	require.NoError(t, err)

	signer := mocks.NewSigner(t)

	signer.On("Address").
		Return(kp.Address()).
		Maybe()

	signer.On("Sign", mock.Anything).
		Return(func(message []byte) ([]byte, error) {
			return kp.Sign(message)
		}).
		Maybe()

	signer.On("SignDecorated", mock.Anything).
		Return(func(message []byte) (xdr.DecoratedSignature, error) {
			return kp.SignDecorated(message)
		}).
		Maybe()

	signer.On("KeypairFull").
		Return(kp).
		Maybe()

	return signer
}

func newTestInvoker(t *testing.T, rpc *mocks.RpcClient) (*rpcInvoker, *mocks.Signer) {
	t.Helper()

	signer := newTestSigner(t)
	invoker := newTestInvokerWithSigner(rpc, signer, testNetworkPassphrase)

	return invoker, signer
}

// newTestInvokerBare returns an invoker with a no-op mock RPC client and a random
// test signer, for tests that never call RPC methods.
func newTestInvokerBare(t *testing.T) *rpcInvoker {
	t.Helper()

	return newTestInvokerWithSigner(
		mocks.NewRpcClient(t),
		newTestSigner(t),
		testNetworkPassphrase,
	)
}

func newTestInvokerWithSigner(
	rpc *mocks.RpcClient,
	signer bindings.Signer,
	passphrase string,
) *rpcInvoker {
	return &rpcInvoker{
		client:            rpc,
		signer:            signer,
		networkPassphrase: passphrase,
	}
}

// submitTestFixtures holds the common objects needed by every submit-path test.
type submitTestFixtures struct {
	rpc     *mocks.RpcClient
	invoker *rpcInvoker
	signer  *mocks.Signer
	op      *txnbuild.InvokeHostFunction
}

// setupSubmitTest creates an invoker with a mock RPC client, builds a valid
// invoke operation ("test_fn"), and configures the mock ledger-entry response.
// Each test adds its own SimulateTransaction / SendTransaction / GetTransaction
// expectations so that error-path tests can override the simulation result.
func setupSubmitTest(t *testing.T) submitTestFixtures {
	t.Helper()
	rpc := mocks.NewRpcClient(t)
	invoker, signer := newTestInvoker(t, rpc)

	contractID := validContractStrkey(t)
	op, err := invoker.invokeOperation(contractID, "test_fn", nil)
	require.NoError(t, err)

	accountB64 := ledgerEntryB64(t, signer, 10)
	rpc.On("GetLedgerEntries", mock.Anything, mock.Anything).
		Return(protocolrpc.GetLedgerEntriesResponse{
			Entries: []protocolrpc.LedgerEntryResult{{DataXDR: accountB64}},
		}, nil).
		Maybe()

	return submitTestFixtures{rpc: rpc, invoker: invoker, signer: signer, op: op}
}

// stubSimulate adds a successful simulation expectation to the mock RPC client.
func (f *submitTestFixtures) stubSimulate(t *testing.T) {
	t.Helper()
	simResult := buildSimResponse(t, f.op)
	f.rpc.On("SimulateTransaction", mock.Anything, mock.Anything).Return(simResult, nil)
}

func marshalB64(t *testing.T, v any) string {
	t.Helper()
	b64, err := xdr.MarshalBase64(v)
	require.NoError(t, err)

	return b64
}

func ledgerEntryB64(t *testing.T, signer bindings.Signer, seqNum uint32) string {
	t.Helper()
	acctID := xdr.MustAddress(signer.Address())
	entry := xdr.LedgerEntryData{
		Type: xdr.LedgerEntryTypeAccount,
		Account: &xdr.AccountEntry{
			AccountId: acctID,
			SeqNum:    xdr.SequenceNumber(seqNum),
			Balance:   100000000,
		},
	}

	return marshalB64(t, &entry)
}

func accountKeyB64(t *testing.T, signer bindings.Signer) string {
	t.Helper()
	acctID := xdr.MustAddress(signer.Address())
	key := xdr.LedgerKey{
		Type:    xdr.LedgerEntryTypeAccount,
		Account: &xdr.LedgerKeyAccount{AccountId: acctID},
	}
	b64, err := key.MarshalBinaryBase64()
	require.NoError(t, err)

	return b64
}

// buildSimResponse builds a valid SimulateTransactionResponse with auth entries
// and Soroban transaction data that the submit path can assemble.
func buildSimResponse(t *testing.T, op *txnbuild.InvokeHostFunction) protocolrpc.SimulateTransactionResponse {
	t.Helper()

	authEntry := xdr.SorobanAuthorizationEntry{
		Credentials: xdr.SorobanCredentials{
			Type: xdr.SorobanCredentialsTypeSorobanCredentialsSourceAccount,
		},
		RootInvocation: xdr.SorobanAuthorizedInvocation{
			Function: xdr.SorobanAuthorizedFunction{
				Type: xdr.SorobanAuthorizedFunctionTypeSorobanAuthorizedFunctionTypeContractFn,
				ContractFn: &xdr.InvokeContractArgs{
					ContractAddress: op.HostFunction.InvokeContract.ContractAddress,
					FunctionName:    op.HostFunction.InvokeContract.FunctionName,
					Args:            op.HostFunction.InvokeContract.Args,
				},
			},
			SubInvocations: nil,
		},
	}
	authXDR := []string{marshalB64(t, &authEntry)}

	sorobanData := xdr.SorobanTransactionData{
		Resources: xdr.SorobanResources{
			Footprint: xdr.LedgerFootprint{ReadOnly: nil, ReadWrite: nil},
		},
		ResourceFee: 100000,
	}

	return protocolrpc.SimulateTransactionResponse{
		MinResourceFee:     100000,
		TransactionDataXDR: marshalB64(t, &sorobanData),
		Results: []protocolrpc.SimulateHostFunctionResult{{
			AuthXDR:        &authXDR,
			ReturnValueXDR: new("AAAAAQ=="), // ScvVoid
		}},
	}
}

// buildSuccessMeta returns a base64-encoded TransactionMeta V3 for a SUCCESS result.
// pass nil for returnValue to use ScvVoid.
func buildSuccessMeta(t *testing.T, returnValue *xdr.ScVal) string {
	t.Helper()
	rv := xdr.ScVal{Type: xdr.ScValTypeScvVoid}
	if returnValue != nil {
		rv = *returnValue
	}
	sorobanMeta := xdr.SorobanTransactionMeta{ReturnValue: rv}
	meta := xdr.TransactionMeta{
		V:          3,
		Operations: &[]xdr.OperationMeta{},
		V3:         &xdr.TransactionMetaV3{SorobanMeta: &sorobanMeta},
	}

	return marshalB64(t, &meta)
}

func TestNewInvoker_RejectsNilClient(t *testing.T) {
	t.Parallel()
	_, err := NewInvokerWithNetworkPassphrase(nil, newTestSigner(t), testNetworkPassphrase)
	require.ErrorContains(t, err, "RPC client is nil")
}

func TestNewInvoker_RejectsNilSigner(t *testing.T) {
	t.Parallel()
	invoker := newTestInvokerWithSigner(mocks.NewRpcClient(t), nil, testNetworkPassphrase)
	require.Nil(t, invoker.signer)
}

func TestNewInvoker_RejectsEmptyPassphrase(t *testing.T) {
	t.Parallel()
	invoker := newTestInvokerWithSigner(mocks.NewRpcClient(t), newTestSigner(t), "")
	require.NotNil(t, invoker)
	require.Empty(t, invoker.networkPassphrase)
}

func TestSourceAccount_EmptyLedger(t *testing.T) {
	t.Parallel()
	rpc := mocks.NewRpcClient(t)
	invoker, signer := newTestInvoker(t, rpc)

	rpc.On("GetLedgerEntries", mock.Anything,
		mock.MatchedBy(func(req protocolrpc.GetLedgerEntriesRequest) bool {
			return len(req.Keys) == 1 && req.Keys[0] == accountKeyB64(t, signer)
		})).
		Return(protocolrpc.GetLedgerEntriesResponse{Entries: nil}, nil)

	acct, err := invoker.sourceAccount(t.Context())
	require.NoError(t, err)
	require.Equal(t, signer.Address(), acct.AccountID)
	require.Equal(t, int64(0), acct.Sequence)
}

func TestSourceAccount_WithExistingAccount(t *testing.T) {
	t.Parallel()
	rpc := mocks.NewRpcClient(t)
	invoker, signer := newTestInvoker(t, rpc)

	rpc.On("GetLedgerEntries", mock.Anything,
		mock.MatchedBy(func(req protocolrpc.GetLedgerEntriesRequest) bool {
			return len(req.Keys) == 1 && req.Keys[0] == accountKeyB64(t, signer)
		})).
		Return(protocolrpc.GetLedgerEntriesResponse{
			Entries: []protocolrpc.LedgerEntryResult{{
				DataXDR: ledgerEntryB64(t, signer, 42),
			}},
		}, nil)

	acct, err := invoker.sourceAccount(t.Context())
	require.NoError(t, err)
	require.Equal(t, signer.Address(), acct.AccountID)
	require.Equal(t, int64(42), acct.Sequence)
}

func TestSourceAccount_RPCError(t *testing.T) {
	t.Parallel()
	rpc := mocks.NewRpcClient(t)
	invoker, signer := newTestInvoker(t, rpc)

	rpc.On("GetLedgerEntries", mock.Anything,
		mock.MatchedBy(func(req protocolrpc.GetLedgerEntriesRequest) bool {
			return len(req.Keys) == 1 && req.Keys[0] == accountKeyB64(t, signer)
		})).
		Return(protocolrpc.GetLedgerEntriesResponse{}, fmt.Errorf("rpc down"))

	_, err := invoker.sourceAccount(t.Context())
	require.ErrorContains(t, err, "rpc down")
}

func TestInvokeOperation_BuildsCorrectInvoke(t *testing.T) {
	t.Parallel()
	contractID := validContractStrkey(t)
	signer := newTestSigner(t)
	invoker := newTestInvokerWithSigner(mocks.NewRpcClient(t), signer, testNetworkPassphrase)

	op, err := invoker.invokeOperation(contractID, "test_fn", nil)
	require.NoError(t, err)

	require.Equal(t, signer.Address(), op.SourceAccount)
	require.Equal(t, xdr.HostFunctionTypeHostFunctionTypeInvokeContract, op.HostFunction.Type)
	require.Equal(t, xdr.ScSymbol("test_fn"), op.HostFunction.InvokeContract.FunctionName)
	require.Empty(t, op.HostFunction.InvokeContract.Args)
}

func TestInvokeOperation_BadContractID(t *testing.T) {
	t.Parallel()
	invoker := newTestInvokerBare(t)

	_, err := invoker.invokeOperation("not-a-contract", "fn", nil)
	require.ErrorContains(t, err, "decode Stellar contract ID")
}

func TestNewTransaction_ProducesValidTransaction(t *testing.T) {
	t.Parallel()
	invoker := newTestInvokerBare(t)

	contractID := validContractStrkey(t)
	op, err := invoker.invokeOperation(contractID, "fn", nil)
	require.NoError(t, err)

	tx, err := invoker.newTransaction(new(txnbuild.NewSimpleAccount(invoker.signer.Address(), 1)), op, txnbuild.MinBaseFee)
	require.NoError(t, err)

	require.NotNil(t, tx)
	require.Equal(t, invoker.signer.Address(), tx.SourceAccount().AccountID)

	xdrStr, err := tx.Base64()
	require.NoError(t, err)
	require.NotEmpty(t, xdrStr)
}

func TestReturnValue_VoidReturn(t *testing.T) {
	t.Parallel()
	meta := &xdr.TransactionMeta{
		V:          3,
		Operations: &[]xdr.OperationMeta{},
		V3:         &xdr.TransactionMetaV3{SorobanMeta: nil},
	}
	rv, err := returnValue(meta)
	require.Nil(t, rv)
	require.ErrorIs(t, err, errStellarVoidReturn)
}

func TestReturnValue_NilMeta(t *testing.T) {
	t.Parallel()
	rv, err := returnValue(nil)
	require.Nil(t, rv)
	require.ErrorIs(t, err, errStellarVoidReturn)
}

func TestReturnValue_V3WithValue(t *testing.T) {
	t.Parallel()
	expected := xdr.ScVal{Type: xdr.ScValTypeScvU32, U32: new(xdr.Uint32)}
	*expected.U32 = 999
	sorobanMeta := xdr.SorobanTransactionMeta{ReturnValue: expected}
	meta := &xdr.TransactionMeta{
		V:          3,
		Operations: &[]xdr.OperationMeta{},
		V3:         &xdr.TransactionMetaV3{SorobanMeta: &sorobanMeta},
	}
	rv, err := returnValue(meta)
	require.NoError(t, err)
	require.NotNil(t, rv)
	require.Equal(t, xdr.ScValTypeScvU32, rv.Type)
	require.EqualValues(t, 999, *rv.U32)
}

func TestReturnValue_V4WithValue(t *testing.T) {
	t.Parallel()
	retVal := xdr.ScVal{Type: xdr.ScValTypeScvBool, B: new(bool)}
	*retVal.B = true
	v4SorobanMeta := xdr.SorobanTransactionMetaV2{ReturnValue: &retVal}
	meta := &xdr.TransactionMeta{
		V:  4,
		V4: &xdr.TransactionMetaV4{SorobanMeta: &v4SorobanMeta},
	}
	rv, err := returnValue(meta)
	require.NoError(t, err)
	require.NotNil(t, rv)
	require.Equal(t, xdr.ScValTypeScvBool, rv.Type)
	require.True(t, *rv.B)
}

func TestReturnValue_UnsupportedVersion(t *testing.T) {
	t.Parallel()
	meta := &xdr.TransactionMeta{V: 99}
	_, err := returnValue(meta)
	require.ErrorContains(t, err, "unsupported Stellar transaction meta version")
}

func TestSimulateContract_ReturnsValue(t *testing.T) {
	t.Parallel()
	rpc := mocks.NewRpcClient(t)
	invoker, signer := newTestInvoker(t, rpc)

	contractID := validContractStrkey(t)
	keyB64 := accountKeyB64(t, signer)
	accountB64 := ledgerEntryB64(t, signer, 5)

	rpc.On("GetLedgerEntries", mock.Anything,
		mock.MatchedBy(func(req protocolrpc.GetLedgerEntriesRequest) bool {
			return len(req.Keys) == 1 && req.Keys[0] == keyB64
		})).
		Return(protocolrpc.GetLedgerEntriesResponse{
			Entries: []protocolrpc.LedgerEntryResult{{DataXDR: accountB64}},
		}, nil)

	u32Val := xdr.ScVal{Type: xdr.ScValTypeScvU32, U32: new(xdr.Uint32)}
	*u32Val.U32 = 1
	rpc.On("SimulateTransaction", mock.Anything, mock.Anything).
		Return(protocolrpc.SimulateTransactionResponse{
			Results: []protocolrpc.SimulateHostFunctionResult{{
				ReturnValueXDR: new(marshalB64(t, &u32Val)),
			}},
		}, nil)

	rv, err := invoker.SimulateContract(t.Context(), contractID, "read_count", nil)
	require.NoError(t, err)
	require.NotNil(t, rv)
	require.Equal(t, xdr.ScValTypeScvU32, rv.Type)
	require.EqualValues(t, 1, *rv.U32)
}

func TestSimulateContract_SimulationError(t *testing.T) {
	t.Parallel()
	rpc := mocks.NewRpcClient(t)
	invoker, signer := newTestInvoker(t, rpc)

	accountB64 := ledgerEntryB64(t, signer, 5)

	rpc.On("GetLedgerEntries", mock.Anything, mock.Anything).
		Return(protocolrpc.GetLedgerEntriesResponse{
			Entries: []protocolrpc.LedgerEntryResult{{DataXDR: accountB64}},
		}, nil)

	rpc.On("SimulateTransaction", mock.Anything, mock.Anything).
		Return(protocolrpc.SimulateTransactionResponse{Error: "HostError"}, nil)

	contractID := validContractStrkey(t)
	_, err := invoker.SimulateContract(t.Context(), contractID, "bad_fn", nil)
	require.ErrorContains(t, err, "HostError")
}

func TestSimulateContract_NoReturnValue(t *testing.T) {
	t.Parallel()
	rpc := mocks.NewRpcClient(t)
	invoker, signer := newTestInvoker(t, rpc)

	accountB64 := ledgerEntryB64(t, signer, 5)

	rpc.On("GetLedgerEntries", mock.Anything, mock.Anything).
		Return(protocolrpc.GetLedgerEntriesResponse{
			Entries: []protocolrpc.LedgerEntryResult{{DataXDR: accountB64}},
		}, nil)

	rpc.On("SimulateTransaction", mock.Anything, mock.Anything).
		Return(protocolrpc.SimulateTransactionResponse{
			Results: []protocolrpc.SimulateHostFunctionResult{{}},
		}, nil)

	contractID := validContractStrkey(t)
	_, err := invoker.SimulateContract(t.Context(), contractID, "fn", nil)
	require.ErrorIs(t, err, errStellarVoidReturn)
}

func TestSubmit_Success(t *testing.T) {
	t.Parallel()
	f := setupSubmitTest(t)

	simResult := buildSimResponse(t, f.op)
	f.rpc.On("SimulateTransaction", mock.Anything, mock.Anything).
		Return(simResult, nil).Once()

	f.rpc.On("SendTransaction", mock.Anything, mock.Anything).
		Return(protocolrpc.SendTransactionResponse{
			Hash:   "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
			Status: "PENDING",
		}, nil)

	metaB64 := buildSuccessMeta(t, nil)
	f.rpc.On("GetTransaction", mock.Anything,
		mock.MatchedBy(func(req protocolrpc.GetTransactionRequest) bool {
			return req.Hash == "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
		})).
		Return(protocolrpc.GetTransactionResponse{
			TransactionDetails: protocolrpc.TransactionDetails{
				Status:        "SUCCESS",
				ResultMetaXDR: metaB64,
			},
		}, nil)

	meta, err := f.invoker.submit(t.Context(), f.op)
	require.NoError(t, err)
	require.NotNil(t, meta)
	require.Equal(t, int32(3), meta.V)
}

func TestSubmit_SimulationError(t *testing.T) {
	t.Parallel()
	f := setupSubmitTest(t)
	f.rpc.On("SimulateTransaction", mock.Anything, mock.Anything).
		Return(protocolrpc.SimulateTransactionResponse{Error: "ContractError(1)"}, nil)

	_, err := f.invoker.submit(t.Context(), f.op)
	require.ErrorContains(t, err, "ContractError")
}

func TestSubmit_ContextCancel(t *testing.T) {
	t.Parallel()
	f := setupSubmitTest(t)
	f.stubSimulate(t)

	f.rpc.On("SendTransaction", mock.Anything, mock.Anything).
		Return(protocolrpc.SendTransactionResponse{
			Hash:   "abcdef01",
			Status: "PENDING",
		}, nil)

	f.rpc.On("GetTransaction", mock.Anything, mock.Anything).
		Return(protocolrpc.GetTransactionResponse{
			TransactionDetails: protocolrpc.TransactionDetails{Status: "NOT_FOUND"},
		}, nil)

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	_, err := f.invoker.submit(ctx, f.op)
	require.ErrorIs(t, err, context.Canceled)
}

func TestSubmit_FailedTransaction(t *testing.T) {
	t.Parallel()
	f := setupSubmitTest(t)
	f.stubSimulate(t)

	f.rpc.On("SendTransaction", mock.Anything, mock.Anything).
		Return(protocolrpc.SendTransactionResponse{
			Hash:   "deadbeef",
			Status: "PENDING",
		}, nil)

	f.rpc.On("GetTransaction", mock.Anything, mock.Anything).
		Return(protocolrpc.GetTransactionResponse{
			TransactionDetails: protocolrpc.TransactionDetails{Status: "FAILED"},
		}, nil)

	_, err := f.invoker.submit(t.Context(), f.op)
	require.ErrorContains(t, err, "stellar transaction failed")
	require.ErrorContains(t, err, "deadbeef")
}

func TestInvokeContract_ReturnsValue(t *testing.T) {
	t.Parallel()
	f := setupSubmitTest(t)
	f.stubSimulate(t)

	f.rpc.On("SendTransaction", mock.Anything, mock.Anything).
		Return(protocolrpc.SendTransactionResponse{
			Hash:   "abcdef01",
			Status: "PENDING",
		}, nil)

	metarv := xdr.ScVal{Type: xdr.ScValTypeScvU32, U32: new(xdr.Uint32)}
	*metarv.U32 = 7
	metaB64 := buildSuccessMeta(t, &metarv)
	f.rpc.On("GetTransaction", mock.Anything, mock.Anything).
		Return(protocolrpc.GetTransactionResponse{
			TransactionDetails: protocolrpc.TransactionDetails{
				Status:        "SUCCESS",
				ResultMetaXDR: metaB64,
			},
		}, nil)

	result, err := f.invoker.InvokeContract(t.Context(), validContractStrkey(t), "get_count", nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, xdr.ScValTypeScvU32, result.Type)
	require.EqualValues(t, 7, *result.U32)
}

func TestGetEvents_ReturnsFilteredEvents(t *testing.T) {
	t.Parallel()
	rpc := mocks.NewRpcClient(t)
	invoker, _ := newTestInvoker(t, rpc)

	expected := []protocolrpc.EventInfo{{
		ContractID: "contract",
		EventType:  "contract",
	}}
	rpc.On("GetEvents", mock.Anything, mock.Anything).
		Return(protocolrpc.GetEventsResponse{Events: expected}, nil)

	events, err := invoker.GetEvents(t.Context(), "contract", 0, []string{"event_topic"})
	require.NoError(t, err)
	require.Equal(t, expected, events)
}

func TestGetEvents_RPCError(t *testing.T) {
	t.Parallel()
	rpc := mocks.NewRpcClient(t)
	invoker, _ := newTestInvoker(t, rpc)

	rpc.On("GetEvents", mock.Anything, mock.Anything).
		Return(protocolrpc.GetEventsResponse{}, fmt.Errorf("rpc error"))

	_, err := invoker.GetEvents(t.Context(), "contract", 0, nil)
	require.ErrorContains(t, err, "rpc error")
}

func TestInvokeOperation_StrkeyDecodeRoundtrip(t *testing.T) {
	t.Parallel()
	var contractID [32]byte
	for i := range contractID {
		contractID[i] = byte(i + 1)
	}
	strkeyID, err := strkey.Encode(strkey.VersionByteContract, contractID[:])
	require.NoError(t, err)

	invoker := newTestInvokerBare(t)
	op, err := invoker.invokeOperation(strkeyID, "fn", nil)
	require.NoError(t, err)

	args := op.HostFunction.InvokeContract
	require.Equal(t, xdr.ScSymbol("fn"), args.FunctionName)
	addrBytes := args.ContractAddress.MustContractId()
	require.Equal(t, xdr.ContractId(contractID), addrBytes)
}

func TestSubmit_RejectsRestorePreamble(t *testing.T) {
	t.Parallel()
	f := setupSubmitTest(t)

	restorePreamble := protocolrpc.RestorePreamble{MinResourceFee: 50000}
	f.rpc.On("SimulateTransaction", mock.Anything, mock.Anything).
		Return(protocolrpc.SimulateTransactionResponse{RestorePreamble: &restorePreamble}, nil)

	_, err := f.invoker.submit(t.Context(), f.op)
	require.ErrorContains(t, err, "ledger restore")
}

func TestSendTransaction_NonPendingStatus(t *testing.T) {
	t.Parallel()
	f := setupSubmitTest(t)
	f.stubSimulate(t)

	f.rpc.On("SendTransaction", mock.Anything, mock.Anything).
		Return(protocolrpc.SendTransactionResponse{
			Hash:   "abc",
			Status: "ERROR",
		}, nil)

	_, err := f.invoker.submit(t.Context(), f.op)
	require.ErrorContains(t, err, "status ERROR")
}

// validContractStrkey generates a valid C... strkey for testing.
func validContractStrkey(t *testing.T) string {
	t.Helper()
	var raw [32]byte
	for i := range raw {
		raw[i] = byte(i + 1)
	}
	id, err := strkey.Encode(strkey.VersionByteContract, raw[:])
	require.NoError(t, err)

	return id
}
