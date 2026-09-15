package stellar_test

import (
	"testing"

	"github.com/stellar/go-stellar-sdk/xdr"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/stellar"
	"github.com/smartcontractkit/mcms/types"
)

func TestDecoder_Decode_RoundTrip(t *testing.T) {
	t.Parallel()

	target := testContractID(t, 140)

	// The arguments exercise the display conversions the analyzer relies on.
	args := []xdr.ScVal{
		scvSym("set_config"),
		scvU128(0, 1000),
		scvBool(true),
		scvVec(scvStr("a"), scvBytes([]byte{0x01})),
	}

	tx, err := stellar.NewTransaction(target, "configure", args, "FeeQuoter", []string{"CCIP"})
	require.NoError(t, err)

	decoded, err := stellar.NewDecoder().Decode(tx, "ignored-contract-interfaces")
	require.NoError(t, err)

	require.Equal(t, "FeeQuoter::configure", decoded.MethodName())
	require.Equal(t, []string{"arg0", "arg1", "arg2", "arg3"}, decoded.Keys())
	require.Equal(t, []any{
		"set_config",
		"1000",
		true,
		[]any{"a", "0x01"},
	}, decoded.Args())

	method, argsStr, err := decoded.String()
	require.NoError(t, err)
	require.Equal(t, "configure", method)
	require.Contains(t, argsStr, `"arg0": "set_config"`)
	require.Contains(t, argsStr, `"arg1": "1000"`)
}

func TestDecoder_Decode_NoContractType(t *testing.T) {
	t.Parallel()

	target := testContractID(t, 141)

	tx, err := stellar.NewTransaction(target, "accept_ownership", nil, "", nil)
	require.NoError(t, err)

	decoded, err := stellar.NewDecoder().Decode(tx, "")
	require.NoError(t, err)

	require.Equal(t, "accept_ownership", decoded.MethodName())
	require.Empty(t, decoded.Keys())
	require.Empty(t, decoded.Args())

	method, argsStr, err := decoded.String()
	require.NoError(t, err)
	require.Equal(t, "accept_ownership", method)
	require.Equal(t, "{}", argsStr)
}

func TestDecoder_Decode_EmptyData(t *testing.T) {
	t.Parallel()

	_, err := stellar.NewDecoder().Decode(types.Transaction{To: testContractID(t, 142)}, "")
	require.ErrorContains(t, err, "no invoke payload")
}

func TestDecoder_Decode_InvalidData(t *testing.T) {
	t.Parallel()

	tx := types.Transaction{
		To:   testContractID(t, 143),
		Data: []byte{0xff, 0xff, 0xff},
	}

	_, err := stellar.NewDecoder().Decode(tx, "")
	require.ErrorContains(t, err, "decode stellar invoke payload")
}

func TestDecoder_Decode_InterfaceContractIgnored(t *testing.T) {
	t.Parallel()

	target := testContractID(t, 144)

	// The same transaction must decode identically whatever string the
	// caller passes as contractInterfaces: Soroban invocations are
	// self-describing.
	tx, err := stellar.NewTransaction(target, "get_op_count", nil, "MCMS", nil)
	require.NoError(t, err)

	withInterfaces, err := stellar.NewDecoder().Decode(tx, "SomeRegistry")
	require.NoError(t, err)

	withoutInterfaces, err := stellar.NewDecoder().Decode(tx, "")
	require.NoError(t, err)

	require.Equal(t, withInterfaces.MethodName(), withoutInterfaces.MethodName())
	require.Equal(t, withInterfaces.Args(), withoutInterfaces.Args())
}
