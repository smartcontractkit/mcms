package solana

import (
	"testing"

	"github.com/gagliardetto/solana-go"
	cpistub "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/external_program_cpi_stub"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
)

func TestNewTransactionFromInstruction(t *testing.T) {
	t.Parallel()

	cpistub.SetProgramID(testCPIStubProgramID)

	tests := []struct {
		name         string
		instruction  solana.Instruction
		contractType string
		tags         []string
		want         types.Transaction
		wantErr      string
	}{
		{
			name: "success",
			instruction: cpistub.NewAccountMutInstruction(
				solana.MPK("8pPNjm5F2xGUG8q7fFwNLcDmAnMDRamEotiDZbJ5seqo"),
				solana.MPK("HzWdnV141bP1PXXce4NJ6NCJgd6jr3kMaySuevbpgwaV"),
				solana.SystemProgramID,
			).Build(),
			contractType: "CPIStub",
			tags:         []string{"tag1", "tag2"},
			want: types.Transaction{
				To:                testCPIStubProgramID.String(),
				Data:              base64Decode(t, "DAKJExbrkEY="),
				OperationMetadata: types.OperationMetadata{ContractType: "CPIStub", Tags: []string{"tag1", "tag2"}},
				AdditionalFields: toJSON(t, AdditionalFields{Accounts: []*solana.AccountMeta{
					{PublicKey: solana.MPK("8pPNjm5F2xGUG8q7fFwNLcDmAnMDRamEotiDZbJ5seqo"), IsWritable: true},
					{PublicKey: solana.MPK("HzWdnV141bP1PXXce4NJ6NCJgd6jr3kMaySuevbpgwaV"), IsSigner: true},
					{PublicKey: solana.SystemProgramID},
				}}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewTransactionFromInstruction(tt.instruction, tt.contractType, tt.tags)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
