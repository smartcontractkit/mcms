package types //nolint:revive,nolintlint // allow pkg name 'types'

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSignatureFromBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    []byte
		want    Signature
		wantErr string
	}{
		{
			name: "success",
			give: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0xfe, 0xdc, 0xba, 0x09, 0x87, 0x65, 0x43, 0x21,
				0x1b,
			},
			want: Signature{
				R: common.HexToHash("0x1234567890abcdef"),
				S: common.HexToHash("0xfedcba0987654321"),
				V: 27,
			},
		},
		{
			name:    "failure: invalid length",
			give:    []byte{0x00},
			wantErr: "invalid signature length: 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewSignatureFromBytes(tt.give)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestToBytes(t *testing.T) {
	t.Parallel()

	sig := Signature{
		R: common.HexToHash("0x1234567890abcdef"),
		S: common.HexToHash("0xfedcba0987654321"),
		V: 27,
	}

	want := []byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xfe, 0xdc, 0xba, 0x09, 0x87, 0x65, 0x43, 0x21,
		0x1b,
	}

	got := sig.ToBytes()
	assert.Equal(t, want, got)
}

func TestRecover(t *testing.T) {
	t.Parallel()

	// Private key to use for signing
	pk, err := crypto.GenerateKey()
	require.NoError(t, err)

	// Hash to sign
	hash := common.HexToHash("0xabcdef1234567890")
	sigBytes, err := crypto.Sign(hash.Bytes(), pk)
	require.NoError(t, err)

	// Signature object for the hash
	sig, err := NewSignatureFromBytes(sigBytes)
	require.NoError(t, err)

	tests := []struct {
		name          string
		giveSignature Signature
		giveHash      common.Hash
		want          common.Address
		wantErr       string
	}{
		{
			name:          "success",
			giveSignature: sig,
			giveHash:      hash,
			want:          crypto.PubkeyToAddress(pk.PublicKey),
		},
		{
			name: "success: adjusts v when larger than 1",
			giveSignature: Signature{ // Random values here will cause a failure
				R: sig.R,
				S: sig.S,
				V: sig.V + 27,
			},
			giveHash: hash,
			want:     crypto.PubkeyToAddress(pk.PublicKey),
		},
		{
			name: "failure: could not recover",
			giveSignature: Signature{ // Random values here will cause a failure
				R: common.HexToHash("0x0"),
				S: common.HexToHash("0xf"),
				V: 1,
			},
			giveHash: hash,
			wantErr:  "failed to recover public key: recovery failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.giveSignature.Recover(tt.giveHash)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
