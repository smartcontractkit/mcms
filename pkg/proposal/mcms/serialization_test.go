package mcms

import (
	"bytes"
	"io"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/pkg/proposal/mcms/types"
)

const v1Proposal = `{
	"version": "v1",
	"kind": "Proposal",
	"validUntil": 4128029039,
	"signatures": [
		{
			"r": "0x0000000000000000000000000000000000000000000000000000000000000001",
			"s": "0x0000000000000000000000000000000000000000000000000000000000000002",
			"v": 0
		}
	],
	"overridePreviousRoot": false,
	"chainMetadata": {
		"16015286601757825753": {
			"startingOpCount": 0,
			"mcmAddress": "0x0000000000000000000000000000000000000000"
		}
	},
	"description": "Test Proposal",
	"transactions": [
		{
			"chainSelector": 16015286601757825753,
			"to": "0xa4D66959e4580b341D096Eb94311e77a4bac6773",
			"data": "dGVzdGRhdGE=",
			"additionalFields": {"value": 0},
			"contractType": "Storage",
			"tags": ["tag1", "tag2"]
		}
	]
}`

func Test_Decoder_Decode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    []byte
		want    MCMSProposal
		wantErr string
	}{
		{
			name: "success: decode v1 proposal",
			give: []byte(v1Proposal),
			want: MCMSProposal{
				Version:    "v1",
				Kind:       "Proposal",
				ValidUntil: 4128029039,
				Signatures: []Signature{
					{
						R: common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"),
						S: common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002"),
						V: 0,
					},
				},
				ChainMetadata: map[types.ChainIdentifier]ChainMetadata{
					types.ChainIdentifier(16015286601757825753): {
						StartingOpCount: 0,
						MCMAddress:      common.HexToAddress("0x0000000000000000000000000000000000000000"),
					},
				},
				Description: "Test Proposal",
				Transactions: []types.ChainOperation{
					{
						ChainIdentifier: types.ChainIdentifier(16015286601757825753),
						Operation: types.Operation{
							To:               common.HexToAddress("0xa4D66959e4580b341D096Eb94311e77a4bac6773"),
							Data:             []byte("testdata"),
							ContractType:     "Storage",
							AdditionalFields: []byte(`{"value": 0}`),
							Tags:             []string{"tag1", "tag2"},
						},
					},
				},
			},
		},
		{
			name:    "error: invalid JSON",
			give:    []byte(``),
			wantErr: "unexpected end of JSON input",
		},
		{
			name:    "error: invalid version",
			give:    []byte(`{"version": "v-1"}`),
			wantErr: "invalid version: v-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dec := newJSONDecoder(bytes.NewReader(tt.give))
			var got MCMSProposal
			err := dec.Decode(&got)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_Encoder_Encode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    MCMSProposal
		want    string
		wantErr string
	}{
		{
			name: "success: encode v1 proposal",
			give: MCMSProposal{
				Version:    "v1",
				Kind:       "Proposal",
				ValidUntil: 4128029039,
				Signatures: []Signature{
					{
						R: common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"),
						S: common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002"),
						V: 0,
					},
				},
				ChainMetadata: map[types.ChainIdentifier]ChainMetadata{
					types.ChainIdentifier(16015286601757825753): {
						StartingOpCount: 0,
						MCMAddress:      common.HexToAddress("0x0000000000000000000000000000000000000000"),
					},
				},
				Description: "Test Proposal",
				Transactions: []types.ChainOperation{
					{
						ChainIdentifier: types.ChainIdentifier(16015286601757825753),
						Operation: types.Operation{
							To:               common.HexToAddress("0xa4D66959e4580b341D096Eb94311e77a4bac6773"),
							Data:             []byte("testdata"),
							ContractType:     "Storage",
							AdditionalFields: []byte(`{"value": 0}`),
							Tags:             []string{"tag1", "tag2"},
						},
					},
				},
			},
			want: v1Proposal,
		},
		{
			name: "error: invalid version",
			give: MCMSProposal{
				Version: "v-1",
			},
			wantErr: "invalid version: v-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			enc := newJSONEncoder(io.Writer(&buf))
			err := enc.Encode(&tt.give)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.JSONEq(t, tt.want, buf.String())
			}
		})
	}
}
