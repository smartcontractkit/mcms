package aptos

import (
	"math/big"
	"testing"
	"time"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-aptos/bindings/mcms"
	module_mcms "github.com/smartcontractkit/chainlink-aptos/bindings/mcms/mcms"
	module_mcms_account "github.com/smartcontractkit/chainlink-aptos/bindings/mcms/mcms_account"

	"github.com/smartcontractkit/mcms/types"
)

func TestDecoder(t *testing.T) {
	t.Parallel()

	t.Run("success - mcms_account::transfer_ownership", func(t *testing.T) {
		t.Parallel()
		functionInfo := module_mcms_account.FunctionInfo

		toAddr := Must(hexToAddress("0x31ecd2c5d71b042fd4f1276316ed64c1f7e795606891a929ccf985576ed06577"))

		mcmsC := mcms.Bind(aptos.AccountFour, nil)
		module, function, _, args, err := mcmsC.MCMSAccount().Encoder().TransferOwnership(toAddr)
		require.NoError(t, err)

		transaction, err := NewTransaction(
			module.PackageName,
			module.ModuleName,
			function,
			aptos.AccountThree,
			ArgsToData(args),
			"MCMS",
			nil,
		)
		require.NoError(t, err)

		want := &DecodedOperation{
			PackageName:  "mcms",
			ModuleName:   module.ModuleName,
			FunctionName: function,
			InputKeys:    []string{"to"},
			InputArgs:    []any{toAddr},
		}

		d := NewDecoder()
		got, err := d.Decode(transaction, functionInfo)

		require.NoError(t, err)
		require.Equal(t, want, got)
	})
	t.Run("success - mcms::set_root", func(t *testing.T) {
		t.Parallel()
		functionInfo := module_mcms.FunctionInfo

		role := byte(1)
		root := common.HexToHash("0x61cf64140590b40dd677d1030320b73e2c0f0c807b42b1dd85c178dc1c72ab20").Bytes()
		validUntil := uint64(time.Now().Unix()) //nolint:gosec
		chainIDBig := big.NewInt(11572)
		multisigAddress := aptos.AccountFour
		preOpCount := uint64(12)
		postOpCount := uint64(17)
		overridePreviousRoot := true
		metadataProof := [][]byte{{0x12, 0x34}, {0x56, 0x78}}
		signatures := [][]byte{common.HexToHash("0x64f35785dd97d5f11d8be461445ee34d1e2fa58b9343a6bc2e030e92b9085296").Bytes(), common.HexToHash("0x8518a14816bb91bff897661ee6a1f07bea0a29cd6c53a7dbc5066e08b1ff2471").Bytes(), common.HexToHash("0xfa2f614181ed154a688868e9010168554ccabb968e7b67a47775ec6f1b692bfe").Bytes()}

		mcmsC := mcms.Bind(multisigAddress, nil)
		module, function, _, args, err := mcmsC.MCMS().Encoder().SetRoot(
			role,
			root,
			validUntil,
			chainIDBig,
			multisigAddress,
			preOpCount,
			postOpCount,
			overridePreviousRoot,
			metadataProof,
			signatures,
		)
		require.NoError(t, err)

		transaction, err := NewTransaction(
			module.PackageName,
			module.ModuleName,
			function,
			aptos.AccountThree,
			ArgsToData(args),
			"MCMS",
			nil,
		)
		require.NoError(t, err)

		want := &DecodedOperation{
			PackageName:  "mcms",
			ModuleName:   module.ModuleName,
			FunctionName: function,
			InputKeys:    []string{"role", "root", "valid_until", "chain_id", "multisig_addr", "pre_op_count", "post_op_count", "override_previous_root", "metadata_proof", "signatures"},
			InputArgs:    []any{role, root, validUntil, chainIDBig, multisigAddress, preOpCount, postOpCount, overridePreviousRoot, metadataProof, signatures},
		}

		d := NewDecoder()
		got, err := d.Decode(transaction, functionInfo)

		require.NoError(t, err)
		require.Equal(t, want, got)
	})
	t.Run("failure - invalid contractInterfaces", func(t *testing.T) {
		t.Parallel()
		functionInfo := "asdf"
		_, err := NewDecoder().Decode(types.Transaction{}, functionInfo)
		require.ErrorContains(t, err, "parse function info")
	})
	t.Run("failure - invalid additional fields", func(t *testing.T) {
		t.Parallel()
		functionInfo := module_mcms.FunctionInfo
		_, err := NewDecoder().Decode(types.Transaction{AdditionalFields: []byte("invalidJson")}, functionInfo)
		require.ErrorContains(t, err, "unmarshal additional fields")
	})
}
