package mcms

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

const testPrivateKeyHex = "b17c4c6a409cebce4b39977689180900d9009d5c55a57ff9fd9cb962b24ae99d"

func TestPrivateKeySigner_Sign(t *testing.T) {
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
			want: "286961047426c8463f85f43ddd19d7071ed2ecdab1522f654e4e3ff92cfc9e260fd18486a40e0379aec9ba50230c31b353497305c84c55793141c9df654f99a900",
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

func TestPrivateKeySigner_GetAddress(t *testing.T) {
	t.Parallel()

	privKey, err := crypto.HexToECDSA(testPrivateKeyHex)
	require.NoError(t, err)

	signer := NewPrivateKeySigner(privKey)
	addr, err := signer.GetAddress()

	require.NoError(t, err)
	want := common.HexToAddress("0xFe6d23D3C194bA84C035be35ad82775ddf0BFf4e")
	require.Equal(t, want, addr)
}
