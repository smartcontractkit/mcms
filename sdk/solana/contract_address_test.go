package solana

import (
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func Test_parseContractAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		address        string
		wantProgramID  solana.PublicKey
		wantPDASeed    PDASeed
		wantContractID string
		wantErr        string
	}{
		{
			name:           "success",
			address:        "6UmMZr5MEqiKWD5jqTJd1WCR5kT8oZuFYBLJFi1o6GQX.test-mcm",
			wantProgramID:  testProgramID,
			wantPDASeed:    testPDASeed,
			wantContractID: "6UmMZr5MEqiKWD5jqTJd1WCR5kT8oZuFYBLJFi1o6GQX.test-mcm",
		},
		{
			name:    "failure: long pda seed",
			address: "6UmMZr5MEqiKWD5jqTJd1WCR5kT8oZuFYBLJFi1o6GQX.really-long-pda-seed-value-0123456789abcdef",
			wantErr: "pda seed is too long (max 32 bytes)",
		},
		{
			name:    "failure: invalid format",
			address: "string-without-a-dot",
			wantErr: "invalid solana contract address format: \"string-without-a-dot\"",
		},
		{
			name:    "failure: invalid program id",
			address: "invalid-program-id.pda-seed",
			wantErr: "unable to parse solana program id: decode: invalid base58 digit",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			programID, pdaSeed, err := ParseContractAddress(tt.address)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Empty(t, cmp.Diff(tt.wantProgramID, programID))
				require.Empty(t, cmp.Diff(tt.wantPDASeed, pdaSeed))

				contractAddress := ContractAddress(programID, pdaSeed)
				require.Empty(t, cmp.Diff(tt.wantContractID, contractAddress))
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
