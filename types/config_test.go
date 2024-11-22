package types

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define signers for use all tests
var (
	signer1 = common.HexToAddress("0x1")
	signer2 = common.HexToAddress("0x2")
	signer3 = common.HexToAddress("0x3")
	signer4 = common.HexToAddress("0x4")
)

func Test_NewConfig(t *testing.T) {
	t.Parallel()

	var (
		signers      = []common.Address{signer1, signer2}
		groupSigners = []Config{
			{Quorum: 1, Signers: []common.Address{signer3}},
		}
	)

	// Valid configuration
	got, err := NewConfig(1, signers, groupSigners)
	require.NoError(t, err)

	assert.NotNil(t, got)
	assert.Equal(t, uint8(1), got.Quorum)
	assert.Equal(t, signers, got.Signers)
	assert.Equal(t, groupSigners, got.GroupSigners)

	// Invalid configuration
	got, err = NewConfig(0, signers, groupSigners)
	require.Error(t, err)
	assert.Equal(t, Config{}, got)
}

func Test_Config_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    Config
		wantErr string
	}{
		{
			name: "success: valid configuration",
			give: Config{
				Quorum:       2,
				Signers:      []common.Address{signer1, signer2},
				GroupSigners: []Config{},
			},
		},
		{
			name: "failure: invalid quorum of 0",
			give: Config{
				Quorum: 0,
			},
			wantErr: "invalid MCMS config: Quorum must be greater than 0",
		},
		{
			name: "failure: no signers or groups",
			give: Config{
				Quorum:       2,
				Signers:      []common.Address{},
				GroupSigners: []Config{},
			},
			wantErr: "invalid MCMS config: Config must have at least one signer or group",
		},
		{
			name: "failure: quorum greater than the sum of number of signers and groups",
			give: Config{
				Quorum:  3,
				Signers: []common.Address{signer1},
				GroupSigners: []Config{
					{
						Quorum:       1,
						Signers:      []common.Address{signer2},
						GroupSigners: []Config{},
					},
				},
			},
			wantErr: "invalid MCMS config: Quorum must be less than or equal to the number of signers and groups",
		},
		{
			name: "failure: invalid group signer",
			give: Config{
				Quorum:  2,
				Signers: []common.Address{signer1, signer2},
				GroupSigners: []Config{
					{Quorum: 0},
				},
			},
			wantErr: "invalid MCMS config: Quorum must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.give.Validate()

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_Config_Equals(t *testing.T) {
	t.Parallel()

	// The config that is being matched against. This is compared against the give field in
	// every test case.
	var config = Config{
		Quorum:  1,
		Signers: []common.Address{signer1, signer2},
		GroupSigners: []Config{
			{
				Quorum:  1,
				Signers: []common.Address{signer3},
			},
		},
	}

	tests := []struct {
		name string
		give Config
		want bool
	}{
		{
			name: "success: equal configurations",
			give: config,
			want: true,
		},
		{
			name: "failure: mismatching quorum",
			give: Config{
				Quorum: 1,
			},
			want: false,
		},
		{
			name: "failure: mismatching signers length",
			give: Config{
				Quorum:  2,
				Signers: []common.Address{signer1},
			},
			want: false,
		},
		{
			name: "failure: mismatching signers",
			give: Config{
				Quorum:  2,
				Signers: []common.Address{signer1, signer3}, // Signer 3 instead of 2
			},
			want: false,
		},
		{
			name: "failure: mismatching group signers length",
			give: Config{
				Quorum:       1,
				Signers:      []common.Address{signer1, signer2},
				GroupSigners: []Config{}, // No group signers but there should be 1
			},
			want: false,
		},
		{
			name: "failure: mismatching group signers",
			give: Config{
				Quorum:  1,
				Signers: []common.Address{signer1, signer2},
				GroupSigners: []Config{
					{
						Quorum:  1,
						Signers: []common.Address{signer3, signer4}, // Additional signer4
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, config.Equals(&tt.give))
		})
	}
}
func Test_Config_GetAllSigners(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		give Config
		want []common.Address
	}{
		{
			name: "success: single level signers",
			give: Config{
				Quorum:  1,
				Signers: []common.Address{signer1, signer2},
			},
			want: []common.Address{signer1, signer2},
		},
		{
			name: "success: nested group signers",
			give: Config{
				Quorum:  1,
				Signers: []common.Address{signer1},
				GroupSigners: []Config{
					{
						Quorum:  1,
						Signers: []common.Address{signer2},
						GroupSigners: []Config{
							{
								Quorum:  1,
								Signers: []common.Address{signer3, signer4},
							},
						},
					},
				},
			},
			want: []common.Address{signer1, signer2, signer3, signer4},
		},
		{
			name: "success: no signers",
			give: Config{
				Quorum:       1,
				Signers:      []common.Address{},
				GroupSigners: []Config{},
			},
			want: []common.Address{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.give.GetAllSigners()
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func Test_Config_CanSetRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		config           Config
		recoveredSigners []common.Address
		want             bool
		wantErr          string
	}{
		{
			name: "success: single level signers reach consensus",
			config: Config{
				Quorum:  2,
				Signers: []common.Address{signer1, signer2},
			},
			recoveredSigners: []common.Address{signer1, signer2},
			want:             true,
		},
		{
			name: "success: nested group signers reach consensus",
			config: Config{
				Quorum:  2,
				Signers: []common.Address{signer1},
				GroupSigners: []Config{
					{
						Quorum:  2,
						Signers: []common.Address{signer2, signer3},
					},
				},
			},
			recoveredSigners: []common.Address{signer1, signer2, signer3},
			want:             true,
		},
		{
			name: "failure: single level signers do not reach consensus",
			config: Config{
				Quorum:  2,
				Signers: []common.Address{signer1, signer2},
			},
			recoveredSigners: []common.Address{signer1},
			want:             false,
		},
		{
			name: "failure: nested group signers do not reach consensus",
			config: Config{
				Quorum:  2,
				Signers: []common.Address{signer1},
				GroupSigners: []Config{
					{
						Quorum:  2,
						Signers: []common.Address{signer2},
						GroupSigners: []Config{
							{
								Quorum:  2,
								Signers: []common.Address{signer3, signer4},
							},
						},
					},
				},
			},
			recoveredSigners: []common.Address{signer1, signer2, signer3},
			want:             false,
		},
		{
			name: "failure: invalid recovered signer",
			config: Config{
				Quorum:  1,
				Signers: []common.Address{signer1},
			},
			recoveredSigners: []common.Address{signer4},
			want:             false,
			wantErr:          "recovered signer 0x0000000000000000000000000000000000000004 is not a valid signer in the MCMS proposal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.config.CanSetRoot(tt.recoveredSigners)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_unorderedArrayEquals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a, b []common.Address
		want bool
	}{
		{
			name: "success: equal arrays in same order",
			a:    []common.Address{signer1, signer2, signer3},
			b:    []common.Address{signer1, signer2, signer3},
			want: true,
		},
		{
			name: "success: equal arrays in different order",
			a:    []common.Address{signer1, signer2, signer3},
			b:    []common.Address{signer3, signer1, signer2},
			want: true,
		},
		{
			name: "failure: different lengths",
			a:    []common.Address{signer1, signer2},
			b:    []common.Address{signer1, signer2, signer3},
			want: false,
		},
		{
			name: "failure: different elements",
			a:    []common.Address{signer1, signer2, signer3},
			b:    []common.Address{signer1, signer2, signer4},
			want: false,
		},
		{
			name: "success: both empty arrays",
			a:    []common.Address{},
			b:    []common.Address{},
			want: true,
		},
		{
			name: "failure: one empty array",
			a:    []common.Address{signer1},
			b:    []common.Address{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := unorderedArrayEquals(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}
