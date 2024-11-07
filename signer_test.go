package mcms

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

const testPrivateKeyHex = "b17c4c6a409cebce4b39977689180900d9009d5c55a57ff9fd9cb962b24ae99d"

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
