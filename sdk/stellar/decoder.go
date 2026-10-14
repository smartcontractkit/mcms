package stellar

import (
	"fmt"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Decoder = (*Decoder)(nil)

// Decoder implements sdk.Decoder for Stellar (Soroban) MCMS transactions.
//
// A Soroban invocation is self-describing — a function symbol plus
// positional, typed arguments — so unlike EVM there is no ABI to resolve and
// the contractInterfaces argument is ignored.
type Decoder struct{}

// NewDecoder returns a new Stellar MCMS decoder.
func NewDecoder() *Decoder {
	return &Decoder{}
}

// Decode implements sdk.Decoder. It reverses what NewTransaction encodes into
// types.Transaction.Data, so round-tripping holds by construction.
func (d *Decoder) Decode(op types.Transaction, _ string) (sdk.DecodedOperation, error) {
	if len(op.Data) == 0 {
		return nil, fmt.Errorf("stellar transaction has no invoke payload")
	}

	payload, err := DecodeSorobanInvokePayload(op.Data)
	if err != nil {
		return nil, fmt.Errorf("decode stellar invoke payload: %w", err)
	}

	return NewDecodedOperation(op.ContractType, payload.Function, payload.Args)
}
