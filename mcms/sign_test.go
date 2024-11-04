package mcms

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/mocks"
	"github.com/smartcontractkit/mcms/types"
)

const testPrivateKeyHex = "b17c4c6a409cebce4b39977689180900d9009d5c55a57ff9fd9cb962b24ae99d"

func Test_Sign(t *testing.T) {
	t.Parallel()

	privKey, err := crypto.HexToECDSA(testPrivateKeyHex)
	require.NoError(t, err)

	// Construct a proposal
	proposal := MCMSProposal{
		Version:              "1.0",
		Description:          "Grants RBACTimelock 'Proposer' Role to MCMS Contract",
		ValidUntil:           2004259681,
		Signatures:           []types.Signature{},
		OverridePreviousRoot: false,
		ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
			TestChain1: {
				StartingOpCount: 0,
				MCMAddress:      "0x01",
			},
		},
		Transactions: []types.ChainOperation{
			{
				ChainSelector: TestChain1,
				Operation: evm.NewEVMOperation(
					common.HexToAddress("0x02"),
					[]byte("0x0000000"), // Use some random data since it doesn't matter
					big.NewInt(0),
					"RBACTimelock",
					[]string{"RBACTimelock", "GrantRole"},
				),
			},
		},
	}

	tests := []struct {
		name    string
		give    MCMSProposal
		want    types.Signature
		wantErr string
	}{
		{
			name: "success: signs the proposal",
			give: proposal,
			want: types.Signature{
				R: common.HexToHash("0x859c780e5df453945171c96f271c16b5baeeb6eadfa790d4e4d32ee72607334b"),
				S: common.HexToHash("0x3fd6128a489e81ecce6192804ea26ceaf542ae11f20caae65e6b65662f882eb4"),
				V: 0,
			},
		},
		{
			name:    "failure: invalid proposal",
			give:    MCMSProposal{},
			wantErr: "invalid version: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			inspector := mocks.NewInspector(t)
			inspectors := map[types.ChainSelector]sdk.Inspector{
				TestChain1: inspector,
			}

			// Ensure that there are no signatures to being with
			require.Empty(t, tt.give.Signatures)

			signable, err := tt.give.Signable(true, inspectors)
			require.NoError(t, err)
			require.NotNil(t, signable)

			err = Sign(signable, NewPrivateKeySigner(privKey))

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Len(t, tt.give.Signatures, 1)
				require.Equal(t, tt.want, tt.give.Signatures[0])
			}
		})
	}
}

func Test_PrivateKeySigner_Sign(t *testing.T) {
	t.Parallel()

	privKey, err := crypto.HexToECDSA(testPrivateKeyHex)
	require.NoError(t, err)

	tests := []struct {
		name    string
		give    []byte
		want    string // Hex encoding of the signed payload
		wantErr string
	}{
		{
			name: "success: signs the proposal",
			give: []byte("0x000000000000000000000000000000"),
			want: "403c61c40165ad6f361d2e3f7d2ee9707c48006941838b702a31d6c2782b2e0527e8d93a7462955f1068ea72928959b3ea1be496a389528be5df5bb6b2c515d300",
		},
		{
			name:    "failure: invalid payload length",
			give:    []byte("0x0"),
			wantErr: "hash is required to be exactly 32 bytes (3)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewPrivateKeySigner(privKey).Sign(tt.give)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)

				want, err := hex.DecodeString(tt.want)
				require.NoError(t, err)
				require.Equal(t, want, got)
			}
		})
	}
}
