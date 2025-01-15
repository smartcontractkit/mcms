package solana

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
)

var testAccount = solana.MustPublicKeyFromBase58("4HeqEoSyfYpeC2goFLj9eHgkxV33mR5G7JYAbRsN14uQ")
var testAccount2 = solana.MustPublicKeyFromBase58("HzwwijybRfFkQvtLyVNzTr7EVpQZkoStdr47mvaYMD4y")

func TestNewEncoder(t *testing.T) {
	t.Parallel()

	encoder := NewEncoder(testChainSelector, 5, true)

	require.NotNil(t, encoder)
	require.Equal(t, testChainSelector, encoder.ChainSelector)
	require.Equal(t, uint64(5), encoder.TxCount)
	require.True(t, encoder.OverridePreviousRoot)
}

func TestEncoder_HashOperation(t *testing.T) {
	t.Parallel()
	solanaTx, err := NewTransaction(
		testAccount.String(),
		[]byte("test data"),
		[]*solana.AccountMeta{{PublicKey: testAccount2, IsSigner: true, IsWritable: false}},
		"unit-tests",
		[]string{"test"},
	)
	require.NoError(t, err)
	tests := []struct {
		name     string
		txCount  uint64
		override bool
		opCount  uint32
		metadata types.ChainMetadata
		op       types.Operation
		want     common.Hash
		wantErr  string
	}{
		{
			name:     "success: txcount=3 override=true op-count=2 starting-op-count=123",
			txCount:  3,
			override: true,
			opCount:  2,
			metadata: types.ChainMetadata{StartingOpCount: 123, MCMAddress: ContractAddress(testMCMSProgramID, testPDASeed)},
			op: types.Operation{
				ChainSelector: testChainSelector,
				Transaction: types.Transaction{
					To:   "4HeqEoSyfYpeC2goFLj9eHgkxV33mR5G7JYAbRsN14uQ",
					Data: []byte("0x012345789abcdef"),
					AdditionalFields: []byte(`{
						"remainingAccounts": [{
							"publicKey":  "EDYUM4CJzrCj5fz4PGGWDhiBTxKfzX9mtWNn8YLnNYUs",
							"isSigner":   true,
							"isWritable": true
						}]
					}`),
				},
			},
			want: common.HexToHash("0xb8378ac2ea11c40643c828a93db1d1fb1829d19b261ea4d5f5119e28f2ad03c7"),
		},
		{
			name:     "success: solana encoded tx",
			txCount:  3,
			override: true,
			opCount:  1,
			metadata: types.ChainMetadata{StartingOpCount: 123, MCMAddress: ContractAddress(testMCMSProgramID, testPDASeed)},
			op: types.Operation{
				ChainSelector: testChainSelector,
				Transaction:   solanaTx,
			},
			want: common.HexToHash("0xf41c25f49165165af155c093688ba869707285742fe9c8d089ffb4ccf185e332"),
		},
		{
			name:     "success: txcount=1 override=false op-count=1 starting-op-count=0",
			txCount:  1,
			override: false,
			opCount:  1,
			metadata: types.ChainMetadata{StartingOpCount: 0, MCMAddress: ContractAddress(testMCMSProgramID, testPDASeed)},
			op: types.Operation{
				ChainSelector: testChainSelector,
				Transaction: types.Transaction{
					To:   "4HeqEoSyfYpeC2goFLj9eHgkxV33mR5G7JYAbRsN14uQ",
					Data: []byte{},
					AdditionalFields: []byte(`{
						"accounts": [{
							"publicKey":  "EDYUM4CJzrCj5fz4PGGWDhiBTxKfzX9mtWNn8YLnNYUs",
							"isSigner":   false,
							"isWritable": false
						}]
					}`),
				},
			},
			want: common.HexToHash("0x59b6ff3f8235ffa1952da10f010bbd29c6dd715c00d8a52e2bade498423ea699"),
		},
		{
			name:     "success: no remaining accounts",
			txCount:  1,
			override: false,
			opCount:  1,
			metadata: types.ChainMetadata{StartingOpCount: 0, MCMAddress: ContractAddress(testMCMSProgramID, testPDASeed)},
			op: types.Operation{
				ChainSelector: testChainSelector,
				Transaction: types.Transaction{
					To:   "4HeqEoSyfYpeC2goFLj9eHgkxV33mR5G7JYAbRsN14uQ",
					Data: []byte{},
				},
			},
			want: common.HexToHash("0x3031d39c9c1333d30392b978b08a16ebac8de2ee2b1b427b2a85b669f47649e2"),
		},
		{
			name:     "failure: invalid address",
			txCount:  1,
			override: false,
			opCount:  1,
			metadata: types.ChainMetadata{StartingOpCount: 0, MCMAddress: "invalid"},
			op:       types.Operation{},
			wantErr:  "unable to parse solana contract address: invalid solana contract address format: \"invalid\"",
		},
		{
			name:     "failure: invalid additional fields",
			txCount:  1,
			override: false,
			opCount:  1,
			metadata: types.ChainMetadata{StartingOpCount: 0, MCMAddress: ContractAddress(testMCMSProgramID, testPDASeed)},
			op: types.Operation{
				ChainSelector: testChainSelector,
				Transaction: types.Transaction{
					To:               "4HeqEoSyfYpeC2goFLj9eHgkxV33mR5G7JYAbRsN14uQ",
					Data:             []byte{},
					AdditionalFields: []byte(`invalid`),
				},
			},
			wantErr: "unable to unmarshal additional fields: invalid character 'i' looking for beginning of value",
		},
		{
			name:     "failure: invalid 'to' address",
			txCount:  1,
			override: false,
			opCount:  1,
			metadata: types.ChainMetadata{StartingOpCount: 0, MCMAddress: ContractAddress(testMCMSProgramID, testPDASeed)},
			op: types.Operation{
				ChainSelector: testChainSelector,
				Transaction:   types.Transaction{To: "invalid"},
			},
			wantErr: "unable to get hash from base58 To address: invalid solana contract address format: \"invalid\"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encoder := newTestEncoder(t, tt.txCount, tt.override)
			got, err := encoder.HashOperation(tt.opCount, tt.metadata, tt.op)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestEncoder_HashMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		txCount  uint64
		override bool
		metadata types.ChainMetadata
		want     common.Hash
		wantErr  string
	}{
		{
			name:     "success: txcount=2 override=true starting-op-count=123",
			txCount:  2,
			override: true,
			metadata: types.ChainMetadata{StartingOpCount: 123, MCMAddress: ContractAddress(testMCMSProgramID, testPDASeed)},
			want:     common.HexToHash("0xceb06356f4b7718cdf9b585ba3725a0f4670742d7c367b4ec87b9938c7f6412a"),
		},
		{
			name:     "success: test case to test matching chainlink-ccip implementation",
			txCount:  5,
			override: true,
			metadata: types.ChainMetadata{StartingOpCount: 10, MCMAddress: ContractAddress(testMCMSProgramID, testPDASeed)},
			want:     common.HexToHash("0xaea0d6d8a1f2ed42a6a54473af0a0dc9dcc73442841b5e16c0872f7ef7cadabd"),
		},
		{
			name:     "success: txcount=0 override=false starting-op-count=0",
			txCount:  0,
			override: false,
			metadata: types.ChainMetadata{StartingOpCount: 0, MCMAddress: ContractAddress(testMCMSProgramID, testPDASeed)},
			want:     common.HexToHash("0xa6ce0700aa2f33b3ee31350d0fc8ef88fbaae48c7d3887ddb7a840a3d9bfd166"),
		},
		{
			name:     "failure: invalid mcm address",
			metadata: types.ChainMetadata{StartingOpCount: 123, MCMAddress: "invalid"},
			wantErr:  "unable to parse solana contract address: invalid solana contract address format: \"invalid\"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encoder := newTestEncoder(t, tt.txCount, tt.override)
			got, err := encoder.HashMetadata(tt.metadata)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

// ----- helpers -----

func newTestEncoder(t *testing.T, txCount uint64, overridePreviousRoot bool) *Encoder {
	t.Helper()
	return NewEncoder(testChainSelector, txCount, overridePreviousRoot)
}
