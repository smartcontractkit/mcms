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

func Test_IntToUint32(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    int
		want    uint32
		wantErr bool
	}{
		{name: "Valid int within range", give: 42, want: 42},
		{name: "Negative int", give: -1, wantErr: true},
		{name: "Int exceeds uint32 max value", give: int(math.MaxUint32 + 1), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := IntToUint32(tt.give)

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

func Test_Int64ToUint32(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    int64
		want    uint32
		wantErr bool
	}{
		{name: "Valid int64 within range", give: 42, want: 42},
		{name: "Int64 exceeds uint32 max value", give: math.MaxUint32 + 1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Int64ToUint32(tt.give)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_Uint64ToInt64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    uint64
		want    int64
		wantErr bool
	}{
		{name: "Valid uint64 within range", give: 42, want: 42},
		{name: "Zero value", give: 0, want: 0},
		{name: "Maximum valid value", give: math.MaxInt64, want: math.MaxInt64},
		{name: "Uint64 exceeds int64 max value", give: math.MaxInt64 + 1, wantErr: true},
		{name: "Maximum uint64 value", give: math.MaxUint64, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Uint64ToInt64(tt.give)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "exceeds int64 range")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_Int64ToUint64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    int64
		want    uint64
		wantErr bool
	}{
		{name: "Valid positive int64", give: 42, want: 42},
		{name: "Zero value", give: 0, want: 0},
		{name: "Maximum int64 value", give: math.MaxInt64, want: math.MaxInt64},
		{name: "Negative int64", give: -1, wantErr: true},
		{name: "Large negative int64", give: math.MinInt64, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Int64ToUint64(tt.give)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "is negative, cannot convert to uint64")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
