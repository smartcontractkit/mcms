package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewDuration(t *testing.T) {
	t.Parallel()

	d, err := time.ParseDuration("1h")
	require.NoError(t, err)

	assert.Equal(t, Duration{Duration: d}, NewDuration(d))
}

func Test_ParseDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    string
		want    Duration
		wantErr string
	}{
		{
			name: "success",
			give: "1h",
			want: MustParseDuration("1h"),
		},
		{
			name:    "invalid duration string",
			give:    "a",
			wantErr: "time: invalid duration \"a\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual, err := ParseDuration(tt.give)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, actual)
			}
		})
	}
}

func Test_MustParseDuration(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		d, err := time.ParseDuration("1h")
		require.NoError(t, err)

		got := MustParseDuration("1h")
		assert.Equal(t, Duration{Duration: d}, got)
	})

	assert.Panics(t, func() {
		MustParseDuration("a")
	})
}

func Test_Duration_String(t *testing.T) {
	t.Parallel()

	d, err := time.ParseDuration("1h")
	require.NoError(t, err)

	assert.Equal(t, "1h0m0s", NewDuration(d).String())
}

func Test_Duration_MarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    Duration
		want    []byte
		wantErr string
	}{
		{
			name: "success",
			give: MustParseDuration("1h"),
			want: []byte(`"1h0m0s"`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.give.MarshalJSON()

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_Duration_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    []byte
		want    Duration
		wantErr string
	}{
		{
			name: "valid string",
			give: []byte(`"1h"`),
			want: MustParseDuration("1h"),
		},
		{
			name:    "invalid float64",
			give:    []byte(`1`),
			wantErr: "invalid duration type: float64",
		},
		{
			name:    "invalid time string",
			give:    []byte(`"a"`),
			wantErr: "time: invalid duration \"a\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got Duration
			err := got.UnmarshalJSON(tt.give)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
