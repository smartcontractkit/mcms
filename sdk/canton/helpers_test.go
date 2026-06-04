package canton

import (
	"testing"

	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
	"github.com/stretchr/testify/require"
)

func TestParseTemplateIDFromString(t *testing.T) {
	t.Parallel()

	pkg, mod, ent, err := ParseTemplateIDFromString("#pkg:Module:Entity")
	require.NoError(t, err)
	require.Equal(t, "#pkg", pkg)
	require.Equal(t, "Module", mod)
	require.Equal(t, "Entity", ent)

	_, _, ent, err := ParseTemplateIDFromString("pkg:Module:Entity")
	_ = ent
	require.Error(t, err)
}

func TestFormatTemplateID(t *testing.T) {
	t.Parallel()

	require.Equal(t, "pkg:Module:Entity", FormatTemplateID(&apiv2.Identifier{
		PackageId:  "pkg",
		ModuleName: "Module",
		EntityName: "Entity",
	}))
	require.Empty(t, FormatTemplateID(nil))
}

func TestNormalizeTemplateKey(t *testing.T) {
	t.Parallel()

	require.Equal(t, "Module:Entity", NormalizeTemplateKey("#pkg:Module:Entity"))
	require.Equal(t, "short", NormalizeTemplateKey("short"))
}

func TestTransactionResultHash(t *testing.T) {
	t.Parallel()

	require.Equal(t, "cmd-1", transactionResultHash(nil, "cmd-1"))
	require.Equal(t, "cmd-1", transactionResultHash(&apiv2.Transaction{}, "cmd-1"))
	require.Equal(t, "0xdeadbeef", transactionResultHash(&apiv2.Transaction{
		ExternalTransactionHash: []byte{0xde, 0xad, 0xbe, 0xef},
	}, "cmd-1"))
}
