package stellar

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/stellar/go-stellar-sdk/strkey"
	"github.com/stellar/go-stellar-sdk/xdr"
)

// maxScValDisplayDepth bounds recursion when converting nested Soroban
// vectors and maps, mirroring maxScValErrorDecodeDepth in error_decoder.go.
const maxScValDisplayDepth = 10

// Bit layout for reassembling Soroban 128/256-bit integers from 64-bit limbs.
const (
	limbBits        = 64
	shiftTwoLimbs   = 128
	shiftThreeLimbs = 192
	i128SignBit     = 127
	i128WidthBits   = 128
	i256SignBit     = 255
	i256WidthBits   = 256
)

// ScValToDisplay converts an arbitrary Soroban value into a JSON-friendly Go
// value for proposal review tooling. Addresses become strkey contract/account
// IDs, bytes become 0x-prefixed hex, i128/u128/i256/u256 become decimal
// strings (they do not fit in int64), and vectors and maps recurse.
// Unrecognized future ScVal types degrade to a descriptive string rather than
// panicking, so a new Soroban type can never break the proposal pipeline.
func ScValToDisplay(v xdr.ScVal) any {
	return scValToDisplay(v, 0)
}

func scValToDisplay(v xdr.ScVal, depth int) any {
	if depth > maxScValDisplayDepth {
		return fmt.Sprintf("<soroban value nested deeper than %d levels>", maxScValDisplayDepth)
	}

	switch v.Type {
	case xdr.ScValTypeScvBool:
		if v.B != nil {
			return *v.B
		}

	case xdr.ScValTypeScvVoid:
		return nil

	case xdr.ScValTypeScvError:
		if v.Error != nil {
			return fmt.Sprintf("soroban error: %v", *v.Error)
		}

	case xdr.ScValTypeScvU32:
		if v.U32 != nil {
			return uint32(*v.U32)
		}

	case xdr.ScValTypeScvI32:
		if v.I32 != nil {
			return int32(*v.I32)
		}

	case xdr.ScValTypeScvU64:
		if v.U64 != nil {
			return uint64(*v.U64)
		}

	case xdr.ScValTypeScvI64:
		if v.I64 != nil {
			return int64(*v.I64)
		}

	case xdr.ScValTypeScvTimepoint:
		if v.Timepoint != nil {
			return uint64(*v.Timepoint)
		}

	case xdr.ScValTypeScvDuration:
		if v.Duration != nil {
			return uint64(*v.Duration)
		}

	case xdr.ScValTypeScvU128:
		if v.U128 != nil {
			return u128String(*v.U128)
		}

	case xdr.ScValTypeScvI128:
		if v.I128 != nil {
			return i128String(*v.I128)
		}

	case xdr.ScValTypeScvU256:
		if v.U256 != nil {
			return u256String(*v.U256)
		}

	case xdr.ScValTypeScvI256:
		if v.I256 != nil {
			return i256String(*v.I256)
		}

	case xdr.ScValTypeScvBytes:
		if v.Bytes != nil {
			return "0x" + hex.EncodeToString(*v.Bytes)
		}

	case xdr.ScValTypeScvString:
		if v.Str != nil {
			return string(*v.Str)
		}

	case xdr.ScValTypeScvSymbol:
		if v.Sym != nil {
			return string(*v.Sym)
		}

	case xdr.ScValTypeScvVec:
		if v.Vec != nil && *v.Vec != nil {
			out := make([]any, 0, len(**v.Vec))
			for _, entry := range **v.Vec {
				out = append(out, scValToDisplay(entry, depth+1))
			}

			return out
		}

	case xdr.ScValTypeScvMap:
		if v.Map != nil && *v.Map != nil {
			return scMapToDisplay(**v.Map, depth)
		}

	case xdr.ScValTypeScvAddress:
		if v.Address != nil {
			return scAddressToDisplay(*v.Address)
		}

	case xdr.ScValTypeScvContractInstance:
		if v.Instance != nil {
			return "<contract instance>"
		}

	case xdr.ScValTypeScvLedgerKeyContractInstance, xdr.ScValTypeScvLedgerKeyNonce,
		xdr.ScValTypeScvExecutableTag:
		return fmt.Sprintf("<soroban value type %d>", int32(v.Type))
	}

	// Covers unset fields of a known type and any future ScValType.
	return fmt.Sprintf("<unsupported soroban value type %d>", int32(v.Type))
}

// scMapToDisplay converts a Soroban map into a Go map keyed by the string
// representation of each entry key (typically symbols or strings).
func scMapToDisplay(entries xdr.ScMap, depth int) map[string]any {
	out := make(map[string]any, len(entries))

	for _, entry := range entries {
		key := fmt.Sprintf("%v", scValToDisplay(entry.Key, depth+1))
		out[key] = scValToDisplay(entry.Val, depth+1)
	}

	return out
}

// scAddressToDisplay renders a Soroban address as its strkey form: C... for
// contracts, G... for accounts.
func scAddressToDisplay(address xdr.ScAddress) string {
	switch address.Type {
	case xdr.ScAddressTypeScAddressTypeAccount:
		if address.AccountId != nil {
			return address.AccountId.Address()
		}

	case xdr.ScAddressTypeScAddressTypeContract:
		if address.ContractId != nil {
			return strkey.MustEncode(strkey.VersionByteContract, address.ContractId[:])
		}

	case xdr.ScAddressTypeScAddressTypeMuxedAccount,
		xdr.ScAddressTypeScAddressTypeClaimableBalance,
		xdr.ScAddressTypeScAddressTypeLiquidityPool:
		// Not representable as a plain strkey; fall through to the summary.
	}

	return fmt.Sprintf("<unsupported soroban address type %d>", int32(address.Type))
}

// u128String renders a Soroban uint128 as a decimal string.
func u128String(parts xdr.UInt128Parts) string {
	value := new(big.Int).SetUint64(uint64(parts.Lo))
	value.Or(value, new(big.Int).Lsh(new(big.Int).SetUint64(uint64(parts.Hi)), limbBits))

	return value.String()
}

// i128String renders a Soroban int128 as a decimal string, applying two's
// complement for negative values.
func i128String(parts xdr.Int128Parts) string {
	value := new(big.Int).SetUint64(uint64(parts.Lo))
	//nolint:gosec // G115: intentional two's-complement bit reassembly.
	value.Or(value, new(big.Int).Lsh(new(big.Int).SetUint64(uint64(parts.Hi)), limbBits))
	if value.Bit(i128SignBit) == 1 {
		value.Sub(value, new(big.Int).Lsh(big.NewInt(1), i128WidthBits))
	}

	return value.String()
}

// u256String renders a Soroban uint256 as a decimal string. The parts hold
// four 64-bit limbs, most-significant first in field order.
func u256String(parts xdr.UInt256Parts) string {
	value := new(big.Int).SetUint64(uint64(parts.LoLo))
	value.Or(value, new(big.Int).Lsh(new(big.Int).SetUint64(uint64(parts.LoHi)), limbBits))
	value.Or(value, new(big.Int).Lsh(new(big.Int).SetUint64(uint64(parts.HiLo)), shiftTwoLimbs))
	value.Or(value, new(big.Int).Lsh(new(big.Int).SetUint64(uint64(parts.HiHi)), shiftThreeLimbs))

	return value.String()
}

// i256String renders a Soroban int256 as a decimal string, applying two's
// complement for negative values.
func i256String(parts xdr.Int256Parts) string {
	value := new(big.Int).SetUint64(uint64(parts.LoLo))
	value.Or(value, new(big.Int).Lsh(new(big.Int).SetUint64(uint64(parts.LoHi)), limbBits))
	value.Or(value, new(big.Int).Lsh(new(big.Int).SetUint64(uint64(parts.HiLo)), shiftTwoLimbs))
	//nolint:gosec // G115: intentional two's-complement bit reassembly.
	value.Or(value, new(big.Int).Lsh(new(big.Int).SetUint64(uint64(parts.HiHi)), shiftThreeLimbs))
	if value.Bit(i256SignBit) == 1 {
		value.Sub(value, new(big.Int).Lsh(big.NewInt(1), i256WidthBits))
	}

	return value.String()
}
