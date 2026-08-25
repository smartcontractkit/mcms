package stellar

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/chainlink-stellar/bindings/contracts/mcms"
	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"
	"github.com/stellar/go-stellar-sdk/xdr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/stellar/mocks"
)

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

	rootTuple := xdr.ScVec{
		scval.Bytes32ToScVal(root),
		scval.Uint32ToScVal(123),
	}
	rootTuplePtr := &rootTuple
	rootValue := xdr.ScVal{
		Type: xdr.ScValTypeScvVec,
		Vec:  &rootTuplePtr,
	}

	metadataValue, err := (mcms.StellarRootMetadata{
		ConfigVersion:   4,
		EncodingVersion: 1,
		Multisig:        contractID,
		PreOpCount:      7,
		PostOpCount:     8,
	}).ToScVal()
	require.NoError(t, err)

	owner := contractID

	invoker := mocks.NewInvoker(t)

	invoker.
		On(
			"SimulateContract",
			mock.Anything,
			contractID,
			"get_config",
			mock.Anything,
		).
		Return(&configValue, nil).
		Once()

	invoker.
		On(
			"SimulateContract",
			mock.Anything,
			contractID,
			"get_op_count",
			mock.Anything,
		).
		Return(new(scval.Uint64ToScVal(7)), nil).
		Once()

	invoker.
		On(
			"SimulateContract",
			mock.Anything,
			contractID,
			"get_root",
			mock.Anything,
		).
		Return(&rootValue, nil).
		Once()

	invoker.
		On(
			"SimulateContract",
			mock.Anything,
			contractID,
			"get_root_metadata",
			mock.Anything,
		).
		Return(&metadataValue, nil).
		Once()

	invoker.
		On(
			"SimulateContract",
			mock.Anything,
			contractID,
			"owner",
			mock.Anything,
		).
		Return(new(scval.OptionalAddressToScVal(&owner)), nil).
		Once()

	invoker.
		On(
			"SimulateContract",
			mock.Anything,
			contractID,
			"get_pending_owner",
			mock.Anything,
		).
		Return(new(scval.OptionalAddressToScVal(nil)), nil).
		Once()

	inspector := NewInspectorFromInvoker(invoker)

	config, err := inspector.GetConfig(t.Context(), contractID)
	require.NoError(t, err)
	require.Equal(t, uint8(1), config.Quorum)
	require.Equal(
		t,
		[]common.Address{common.HexToAddress("0x1234")},
		config.Signers,
	)

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
	require.JSONEq(
		t,
		`{"configVersion":4,"encodingVersion":1}`,
		string(metadata.AdditionalFields),
	)

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

	_, err = toSDKConfig(&mcms.Config{
		GroupQuorums: quorums,
	})
	require.ErrorContains(t, err, "no signers")

	var malformed [32]byte
	malformed[0] = 1
	_, err = toSDKConfig(&mcms.Config{
		GroupQuorums: quorums,
		Signers: []mcms.Signer{
			{
				Addr:  malformed,
				Group: 0,
			},
		},
	})
	require.ErrorContains(t, err, "not a padded EVM address")
}
