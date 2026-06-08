package canton

import (
	"encoding/hex"
	"fmt"
	"reflect"

	"github.com/smartcontractkit/chainlink-canton/bindings/generated/latest/ccip/burnminttokenpool"
	"github.com/smartcontractkit/chainlink-canton/bindings/generated/latest/ccip/ccipruntime"
	"github.com/smartcontractkit/chainlink-canton/bindings/generated/latest/ccip/committeeverifier"
	"github.com/smartcontractkit/chainlink-canton/bindings/generated/latest/ccip/core"
	"github.com/smartcontractkit/chainlink-canton/bindings/generated/latest/ccip/executor"
	"github.com/smartcontractkit/chainlink-canton/bindings/generated/latest/ccip/factory"
	"github.com/smartcontractkit/chainlink-canton/bindings/generated/latest/ccip/lockreleasetokenpool"
	"github.com/smartcontractkit/chainlink-canton/bindings/generated/latest/ccip/receiver"
	"github.com/smartcontractkit/chainlink-canton/bindings/generated/latest/ccip/sender"
	"github.com/smartcontractkit/chainlink-canton/bindings/generated/latest/coin"
	"github.com/smartcontractkit/chainlink-canton/bindings/generated/latest/link"
	"github.com/smartcontractkit/chainlink-canton/bindings/generated/latest/mcms"
)

// hexUnmarshaler / hexMarshaler are implemented by every generated Canton choice-argument struct.
type hexUnmarshaler interface{ UnmarshalHex(string) error }

type hexMarshaler interface{ MarshalHex() (string, error) }

// mcmsEncoders is the set of distinct generated MCMSEncoder interfaces covering every
// MCMS-governed contract (one per ContractType in chainlink-canton/deployment/operations).

// Choices and their arg types are derived from these interfaces by reflection rather than a
// hand-maintained map, so the registry cannot drift when bindings regenerate. To add a new
// governed contract add one line here; skip it if its MCMSEncoder is an alias of one already listed.
var mcmsEncoders = []reflect.Type{
	reflect.TypeFor[core.MCMSEncoder](),                 // GlobalConfig, FeeQuoter, RMNRemote, TokenAdminRegistry
	reflect.TypeFor[ccipruntime.MCMSEncoder](),          // PerPartyRouterFactory, OnRamp, OffRamp
	reflect.TypeFor[committeeverifier.MCMSEncoder](),    // CommitteeVerifier
	reflect.TypeFor[sender.MCMSEncoder](),               // CCIPSender
	reflect.TypeFor[receiver.MCMSEncoder](),             // CCIPReceiver
	reflect.TypeFor[burnminttokenpool.MCMSEncoder](),    // BurnMintTokenPool
	reflect.TypeFor[lockreleasetokenpool.MCMSEncoder](), // LockReleaseTokenPool
	reflect.TypeFor[executor.MCMSEncoder](),             // Executor
	reflect.TypeFor[factory.MCMSEncoder](),              // CCIPFactory (deploys)
	reflect.TypeFor[coin.MCMSEncoder](),                 // CoinRegistry (CantonCoinRegistry)
	reflect.TypeFor[link.MCMSEncoder](),                 // LinkRegistry
	reflect.TypeFor[mcms.MCMSEncoder](),                 // MCMS (CantonMCMS)
}

// encoderByContract maps a Daml template entity name to the MCMSEncoder that declares its choices.
// It is a hint to try the right encoder first; a missing entry costs a wider search but never a
// wrong decode — correctness comes from the round-trip check in decodeOperationData.
// Always add an entry when registering a new contract in mcmsEncoders.
var encoderByContract = map[string]reflect.Type{
	"BurnMintTokenPool":     reflect.TypeFor[burnminttokenpool.MCMSEncoder](),
	"LockReleaseTokenPool":  reflect.TypeFor[lockreleasetokenpool.MCMSEncoder](),
	"CommitteeVerifier":     reflect.TypeFor[committeeverifier.MCMSEncoder](),
	"Executor":              reflect.TypeFor[executor.MCMSEncoder](),
	"CCIPFactory":           reflect.TypeFor[factory.MCMSEncoder](),
	"CCIPSender":            reflect.TypeFor[sender.MCMSEncoder](),
	"CCIPReceiver":          reflect.TypeFor[receiver.MCMSEncoder](),
	"PerPartyRouterFactory": reflect.TypeFor[ccipruntime.MCMSEncoder](),
	"OnRamp":                reflect.TypeFor[ccipruntime.MCMSEncoder](),
	"OffRamp":               reflect.TypeFor[ccipruntime.MCMSEncoder](),
	"GlobalConfig":          reflect.TypeFor[core.MCMSEncoder](),
	"FeeQuoter":             reflect.TypeFor[core.MCMSEncoder](),
	"RMNRemote":             reflect.TypeFor[core.MCMSEncoder](),
	"TokenAdminRegistry":    reflect.TypeFor[core.MCMSEncoder](),
	"CoinRegistry":          reflect.TypeFor[coin.MCMSEncoder](),
	"LinkRegistry":          reflect.TypeFor[link.MCMSEncoder](),
	"MCMS":                  reflect.TypeFor[mcms.MCMSEncoder](),
}

// candidateArgTypes returns candidate argument struct types for the given choice, gathered from
// the preferred encoder first then the full set. It looks up <choice>{MCMSParams,Params,""} on
// each encoder — the suffix varies by operation (token pools use MCMSParams, factory deploys use
// Params, some setters use the bare name) — so all three are collected and the round-trip selects.
func candidateArgTypes(contractType, choice string) []reflect.Type {
	seen := map[reflect.Type]bool{}
	var out []reflect.Type
	add := func(enc reflect.Type) {
		if enc == nil {
			return
		}
		for _, name := range []string{choice + "MCMSParams", choice + "Params", choice} {
			if m, ok := enc.MethodByName(name); ok && m.Type.NumIn() >= 1 {
				if t := m.Type.In(0); !seen[t] {
					seen[t] = true
					out = append(out, t)
				}
			}
		}
	}
	add(encoderByContract[contractType])
	for _, enc := range mcmsEncoders {
		add(enc)
	}

	return out
}

// decodeOperationData finds the generated choice-argument struct that matches the raw operation
// bytes and returns a pointer to it.
//
// Selection is by hex round-trip: a candidate passes iff re-MarshalHex reproduces the original
// bytes exactly. This is necessary because go-daml's Unmarshal ignores trailing bytes — a
// shorter struct (e.g. FooParams with 2 fields) can silently decode bytes that belong to FooMCMSParams
// (3 fields) without error, but when re-encoded it produces fewer bytes, failing the comparison.
//
// Fallback: if exactly one candidate decoded but none round-tripped, that candidate is accepted.
// This tolerates the one known case where round-trip can fail on the correct type: TEXTMAP fields
// iterate Go maps whose key order is randomized, so re-encoding may produce different key ordering.
// Current CCIP choice-argument structs do not use TEXTMAP (they use ordered slices), but the
// fallback is a safety net for future contracts. If two or more candidates decode, we error rather
// than guess.
func decodeOperationData(contractType, choice string, raw []byte) (any, error) {
	hexData := hex.EncodeToString(raw)

	candidates := candidateArgTypes(contractType, choice)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no registered choice %q for contract %q", choice, contractType)
	}

	var decodedOnce any
	decodedCount := 0
	for _, t := range candidates {
		ptr := reflect.New(t)
		hu, ok := ptr.Interface().(hexUnmarshaler)
		if !ok {
			continue
		}
		if err := hu.UnmarshalHex(hexData); err != nil {
			continue
		}
		decodedCount++
		if decodedOnce == nil {
			decodedOnce = ptr.Interface()
		}
		if hm, ok := ptr.Interface().(hexMarshaler); ok && roundTrips(hm, raw) {
			return ptr.Interface(), nil
		}
	}

	if decodedCount == 1 {
		return decodedOnce, nil
	}
	if decodedCount > 1 {
		return nil, fmt.Errorf("ambiguous operationData for %s::%s: %d candidate types decoded but none round-tripped", contractType, choice, decodedCount)
	}

	return nil, fmt.Errorf("could not decode operationData for %s::%s with any candidate type", contractType, choice)
}

// roundTrips reports whether re-encoding v reproduces the original operation bytes.
func roundTrips(v hexMarshaler, raw []byte) bool {
	out, err := v.MarshalHex()
	if err != nil {
		return false
	}

	return out == string(raw)
}
