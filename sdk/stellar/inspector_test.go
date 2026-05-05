package stellar

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	stellarmcms "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/mcms"
	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"
	protocolrpc "github.com/stellar/go-stellar-sdk/protocols/rpc"
	"github.com/stellar/go-stellar-sdk/xdr"

	"github.com/smartcontractkit/mcms/types"
)

func TestConfigTransformer_ToConfig_ToChainConfig_roundTrip(t *testing.T) {
	t.Parallel()

	tr := NewConfigTransformer()

	want := types.Config{
		Quorum:  1,
		Signers: []common.Address{{1}},
	}

	chainCfg, err := tr.ToChainConfig(want, nil)
	require.NoError(t, err)

	got, err := tr.ToConfig(chainCfg)
	require.NoError(t, err)

	require.True(t, got.Equals(&want))
}

type mockInvoker struct {
	cfg      *stellarmcms.Config
	root     [32]byte
	valid    uint32
	opCount  uint64
	rootMeta *stellarmcms.StellarRootMetadata
}

func (m *mockInvoker) InvokeContract(context.Context, string, string, []xdr.ScVal) (*xdr.ScVal, error) {
	return nil, invokerNotImplementedError{}
}

func (m *mockInvoker) GetEvents(context.Context, string, uint32, []string) ([]protocolrpc.EventInfo, error) {
	return nil, invokerNotImplementedError{}
}

func (m *mockInvoker) SimulateContract(_ context.Context, _ string, fn string, _ []xdr.ScVal) (*xdr.ScVal, error) {
	switch fn {
	case "get_config":
		v, err := m.cfg.ToScVal()
		if err != nil {
			return nil, err
		}

		return &v, nil

	case "get_root":
		v := scval.VecToScVal([]xdr.ScVal{
			scval.Bytes32ToScVal(m.root),
			scval.Uint32ToScVal(m.valid),
		})

		return &v, nil

	case "get_op_count":
		v := scval.Uint64ToScVal(m.opCount)

		return &v, nil

	case "get_root_metadata":
		v, err := m.rootMeta.ToScVal()
		if err != nil {
			return nil, err
		}

		return &v, nil

	default:
		return nil, invokerNotImplementedError{}
	}
}

type invokerNotImplementedError struct{}

func (invokerNotImplementedError) Error() string {
	return "mock invoker: not implemented"
}

func TestInspector_readsViaInvoker(t *testing.T) {
	t.Parallel()

	var paddedSigner [32]byte
	copy(paddedSigner[evmAddressABIWordLeadingZeroBytes:], []byte{0xab, 0xcd})

	cfg := &stellarmcms.Config{
		GroupQuorums: func() (out [32]byte) {
			out[0] = 1

			return out
		}(),
		GroupParents: [32]byte{},
		Signers: []stellarmcms.Signer{
			{Addr: paddedSigner, Group: 0, Index: 0},
		},
	}

	meta := &stellarmcms.StellarRootMetadata{
		PreOpCount:  7,
		PostOpCount: 8,
	}

	inv := &mockInvoker{
		cfg:      cfg,
		root:     [32]byte{9},
		valid:    42,
		opCount:  100,
		rootMeta: meta,
	}

	insp := NewInspector(inv)

	ctx := context.Background()

	const contractHex = "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"

	gotCfg, err := insp.GetConfig(ctx, contractHex)
	require.NoError(t, err)
	require.Equal(t, uint8(1), gotCfg.Quorum)
	require.Len(t, gotCfg.Signers, 1)
	require.Equal(t, common.Address{0xab, 0xcd}, gotCfg.Signers[0])

	opCount, err := insp.GetOpCount(ctx, contractHex)
	require.NoError(t, err)
	require.Equal(t, uint64(100), opCount)

	root, validUntil, err := insp.GetRoot(ctx, contractHex)
	require.NoError(t, err)
	require.Equal(t, common.Hash([32]byte{9}), root)
	require.Equal(t, uint32(42), validUntil)

	md, err := insp.GetRootMetadata(ctx, contractHex)
	require.NoError(t, err)
	require.Equal(t, uint64(7), md.StartingOpCount)
	require.Equal(t, contractHex, md.MCMAddress)
}
