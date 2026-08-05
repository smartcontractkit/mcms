package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChainMetadata_AllMCMs(t *testing.T) {
	t.Parallel()

	primary := ChainMetadata{StartingOpCount: 5, MCMAddress: "0xaaa"}
	additional := ChainMetadata{StartingOpCount: 2, MCMAddress: "0xbbb"}
	md := ChainMetadata{
		StartingOpCount: primary.StartingOpCount,
		MCMAddress:      primary.MCMAddress,
		AdditionalMCMs:  []ChainMetadata{additional},
	}

	all := md.AllMCMs()
	require.Len(t, all, 2)
	assert.Equal(t, primary.MCMAddress, all[0].MCMAddress)
	assert.Equal(t, additional.MCMAddress, all[1].MCMAddress)
	// The primary view must not carry the nested list
	assert.Empty(t, all[0].AdditionalMCMs)

	// Single-MCM chain: exactly one entry
	single := ChainMetadata{MCMAddress: "0xaaa"}
	require.Len(t, single.AllMCMs(), 1)
}

func TestChainMetadata_GetMCM(t *testing.T) {
	t.Parallel()

	md := ChainMetadata{
		StartingOpCount: 5,
		MCMAddress:      "0xaaa",
		AdditionalMCMs: []ChainMetadata{
			{StartingOpCount: 2, MCMAddress: "0xbbb"},
		},
	}

	tests := []struct {
		name      string
		give      string
		wantAddr  string
		wantCount uint64
		wantOK    bool
	}{
		{name: "empty resolves to primary", give: "", wantAddr: "0xaaa", wantCount: 5, wantOK: true},
		{name: "explicit primary", give: "0xaaa", wantAddr: "0xaaa", wantCount: 5, wantOK: true},
		{name: "additional instance", give: "0xbbb", wantAddr: "0xbbb", wantCount: 2, wantOK: true},
		{name: "unknown address", give: "0xccc", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := md.GetMCM(tt.give)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantAddr, got.MCMAddress)
				assert.Equal(t, tt.wantCount, got.StartingOpCount)
			}
		})
	}
}

func TestChainMetadata_ValidateMultiMCM(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    ChainMetadata
		wantErr string
	}{
		{
			name: "valid: single MCM",
			give: ChainMetadata{MCMAddress: "0xaaa"},
		},
		{
			name: "valid: primary plus additional",
			give: ChainMetadata{
				MCMAddress:     "0xaaa",
				AdditionalMCMs: []ChainMetadata{{MCMAddress: "0xbbb"}},
			},
		},
		{
			name: "failure: empty additional address",
			give: ChainMetadata{
				MCMAddress:     "0xaaa",
				AdditionalMCMs: []ChainMetadata{{MCMAddress: ""}},
			},
			wantErr: "additional MCM entries must have a non-empty MCMAddress",
		},
		{
			name: "failure: duplicate of primary",
			give: ChainMetadata{
				MCMAddress:     "0xaaa",
				AdditionalMCMs: []ChainMetadata{{MCMAddress: "0xaaa"}},
			},
			wantErr: `duplicate MCMAddress "0xaaa" in chain metadata`,
		},
		{
			name: "failure: duplicate additionals",
			give: ChainMetadata{
				MCMAddress: "0xaaa",
				AdditionalMCMs: []ChainMetadata{
					{MCMAddress: "0xbbb"},
					{MCMAddress: "0xbbb"},
				},
			},
			wantErr: `duplicate MCMAddress "0xbbb" in chain metadata`,
		},
		{
			name: "failure: nested additionals",
			give: ChainMetadata{
				MCMAddress: "0xaaa",
				AdditionalMCMs: []ChainMetadata{
					{MCMAddress: "0xbbb", AdditionalMCMs: []ChainMetadata{{MCMAddress: "0xccc"}}},
				},
			},
			wantErr: "additional MCM entries must not themselves have AdditionalMCMs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.give.ValidateMultiMCM()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func TestChainMetadata_Merge_AdditionalMCMs(t *testing.T) {
	t.Parallel()

	t.Run("union of disjoint additionals", func(t *testing.T) {
		t.Parallel()
		a := ChainMetadata{
			StartingOpCount: 5,
			MCMAddress:      "0xaaa",
			AdditionalMCMs:  []ChainMetadata{{StartingOpCount: 1, MCMAddress: "0xbbb"}},
		}
		b := ChainMetadata{
			StartingOpCount: 7,
			MCMAddress:      "0xaaa",
			AdditionalMCMs:  []ChainMetadata{{StartingOpCount: 3, MCMAddress: "0xccc"}},
		}

		merged, err := a.Merge(b)
		require.NoError(t, err)
		assert.Equal(t, uint64(5), merged.StartingOpCount) // min of primaries
		assert.Equal(t, "0xaaa", merged.MCMAddress)
		require.Len(t, merged.AdditionalMCMs, 2)
		assert.Equal(t, "0xbbb", merged.AdditionalMCMs[0].MCMAddress)
		assert.Equal(t, uint64(1), merged.AdditionalMCMs[0].StartingOpCount)
		assert.Equal(t, "0xccc", merged.AdditionalMCMs[1].MCMAddress)
		assert.Equal(t, uint64(3), merged.AdditionalMCMs[1].StartingOpCount)
	})

	t.Run("same additional address merges per-instance rules", func(t *testing.T) {
		t.Parallel()
		a := ChainMetadata{
			MCMAddress:     "0xaaa",
			AdditionalMCMs: []ChainMetadata{{StartingOpCount: 1, MCMAddress: "0xbbb"}},
		}
		b := ChainMetadata{
			MCMAddress:     "0xaaa",
			AdditionalMCMs: []ChainMetadata{{StartingOpCount: 4, MCMAddress: "0xbbb"}},
		}

		merged, err := a.Merge(b)
		require.NoError(t, err)
		require.Len(t, merged.AdditionalMCMs, 1)
		assert.Equal(t, uint64(1), merged.AdditionalMCMs[0].StartingOpCount) // min
	})

	t.Run("different primaries still rejected", func(t *testing.T) {
		t.Parallel()
		a := ChainMetadata{MCMAddress: "0xaaa"}
		b := ChainMetadata{MCMAddress: "0xbbb"}

		_, err := a.Merge(b)
		require.ErrorContains(t, err, "cannot merge ChainMetadata with different MCMAddress")
	})
}

func TestChainMetadata_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	md := ChainMetadata{
		StartingOpCount:  5,
		MCMAddress:       "0xaaa",
		AdditionalFields: json.RawMessage(`{"chainId": "1"}`),
		AdditionalMCMs: []ChainMetadata{
			{
				StartingOpCount:  2,
				MCMAddress:       "0xbbb",
				AdditionalFields: json.RawMessage(`{"chainId": "1"}`),
			},
		},
	}

	data, err := json.Marshal(md)
	require.NoError(t, err)

	var got ChainMetadata
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, md.MCMAddress, got.MCMAddress)
	assert.Equal(t, md.StartingOpCount, got.StartingOpCount)
	require.Len(t, got.AdditionalMCMs, 1)
	assert.Equal(t, "0xbbb", got.AdditionalMCMs[0].MCMAddress)
	assert.Equal(t, uint64(2), got.AdditionalMCMs[0].StartingOpCount)

	// Single-MCM metadata must not emit the additionalMCMs key
	single := ChainMetadata{StartingOpCount: 1, MCMAddress: "0xaaa"}
	singleData, err := json.Marshal(single)
	require.NoError(t, err)
	assert.NotContains(t, string(singleData), "additionalMCMs")
}
