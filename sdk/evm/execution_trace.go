package evm

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

// traceOutcome is what tracing a failed transaction found.
type traceOutcome int

const (
	// traceUnavailable means the RPC can't trace, or the trace has no failing
	// call below the transaction's own target.
	traceUnavailable traceOutcome = iota
	// traceSucceeded means the transaction doesn't revert with the simulation
	// gas limit.
	traceSucceeded
	// traceReverted means a call inside the transaction reverts.
	traceReverted
)

// rpcCaller is the raw JSON-RPC access debug_traceCall needs. *rpc.Client
// implements it.
type rpcCaller interface {
	CallContext(ctx context.Context, result any, method string, args ...any) error
}

// rpcCallerOf returns client's raw JSON-RPC access, or nil when it has none.
// *ethclient.Client, and clients embedding it, expose it through Client().
func rpcCallerOf(client ContractDeployBackend) rpcCaller {
	if c, ok := client.(interface{ Client() *rpc.Client }); ok {
		if rpcClient := c.Client(); rpcClient != nil {
			return rpcClient
		}

		return nil
	}
	if c, ok := client.(rpcCaller); ok {
		return c
	}

	return nil
}

// traceCallArgs is the transaction debug_traceCall simulates.
type traceCallArgs struct {
	From  common.Address  `json:"from"`
	To    *common.Address `json:"to"`
	Gas   hexutil.Uint64  `json:"gas"`
	Value *hexutil.Big    `json:"value,omitempty"`
	Data  hexutil.Bytes   `json:"data"`
}

// traceCallFrame is one call in the callTracer's output.
type traceCallFrame struct {
	Output       hexutil.Bytes    `json:"output,omitempty"`
	Error        string           `json:"error,omitempty"`
	RevertReason string           `json:"revertReason,omitempty"`
	Calls        []traceCallFrame `json:"calls,omitempty"`
}

// traceUnderlyingRevertReason simulates tx with debug_traceCall's callTracer
// and returns the raw and decoded revert of its innermost failing call. The
// trace runs the whole transaction, so a failing call in a timelock batch sees
// the effects of the calls before it, unlike a replay of that call alone.
func traceUnderlyingRevertReason(
	ctx context.Context, tx *gethtypes.Transaction, opts *bind.TransactOpts, client ContractDeployBackend,
) (string, string, traceOutcome) {
	caller := rpcCallerOf(client)
	if caller == nil || tx == nil || tx.To() == nil {
		return "", "", traceUnavailable
	}

	args := traceCallArgs{
		To:   tx.To(),
		Gas:  hexutil.Uint64(simulationGasLimit(ctx, opts, client)),
		Data: tx.Data(),
	}
	if opts != nil {
		args.From = opts.From
	}
	if value := tx.Value(); value != nil && value.Sign() > 0 {
		args.Value = (*hexutil.Big)(value)
	}

	var frame traceCallFrame
	err := caller.CallContext(ctx, &frame, "debug_traceCall", args, "latest", map[string]string{"tracer": "callTracer"})
	if err != nil {
		return "", "", traceUnavailable
	}
	if frame.Error == "" {
		return "", "", traceSucceeded
	}

	failed := innermostFailure(&frame)
	if failed == &frame {
		// Only the transaction's own target failed; its revert is decoded from
		// the original error.
		return "", "", traceUnavailable
	}

	decoded := decodeRevertReason(failed.Output)
	if decoded == "" {
		decoded = failed.RevertReason
	}
	if len(failed.Output) == 0 {
		return failed.Error, decoded, traceReverted
	}

	return hexutil.Encode(failed.Output), decoded, traceReverted
}

// innermostFailure follows the failing calls down from frame and returns the
// deepest one. A call fails because of the last of its calls that failed;
// earlier ones were caught.
func innermostFailure(frame *traceCallFrame) *traceCallFrame {
	for i := len(frame.Calls) - 1; i >= 0; i-- {
		if frame.Calls[i].Error != "" {
			return innermostFailure(&frame.Calls[i])
		}
	}

	return frame
}
