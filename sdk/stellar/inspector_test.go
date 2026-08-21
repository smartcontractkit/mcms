package stellar

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/chainlink-stellar/bindings/contracts/mcms"
	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"
	protocolrpc "github.com/stellar/go-stellar-sdk/protocols/rpc"
	"github.com/stellar/go-stellar-sdk/xdr"
	"github.com/stretchr/testify/require"
)

type inspectorInvoker struct {
	values map[string]xdr.ScVal
}

func (i inspectorInvoker) InvokeContract(context.Context, string, string, []xdr.ScVal) (*xdr.ScVal, error) {
	return nil, fmt.Errorf("unexpected write")
}

func (i inspectorInvoker) SimulateContract(_ context.Context, _ string, function string, _ []xdr.ScVal) (*xdr.ScVal, error) {
	value, ok := i.values[function]
	if !ok {
		return nil, fmt.Errorf("unexpected simulation %s", function)
	}
	return &value, nil
}

func (i inspectorInvoker) GetEvents(context.Context, string, uint32, []string) ([]protocolrpc.EventInfo, error) {
	return nil, nil
}

func TestInspectorReadsCurrentMCMSState(t *testing.T) {
	t.Parallel()
	const contractID = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"

	var signer [32]byte
	copy(signer[12:], common.HexToAddress("0x1234").Bytes())
	var quorums [32]byte
	quorums[0] = 1
	configValue, err := (mcms.Config{
		Signers:      []mcms.Signer{{Addr: signer, Group: 0, Index: 0}},
		GroupQuorums: quorums,
	}).ToScVal()
	require.NoError(t, err)

	var root [32]byte
	root[0] = 9
	rootTuple := xdr.ScVec{scval.Bytes32ToScVal(root), scval.Uint32ToScVal(123)}
	rootTuplePtr := &rootTuple
	rootValue := xdr.ScVal{Type: xdr.ScValTypeScvVec, Vec: &rootTuplePtr}

	metadataValue, err := (mcms.StellarRootMetadata{
		ConfigVersion:   4,
		EncodingVersion: 1,
		Multisig:        contractID,
		PreOpCount:      7,
		PostOpCount:     8,
	}).ToScVal()
	require.NoError(t, err)
	owner := contractID

	inspector := NewInspectorFromInvoker(inspectorInvoker{values: map[string]xdr.ScVal{
		"get_config":        configValue,
		"get_op_count":      scval.Uint64ToScVal(7),
		"get_root":          rootValue,
		"get_root_metadata": metadataValue,
		"owner":             scval.OptionalAddressToScVal(&owner),
		"get_pending_owner": scval.OptionalAddressToScVal(nil),
	}})

	config, err := inspector.GetConfig(t.Context(), contractID)
	require.NoError(t, err)
	require.Equal(t, uint8(1), config.Quorum)
	require.Equal(t, []common.Address{common.HexToAddress("0x1234")}, config.Signers)

	opCount, err := inspector.GetOpCount(t.Context(), contractID)
	require.NoError(t, err)
	require.Equal(t, uint64(7), opCount)

	actualRoot, validUntil, err := inspector.GetRoot(t.Context(), contractID)
	require.NoError(t, err)
	require.Equal(t, common.Hash(root), actualRoot)
	require.Equal(t, uint32(123), validUntil)

	metadata, err := inspector.GetRootMetadata(t.Context(), contractID)
	require.NoError(t, err)
	require.Equal(t, uint64(7), metadata.StartingOpCount)
	require.Equal(t, contractID, metadata.MCMAddress)
	require.JSONEq(t, `{"configVersion":4,"encodingVersion":1}`, string(metadata.AdditionalFields))

	actualOwner, err := inspector.GetOwner(t.Context(), contractID)
	require.NoError(t, err)
	require.Equal(t, &owner, actualOwner)
	pendingOwner, err := inspector.GetPendingOwner(t.Context(), contractID)
	require.NoError(t, err)
	require.Nil(t, pendingOwner)
}

func TestToSDKConfigReturnsConversionErrors(t *testing.T) {
	t.Parallel()
	_, err := toSDKConfig(nil)
	require.Error(t, err)

	var quorums [32]byte
	quorums[0] = 1
	_, err = toSDKConfig(&mcms.Config{GroupQuorums: quorums})
	require.ErrorContains(t, err, "no signers")

	var malformed [32]byte
	malformed[0] = 1
	_, err = toSDKConfig(&mcms.Config{
		GroupQuorums: quorums,
		Signers:      []mcms.Signer{{Addr: malformed, Group: 0}},
	})
	require.ErrorContains(t, err, "not a padded EVM address")
}
