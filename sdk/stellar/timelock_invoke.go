package stellar

import (
	"fmt"

	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"
	"github.com/stellar/go-stellar-sdk/xdr"
)

// sorobanInvokePayloadBytes encodes MCMS/timelock inner call data as XDR for
// ScVec([Symbol(fnName), ...args]), matching decode_invoke_payload in
// chainlink-stellar/contracts/common/helpers/src/soroban_invoke.rs.
func sorobanInvokePayloadBytes(fnName string, args ...xdr.ScVal) ([]byte, error) {
	items := make([]xdr.ScVal, 0, 1+len(args))
	items = append(items, scval.SymbolToScVal(fnName))
	items = append(items, args...)
	val := scval.VecToScVal(items)

	b, err := val.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshal soroban invoke payload: %w", err)
	}

	return b, nil
}
