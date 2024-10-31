package safecast

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_IntToUint8(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    int
		want    uint8
		wantErr bool
	}{
		{name: "Valid int within range", give: 42, want: 42},
		{name: "Negative int", give: -1, wantErr: true},
		{name: "Int exceeds uint32 max value", give: int(math.MaxUint8 + 1), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := IntToUint8(tt.give)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_Uint64ToUint8(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    uint64
		want    uint8
		wantErr bool
	}{
		{name: "Valid uint64 within range", give: 42, want: 42},
		{name: "Uint64 exceeds uint8 max value", give: uint64(math.MaxUint8 + 1), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Uint64ToUint8(tt.give)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_UInt64ToUint32(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    uint64
		want    uint32
		wantErr bool
	}{
		{name: "Valid int64 within range", give: 42, want: 42},
		{name: "Int64 exceeds uint32 max value", give: math.MaxUint32 + 1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Uint64ToUint32(tt.give)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
