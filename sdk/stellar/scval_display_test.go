package stellar_test

import (
	"fmt"
	"testing"

	"github.com/stellar/go-stellar-sdk/xdr"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/stellar"
)

func scvBool(b bool) xdr.ScVal { return xdr.ScVal{Type: xdr.ScValTypeScvBool, B: &b} }

func scvU32(v uint32) xdr.ScVal {
	w := xdr.Uint32(v)

	return xdr.ScVal{Type: xdr.ScValTypeScvU32, U32: &w}
}

func scvI32(v int32) xdr.ScVal {
	w := xdr.Int32(v)

	return xdr.ScVal{Type: xdr.ScValTypeScvI32, I32: &w}
}

func scvU64(v uint64) xdr.ScVal {
	w := xdr.Uint64(v)

	return xdr.ScVal{Type: xdr.ScValTypeScvU64, U64: &w}
}

func scvI64(v int64) xdr.ScVal {
	w := xdr.Int64(v)

	return xdr.ScVal{Type: xdr.ScValTypeScvI64, I64: &w}
}

func scvTimepoint(v xdr.TimePoint) xdr.ScVal {
	w := v

	return xdr.ScVal{Type: xdr.ScValTypeScvTimepoint, Timepoint: &w}
}

func scvU128(hi, lo uint64) xdr.ScVal {
	parts := xdr.UInt128Parts{Hi: xdr.Uint64(hi), Lo: xdr.Uint64(lo)}

	return xdr.ScVal{Type: xdr.ScValTypeScvU128, U128: &parts}
}

func scvI128(hi int64, lo uint64) xdr.ScVal {
	parts := xdr.Int128Parts{Hi: xdr.Int64(hi), Lo: xdr.Uint64(lo)}

	return xdr.ScVal{Type: xdr.ScValTypeScvI128, I128: &parts}
}

func scvU256(hiHi, hiLo, loHi, loLo uint64) xdr.ScVal {
	parts := xdr.UInt256Parts{
		HiHi: xdr.Uint64(hiHi), HiLo: xdr.Uint64(hiLo),
		LoHi: xdr.Uint64(loHi), LoLo: xdr.Uint64(loLo),
	}

	return xdr.ScVal{Type: xdr.ScValTypeScvU256, U256: &parts}
}

func scvI256(hiHi int64, hiLo, loHi, loLo uint64) xdr.ScVal {
	parts := xdr.Int256Parts{
		HiHi: xdr.Int64(hiHi), HiLo: xdr.Uint64(hiLo),
		LoHi: xdr.Uint64(loHi), LoLo: xdr.Uint64(loLo),
	}

	return xdr.ScVal{Type: xdr.ScValTypeScvI256, I256: &parts}
}

func scvBytes(b []byte) xdr.ScVal {
	v := xdr.ScBytes(b)

	return xdr.ScVal{Type: xdr.ScValTypeScvBytes, Bytes: &v}
}

func scvStr(s string) xdr.ScVal {
	v := xdr.ScString(s)

	return xdr.ScVal{Type: xdr.ScValTypeScvString, Str: &v}
}

func scvSym(s string) xdr.ScVal {
	v := xdr.ScSymbol(s)

	return xdr.ScVal{Type: xdr.ScValTypeScvSymbol, Sym: &v}
}

func scvVec(entries ...xdr.ScVal) xdr.ScVal {
	vec := xdr.ScVec(entries)
	ptr := &vec

	return xdr.ScVal{Type: xdr.ScValTypeScvVec, Vec: &ptr}
}

func scvMap(pairs ...xdr.ScMapEntry) xdr.ScVal {
	m := xdr.ScMap(pairs)
	ptr := &m

	return xdr.ScVal{Type: xdr.ScValTypeScvMap, Map: &ptr}
}

func mapEntry(key, value xdr.ScVal) xdr.ScMapEntry {
	entry := xdr.ScMapEntry{Key: key, Val: value}

	return entry
}

func TestScValToDisplay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   xdr.ScVal
		want any
	}{
		{name: "bool true", in: scvBool(true), want: true},
		{name: "bool false", in: scvBool(false), want: false},
		{name: "void", in: xdr.ScVal{Type: xdr.ScValTypeScvVoid}, want: nil},
		{name: "u32", in: scvU32(4294967295), want: uint32(4294967295)},
		{name: "i32 negative", in: scvI32(-1), want: int32(-1)},
		{name: "u64", in: scvU64(18446744073709551615), want: uint64(18446744073709551615)},
		{name: "i64 negative", in: scvI64(-1), want: int64(-1)},
		{name: "timepoint", in: scvTimepoint(1726444800), want: uint64(1726444800)},
		{name: "duration", in: scvU64(3600), want: uint64(3600)},
		{name: "u128 max", in: scvU128(18446744073709551615, 18446744073709551615),
			want: "340282366920938463463374607431768211455"},
		{name: "u128 hi only", in: scvU128(1, 0), want: "18446744073709551616"},
		{name: "i128 positive", in: scvI128(0, 42), want: "42"},
		{name: "i128 negative", in: scvI128(-1, 18446744073709551615), want: "-1"},
		{name: "u256", in: scvU256(0, 0, 0, 1), want: "1"},
		{name: "u256 max", in: scvU256(18446744073709551615, 18446744073709551615, 18446744073709551615, 18446744073709551615),
			want: "115792089237316195423570985008687907853269984665640564039457584007913129639935"},
		{name: "i256 negative", in: scvI256(-1, 18446744073709551615, 18446744073709551615, 18446744073709551615),
			want: "-1"},
		{name: "bytes", in: scvBytes([]byte{0xde, 0xad}), want: "0xdead"},
		{name: "string", in: scvStr("hello"), want: "hello"},
		{name: "symbol", in: scvSym("set_config"), want: "set_config"},
		{
			name: "vec of scalars",
			in:   scvVec(scvU32(1), scvStr("two")),
			want: []any{uint32(1), "two"},
		},
		{
			name: "nested vec of vec",
			in:   scvVec(scvVec(scvBool(true))),
			want: []any{[]any{true}},
		},
		{
			name: "map with symbol keys",
			in: scvMap(
				mapEntry(scvSym("amount"), scvU128(0, 100)),
				mapEntry(scvSym("target"), scvStr("router")),
			),
			want: map[string]any{
				"amount": "100",
				"target": "router",
			},
		},
		{
			name: "map with vec values",
			in: scvMap(
				mapEntry(scvSym("calls"), scvVec(scvU32(1), scvU32(2))),
			),
			want: map[string]any{
				"calls": []any{uint32(1), uint32(2)},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.want, stellar.ScValToDisplay(tc.in))
		})
	}
}

func TestScValToDisplay_ContractAddress(t *testing.T) {
	t.Parallel()

	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = byte(200 + i)
	}

	var contractID xdr.ContractId
	copy(contractID[:], raw)

	address := xdr.ScAddress{Type: xdr.ScAddressTypeScAddressTypeContract, ContractId: &contractID}
	got := stellar.ScValToDisplay(xdr.ScVal{Type: xdr.ScValTypeScvAddress, Address: &address})
	require.Equal(t, testContractID(t, 200), got)
}

func TestScValToDisplay_UnknownTypeForwardCompat(t *testing.T) {
	t.Parallel()

	// An ScValType value the walker does not know must degrade to a
	// descriptive string, never panic.
	future := xdr.ScVal{Type: xdr.ScValType(99)}
	got := stellar.ScValToDisplay(future)
	require.Equal(t, "<unsupported soroban value type 99>", got)
}

func TestScValToDisplay_UnsetArmOfKnownType(t *testing.T) {
	t.Parallel()

	// A known type whose field pointer is nil must degrade, not panic.
	got := stellar.ScValToDisplay(xdr.ScVal{Type: xdr.ScValTypeScvBool})
	require.Equal(t, "<unsupported soroban value type 0>", got)
}

func TestScValToDisplay_DepthGuard(t *testing.T) {
	t.Parallel()

	// Keep in sync with maxScValDisplayDepth in sdk/stellar/scval_display.go.
	const maxDisplayDepth = 10

	nested := scvBool(true)
	for range maxDisplayDepth + 2 {
		nested = scvVec(nested)
	}

	got := stellar.ScValToDisplay(nested)

	// The innermost value sits beyond the depth limit and must be replaced
	// by the depth marker rather than recursing without bound.
	require.Contains(t, fmt.Sprintf("%v", got), fmt.Sprintf("nested deeper than %d levels", maxDisplayDepth))
}
