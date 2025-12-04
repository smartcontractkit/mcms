package safecast

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntToUint8(t *testing.T) {
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

func TestIntToUint32(t *testing.T) {
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

func TestUint64ToUint8(t *testing.T) {
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

func TestUInt64ToUint32(t *testing.T) {
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

func TestInt64ToUint32(t *testing.T) {
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

func TestUint64ToInt64(t *testing.T) {
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

func TestInt64ToUint64(t *testing.T) {
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

func TestFloat64ToUint64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    float64
		want    uint64
		wantErr bool
	}{
		{name: "Valid positive float64", give: 42.0, want: 42},
		{name: "Zero value", give: 0.0, want: 0},
		{name: "Large safe integer", give: 1<<53 - 1, want: 1<<53 - 1}, // Largest safe integer in float64
		{name: "Large valid integer", give: 1234567890.0, want: 1234567890},
		{name: "Negative float64", give: -1.0, wantErr: true},
		{name: "Large negative float64", give: -1234567890.0, wantErr: true},
		{name: "Float with fractional part", give: 42.5, wantErr: true},
		{name: "Small fractional part", give: 100.1, wantErr: true},
		{name: "Float exceeds uint64 max", give: float64(math.MaxUint64) * 2, wantErr: true},
		{name: "Infinity", give: math.Inf(1), wantErr: true},
		{name: "Negative infinity", give: math.Inf(-1), wantErr: true},
		{name: "NaN", give: math.NaN(), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Float64ToUint64(tt.give)

			if tt.wantErr {
				require.Error(t, err)
				// Check for specific error messages based on the test case
				if tt.give < 0 && !math.IsInf(tt.give, -1) && !math.IsNaN(tt.give) {
					assert.Contains(t, err.Error(), "is negative, cannot convert to uint64")
				} else if tt.give > math.MaxUint64 || math.IsInf(tt.give, 1) {
					assert.Contains(t, err.Error(), "exceeds uint64 range")
				} else if tt.give != math.Trunc(tt.give) && !math.IsNaN(tt.give) {
					assert.Contains(t, err.Error(), "has fractional part, cannot convert to uint64")
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
