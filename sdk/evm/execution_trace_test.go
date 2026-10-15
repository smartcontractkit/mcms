package evm

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/sdk/evm/mocks"
)

// tracingClient answers debug_traceCall with trace, or fails it with err.
type tracingClient struct {
	*mocks.ContractDeployBackend

	trace  string
	err    error
	method string
	args   []any
}

func (c *tracingClient) CallContext(_ context.Context, result any, method string, args ...any) error {
	c.method, c.args = method, args
	if c.err != nil {
		return c.err
	}

	return json.Unmarshal([]byte(c.trace), result)
}

// errorStringRevert is the hex revert data of revert(reason).
func errorStringRevert(reason string) string {
	return hexutil.Encode(append([]byte{0x08, 0xc3, 0x79, 0xa0}, encodeErrorString(reason)...))
}

// batchTrace is an MCMS execute whose timelock batch fails at its second call
// with failingCall.
func batchTrace(failingCall string) string {
	return `{"error":"execution reverted","calls":[{"error":"execution reverted","output":"` +
		errorStringRevert(timelockUnderlyingRevertMessage) + `","calls":[{"output":"0x"},` + failingCall + `]}]}`
}

func TestTraceUnderlyingRevertReason(t *testing.T) {
	t.Parallel()

	ownableRevert := errorStringRevert("Ownable: caller is not the owner")
	sender := common.HexToAddress("0x1111111111111111111111111111111111111111")
	target := common.HexToAddress("0x2222222222222222222222222222222222222222")
	tx := gethtypes.NewTx(&gethtypes.LegacyTx{To: &target, Data: []byte{0x01, 0x02, 0x03, 0x04}})
	opts := &bind.TransactOpts{From: sender, GasLimit: 1_000_000}

	tests := []struct {
		name        string
		noRPC       bool
		trace       string
		rpcErr      error
		wantRaw     string
		wantDecoded string
		wantOutcome traceOutcome
	}{
		{
			name:        "client without raw RPC access",
			noRPC:       true,
			wantOutcome: traceUnavailable,
		},
		{
			name:        "RPC without the debug namespace",
			rpcErr:      errors.New("the method debug_traceCall does not exist/is not available"),
			wantOutcome: traceUnavailable,
		},
		{
			name:        "transaction no longer reverts",
			trace:       `{"output":"0x"}`,
			wantOutcome: traceSucceeded,
		},
		{
			name:        "later call in the batch reverts",
			trace:       batchTrace(`{"error":"execution reverted","output":"` + ownableRevert + `"}`),
			wantRaw:     ownableRevert,
			wantDecoded: "Ownable: caller is not the owner",
			wantOutcome: traceReverted,
		},
		{
			name:        "failing call runs out of gas",
			trace:       batchTrace(`{"error":"out of gas"}`),
			wantRaw:     "out of gas",
			wantOutcome: traceReverted,
		},
		{
			name:        "undecodable revert keeps the tracer's reason",
			trace:       batchTrace(`{"error":"execution reverted","output":"0xdeadbeef","revertReason":"custom reason"}`),
			wantRaw:     "0xdeadbeef",
			wantDecoded: "custom reason",
			wantOutcome: traceReverted,
		},
		{
			name:        "only the transaction's target fails",
			trace:       `{"error":"execution reverted","output":"` + ownableRevert + `"}`,
			wantOutcome: traceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.noRPC {
				raw, decoded, outcome := traceUnderlyingRevertReason(context.Background(), tx, opts, mocks.NewContractDeployBackend(t))
				assert.Empty(t, raw)
				assert.Empty(t, decoded)
				assert.Equal(t, tt.wantOutcome, outcome)

				return
			}

			client := &tracingClient{ContractDeployBackend: mocks.NewContractDeployBackend(t), trace: tt.trace, err: tt.rpcErr}
			raw, decoded, outcome := traceUnderlyingRevertReason(context.Background(), tx, opts, client)

			assert.Equal(t, tt.wantRaw, raw)
			assert.Equal(t, tt.wantDecoded, decoded)
			assert.Equal(t, tt.wantOutcome, outcome)

			assert.Equal(t, "debug_traceCall", client.method)
			require.Len(t, client.args, 3)
			assert.Equal(t, traceCallArgs{From: sender, To: &target, Gas: 1_000_000, Data: tx.Data()}, client.args[0])
			assert.Equal(t, "latest", client.args[1])
			assert.Equal(t, map[string]string{"tracer": "callTracer"}, client.args[2])
		})
	}
}

func TestBuildExecutionError_underlyingRevertAcrossBatch(t *testing.T) {
	t.Parallel()

	timelockABI, err := bindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err, errMsgFailedTimelockABI)

	call := bindings.RBACTimelockCall{
		Target: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Value:  big.NewInt(0),
		Data:   []byte{0x12, 0x34, 0x56, 0x78},
	}
	oneCall := packBypasserExecuteBatch(t, timelockABI, []bindings.RBACTimelockCall{call})
	twoCalls := packBypasserExecuteBatch(t, timelockABI, []bindings.RBACTimelockCall{call, call})

	mcmsAddr := common.HexToAddress("0x3333333333333333333333333333333333333333")
	timelockAddr := common.HexToAddress("0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0")
	txPreview := gethtypes.NewTx(&gethtypes.LegacyTx{To: &mcmsAddr, Data: []byte{0x01, 0x02, 0x03, 0x04}})
	opts := &bind.TransactOpts{GasLimit: 1_000_000}

	// replayingClient has no trace; its replay of the first call succeeds.
	replayingClient := func(t *testing.T) ContractDeployBackend {
		t.Helper()

		client := mocks.NewContractDeployBackend(t)
		client.On("CodeAt", mock.Anything, mock.Anything, mock.Anything).Return([]byte{0x00}, nil).Maybe()
		client.On("CallContract", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

		return client
	}
	traceClient := func(trace string) func(t *testing.T) ContractDeployBackend {
		return func(t *testing.T) ContractDeployBackend {
			t.Helper()

			return &tracingClient{ContractDeployBackend: mocks.NewContractDeployBackend(t), trace: trace}
		}
	}

	tests := []struct {
		name          string
		client        func(t *testing.T) ContractDeployBackend
		callData      []byte
		wantDecoded   string
		wantLikelyGas bool
	}{
		{
			name:        "trace finds a later call's revert",
			client:      traceClient(batchTrace(`{"error":"execution reverted","output":"` + errorStringRevert("Ownable: caller is not the owner") + `"}`)),
			callData:    twoCalls,
			wantDecoded: "Ownable: caller is not the owner",
		},
		{
			name:          "trace without a revert points at gas",
			client:        traceClient(`{"output":"0x"}`),
			callData:      twoCalls,
			wantLikelyGas: true,
		},
		{
			name:          "without a trace, the batch's only call succeeding points at gas",
			client:        replayingClient,
			callData:      oneCall,
			wantLikelyGas: true,
		},
		{
			name:     "without a trace, the first call succeeding in a larger batch does not point at gas",
			client:   replayingClient,
			callData: twoCalls,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			execErr := BuildExecutionError(
				context.Background(),
				errors.New("execution reverted: "+timelockUnderlyingRevertMessage),
				txPreview,
				opts,
				mcmsAddr,
				tt.client(t),
				timelockAddr,
				tt.callData,
			)

			require.NotNil(t, execErr)
			assert.Equal(t, tt.wantDecoded, execErr.UnderlyingReasonDecoded)
			assert.Equal(t, tt.wantLikelyGas, execErr.IsLikelyGasEstimationFailure)
		})
	}
}

func TestTimelockCallCount(t *testing.T) {
	t.Parallel()

	timelockABI, err := bindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err, errMsgFailedTimelockABI)

	call := bindings.RBACTimelockCall{Target: common.HexToAddress("0x1234"), Value: big.NewInt(0), Data: []byte{0x01}}

	tests := []struct {
		name     string
		callData []byte
		want     int
	}{
		{name: "bypasserExecuteBatch", callData: packBypasserExecuteBatch(t, timelockABI, []bindings.RBACTimelockCall{call, call}), want: 2},
		{name: "executeBatch", callData: packExecuteBatch(t, timelockABI, []bindings.RBACTimelockCall{call}, common.Hash{}, common.Hash{}), want: 1},
		{name: "other timelock method", callData: timelockABI.Methods["scheduleBatch"].ID, want: 0},
		{name: "too short", callData: []byte{0x01}, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, timelockCallCount(tt.callData))
		})
	}
}
