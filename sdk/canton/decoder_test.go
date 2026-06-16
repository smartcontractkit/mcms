package canton

import (
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-canton/bindings/generated/latest/ccip/core"
	"github.com/smartcontractkit/chainlink-canton/bindings/generated/latest/ccip/factory"
	mcmsapi "github.com/smartcontractkit/chainlink-canton/bindings/generated/latest/mcms/api"
	mcmscore "github.com/smartcontractkit/chainlink-canton/bindings/generated/latest/mcms/core"
	cantontypes "github.com/smartcontractkit/go-daml/pkg/types"

	"github.com/smartcontractkit/mcms/types"
)

// operationDataBytes returns the raw wire bytes for a generated choice-argument struct, i.e. the
// bytes the Canton MCMS encoder stores in tx.Data. In the pinned go-daml MarshalHex returns the
// raw encoded bytes directly as a string (not a hex string), so tx.Data = []byte(MarshalHex()).
func operationDataBytes(t *testing.T, v interface{ MarshalHex() (string, error) }) []byte {
	t.Helper()
	s, err := v.MarshalHex()
	require.NoError(t, err)

	return []byte(s)
}

func additionalFields(t *testing.T, af AdditionalFields) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(af)
	require.NoError(t, err)

	return b
}

// TestDecoder_ExerciseChoice decodes a regular exercise choice (RMNRemote::IsCursedForChain),
// whose operationData is the …MCMSParams variant.
func TestDecoder_ExerciseChoice(t *testing.T) {
	t.Parallel()

	params := core.IsCursedForChainMCMSParams{ChainSelector: cantontypes.NUMERIC("1234")}

	tx := types.Transaction{
		To:   "0x0000000000000000000000000000000000000000000000000000000000000001",
		Data: operationDataBytes(t, params),
		AdditionalFields: additionalFields(t, AdditionalFields{
			TargetInstanceAddress: "rmn-remote-1@alice::abc123",
			FunctionName:          "IsCursedForChain",
			TargetTemplateID:      "#pkg:CCIP.RMNRemote:RMNRemote",
		}),
	}

	dec, err := NewDecoder().Decode(tx, "")
	require.NoError(t, err)
	require.Equal(t, "RMNRemote::IsCursedForChain", dec.MethodName())
	require.Equal(t, []string{"chainSelector"}, dec.Keys())
	require.Len(t, dec.Args(), 1)
	require.Equal(t, "1234", dec.Args()[0])
}

// TestDecoder_Deploy decodes a factory deploy choice (CCIPFactory::DeployRMNRemote), whose
// operationData is the …Params variant. All fields are TEXT/PARTY so values are asserted exactly.
func TestDecoder_Deploy(t *testing.T) {
	t.Parallel()

	params := factory.DeployRMNRemoteParams{
		InstanceId:      "rmn-remote-1",
		RmnOwner:        "alice::abc123",
		CcipOwner:       "bob::def456",
		CustomObservers: []cantontypes.PARTY{"carol::ghi789"},
		CursedSubjects:  []cantontypes.TEXT{"0x01"},
	}

	tx := types.Transaction{
		To:   "0x0000000000000000000000000000000000000000000000000000000000000002",
		Data: operationDataBytes(t, params),
		AdditionalFields: additionalFields(t, AdditionalFields{
			TargetInstanceAddress: "ccip-factory-1@alice::abc123",
			FunctionName:          "DeployRMNRemote",
			TargetTemplateID:      "#pkg:CCIP.Factory:CCIPFactory",
		}),
	}

	dec, err := NewDecoder().Decode(tx, "")
	require.NoError(t, err)
	require.Equal(t, "CCIPFactory::DeployRMNRemote", dec.MethodName())
	require.Equal(t, []string{instanceIDFieldLabel, "rmnOwner", "ccipOwner", "customObservers", "cursedSubjects"}, dec.Keys())
	// toDisplayArg strips Daml type aliases to plain Go primitives for the renderer.
	require.Equal(t, "rmn-remote-1", dec.Args()[0])
	require.Equal(t, "alice::abc123", dec.Args()[1])
	require.Equal(t, "bob::def456", dec.Args()[2])
}

// TestDecoder_UnknownChoice surfaces a descriptive error when the (contract, choice) pair is not
// resolvable from any registered encoder.
func TestDecoder_UnknownChoice(t *testing.T) {
	t.Parallel()

	tx := types.Transaction{
		To:   "0x03",
		Data: []byte{0x01, 0x02},
		AdditionalFields: additionalFields(t, AdditionalFields{
			TargetInstanceAddress: "x@alice::abc",
			FunctionName:          "NotARealChoice",
			TargetTemplateID:      "#pkg:CCIP.RMNRemote:RMNRemote",
		}),
	}

	_, err := NewDecoder().Decode(tx, "")
	require.Error(t, err)
}

// TestRegistry_NoDrift guards the reflection registry: each registered encoder must expose
// choices (so renames/removals in the bindings fail loudly), and every encoderByContract value
// must be one of the registered encoders.
// TestCandidateArgTypes verifies that candidateArgTypes returns the right argument struct types
// in the expected priority order (MCMSParams before Params before bare) for known choices.
func TestCandidateArgTypes(t *testing.T) {
	t.Parallel()

	t.Run("known choice returns candidates in MCMSParams-first order", func(t *testing.T) {
		t.Parallel()
		// IsCursedForChain exists as IsCursedForChainMCMSParams on core.MCMSEncoder
		candidates := candidateArgTypes("RMNRemote", "IsCursedForChain")
		require.NotEmpty(t, candidates)
		// The MCMSParams variant must be first
		require.Equal(t, "IsCursedForChainMCMSParams", candidates[0].Name())
	})

	t.Run("preferred encoder is searched before others", func(t *testing.T) {
		t.Parallel()
		// DeployRMNRemote is on factory, not core — encoderByContract should put factory types first
		candidates := candidateArgTypes("CCIPFactory", "DeployRMNRemote")
		require.NotEmpty(t, candidates)
		require.Contains(t, candidates[0].Name(), "DeployRMNRemote")
	})

	t.Run("unknown choice returns empty", func(t *testing.T) {
		t.Parallel()
		candidates := candidateArgTypes("RMNRemote", "NonExistentChoice")
		require.Empty(t, candidates)
	})

	t.Run("no duplicates across encoders", func(t *testing.T) {
		t.Parallel()
		candidates := candidateArgTypes("GlobalConfig", "ApplyDestChainConfigUpdates")
		seen := map[string]bool{}
		for _, c := range candidates {
			require.False(t, seen[c.Name()], "duplicate candidate type %s", c.Name())
			seen[c.Name()] = true
		}
	})
}

// TestRoundTrips verifies the round-trip check: same bytes → true, different bytes → false.
func TestRoundTrips(t *testing.T) {
	t.Parallel()

	params := core.IsCursedForChainMCMSParams{ChainSelector: cantontypes.NUMERIC("42")}
	raw := operationDataBytes(t, params)

	t.Run("correct struct round-trips", func(t *testing.T) {
		t.Parallel()
		decoded := core.IsCursedForChainMCMSParams{}
		require.NoError(t, decoded.UnmarshalHex(hex.EncodeToString(raw)))
		require.True(t, roundTrips(&decoded, raw))
	})

	t.Run("different bytes do not round-trip", func(t *testing.T) {
		t.Parallel()
		// Truncate one byte — re-encoding would produce the original, not the truncated form
		decoded := core.IsCursedForChainMCMSParams{}
		require.NoError(t, decoded.UnmarshalHex(hex.EncodeToString(raw)))
		require.False(t, roundTrips(&decoded, raw[:len(raw)-1]))
	})
}

func TestRegistry_NoDrift(t *testing.T) {
	t.Parallel()

	registered := make(map[reflect.Type]bool, len(mcmsEncoders))
	for _, enc := range mcmsEncoders {
		require.Positive(t, enc.NumMethod(), "encoder %s exposes no methods", enc)
		registered[enc] = true
	}

	for name, enc := range encoderByContract {
		require.True(t, registered[enc], "encoderByContract[%q] is not in mcmsEncoders", name)
	}
}

// TestRegistry_SplitMCMSEncoders guards the chainlink-canton bindings split where MCMS core
// choices (SetRoot, ExecuteOp) and api choices (ScheduleBatch, …) live on separate encoders.
func TestRegistry_SplitMCMSEncoders(t *testing.T) {
	t.Parallel()

	apiEncoder := reflect.TypeFor[mcmsapi.MCMSEncoder]()
	coreEncoder := reflect.TypeFor[mcmscore.MCMSEncoder]()

	require.Equal(t, apiEncoder, encoderByContract["MCMS"])
	require.Contains(t, mcmsEncoders, coreEncoder)
	require.Contains(t, mcmsEncoders, apiEncoder)

	tests := []struct {
		name       string
		choice     string
		wantPkgSeg string
	}{
		{
			name:       "timelock schedule batch",
			choice:     "ScheduleBatch",
			wantPkgSeg: "/mcms/api",
		},
		{
			name:       "set root",
			choice:     "SetRoot",
			wantPkgSeg: "/mcms/core",
		},
		{
			name:       "execute op",
			choice:     "ExecuteOp",
			wantPkgSeg: "/mcms/core",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			candidates := candidateArgTypes("MCMS", tt.choice)
			require.NotEmpty(t, candidates, "expected candidates for MCMS::%s", tt.choice)

			found := false
			for _, c := range candidates {
				if strings.Contains(c.PkgPath(), tt.wantPkgSeg) {
					found = true
					break
				}
			}
			require.True(t, found, "expected a %s candidate from package %s, got %v",
				tt.choice, tt.wantPkgSeg, candidateTypeNames(candidates))
		})
	}
}

func candidateTypeNames(candidateTypes []reflect.Type) []string {
	names := make([]string, len(candidateTypes))
	for i, t := range candidateTypes {
		names[i] = t.String()
	}

	return names
}

// TestDecoder_DeployExecutor_FinalityVariant decodes DeployExecutorParams
// which has a core.FinalityConfig Daml variant (uint8 tag + payload)
func TestDecoder_DeployExecutor_FinalityVariant(t *testing.T) {
	t.Parallel()

	const proposalHex = "0e6578656375746f722d77776268714f636369704f776e65723a3a3132323065333832663465353762303831356536626537333730303665333831653662376465343438653036626430333365636536646634393830313738373966353531000000000000000a0000"
	raw, err := hex.DecodeString(proposalHex)
	require.NoError(t, err)

	tx := types.Transaction{
		To:   "0x2e318f4339676a2fb01d8761982ba5dbe9b6b7578e83f34fdddf20f8c5a17509",
		Data: raw,
		AdditionalFields: additionalFields(t, AdditionalFields{
			TargetInstanceAddress: "factory@ccipOwner::1220e382f4e57b0815e6be737006e381e6b7de448e06bd033ece6df498017879f551",
			FunctionName:          "DeployExecutor",
			TargetTemplateID:      "#pkg:CCIP.Factory:CCIPFactory",
		}),
	}

	dec, err := NewDecoder().Decode(tx, "")
	require.NoError(t, err)
	require.Equal(t, "CCIPFactory::DeployExecutor", dec.MethodName())
	require.Equal(t, []string{instanceIDFieldLabel, "owner", "maxCCVsPerMsg", "allowedFinalityConfig", "ccvAllowlistEnabled"}, dec.Keys())
	require.Equal(t, "executor-wwbhq", dec.Args()[0])
	require.Equal(t, "ccipOwner::1220e382f4e57b0815e6be737006e381e6b7de448e06bd033ece6df498017879f551", dec.Args()[1])
	require.Equal(t, int64(10), dec.Args()[2])

	finality, ok := dec.Args()[3].(map[string]any)
	require.True(t, ok, "allowedFinalityConfig should render as a map, got %T", dec.Args()[3])

	// Proposal has `allowedFinalityConfig: { BlockDepth: null, WaitForFinality: {}, WaitForSafe: null }`
	require.NotNil(t, finality["WaitForFinality"])
	require.Nil(t, finality["WaitForSafe"])
	require.Nil(t, finality["BlockDepth"])
	require.Equal(t, false, dec.Args()[4])

	// The variant must round-trip (decode → re-encode reproduces the proposal bytes), so the
	// decoder selects DeployExecutorParams via the strict path rather than the single-decode fallback.
	decoded, err := decodeOperationData("CCIPFactory", "DeployExecutor", raw)
	require.NoError(t, err)
	params, ok := decoded.(*factory.DeployExecutorParams)
	require.True(t, ok, "expected *factory.DeployExecutorParams, got %T", decoded)
	reEncoded, err := params.MarshalHex()
	require.NoError(t, err)
	require.Equal(t, string(raw), reEncoded)
}

// TestDecoder_DeployRateLimiter_EnumFields verifies that RateLimitDirection and RateLimitMode
// are encoded and decoded as single ordinal bytes (matching the Daml MCMS codec wire format),
// not as length-prefixed constructor-name strings. Before the fix, the encoder emitted e.g.
// "\x1b" + "RateLimitDirection_Outbound" (28 bytes) where the ledger reads 1 byte → rejects.
func TestDecoder_DeployRateLimiter_EnumFields(t *testing.T) {
	t.Parallel()

	params := factory.DeployRateLimiterParams{
		InstanceId:          "rl-outbound",
		PoolInstanceId:      "pool-1",
		PoolOwner:           cantontypes.PARTY("owner::abc123"),
		RemoteChainSelector: cantontypes.NUMERIC("16015286601757825753"),
		Direction:           core.RateLimitDirectionRateLimitDirection_Outbound,
		Mode:                core.RateLimitModeRateLimitMode_DefaultFinality,
		IsEnabled:           true,
		Capacity:            cantontypes.NUMERIC("1000"),
		Rate:                cantontypes.NUMERIC("10"),
	}

	raw := operationDataBytes(t, params)

	// Direction must be a single byte (0x00 = Outbound), not 28 bytes for the constructor name.
	// Byte layout: 1+11 instanceId | 1+6 poolInstanceId | 1+13 poolOwner | 1+20 chainSelector | dir | mode
	// Direction at offset 54, Mode at offset 55.
	require.Greater(t, len(raw), 56, "encoded too short; enum is likely still a string")
	require.Equal(t, byte(0x00), raw[54], "Direction Outbound must be single byte 0x00")
	require.Equal(t, byte(0x00), raw[55], "Mode DefaultFinality must be single byte 0x00")

	tx := types.Transaction{
		To:   "0x0000000000000000000000000000000000000000000000000000000000000003",
		Data: raw,
		AdditionalFields: additionalFields(t, AdditionalFields{
			TargetInstanceAddress: "factory@owner::abc123",
			FunctionName:          "DeployRateLimiter",
			TargetTemplateID:      "#pkg:CCIP.Factory:CCIPFactory",
		}),
	}

	dec, err := NewDecoder().Decode(tx, "")
	require.NoError(t, err)
	require.Equal(t, "CCIPFactory::DeployRateLimiter", dec.MethodName())
	require.Equal(t, []string{"instanceId", "poolInstanceId", "poolOwner", "remoteChainSelector", "direction", "mode", "isEnabled", "capacity", "rate"}, dec.Keys())
	require.Equal(t, "rl-outbound", dec.Args()[0])
	// direction and mode decode back to their constructor-name strings via reflection
	require.Equal(t, "RateLimitDirection_Outbound", dec.Args()[4])
	require.Equal(t, "RateLimitMode_DefaultFinality", dec.Args()[5])

	// Round-trip: decoded params must re-encode to the same bytes (strict path, not fallback)
	decoded, err := decodeOperationData("CCIPFactory", "DeployRateLimiter", raw)
	require.NoError(t, err)
	rlp, ok := decoded.(*factory.DeployRateLimiterParams)
	require.True(t, ok, "expected *factory.DeployRateLimiterParams, got %T", decoded)
	reEncoded, err := rlp.MarshalHex()
	require.NoError(t, err)
	require.Equal(t, string(raw), reEncoded, "round-trip must reproduce original bytes")
}
