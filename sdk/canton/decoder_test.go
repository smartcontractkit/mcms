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
	require.Equal(t, []string{"instanceId", "rmnOwner", "ccipOwner", "customObservers", "cursedSubjects"}, dec.Keys())
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

func candidateTypeNames(types []reflect.Type) []string {
	names := make([]string, len(types))
	for i, t := range types {
		names[i] = t.String()
	}
	return names
}
