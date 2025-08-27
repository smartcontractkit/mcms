package solana

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/gagliardetto/solana-go"
	cpistub "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/external_program_cpi_stub"
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

func TestValidateAdditionalFields(t *testing.T) {
	t.Parallel()

	validAccount := &solana.AccountMeta{
		PublicKey: solana.MustPublicKeyFromBase58("11111111111111111111111111111111"),
		IsSigner:  true,
	}
	tests := []struct {
		name        string
		input       json.RawMessage
		wantErr     bool
		errContains string
	}{
		{
			name: "valid fields",
			input: func() json.RawMessage {
				fields := AdditionalFields{
					Accounts: []*solana.AccountMeta{validAccount},
					Value:    big.NewInt(100),
				}
				data, err := json.Marshal(fields)
				require.NoError(t, err)

				return data
			}(),
			wantErr: false,
		},
		{
			name:    "missing accounts field",
			input:   []byte(`{"value": 100}`),
			wantErr: false,
		},
		{
			name:    "empty accounts slice",
			input:   []byte(`{"accounts": [], "value": 100}`),
			wantErr: false,
		},
		{
			name:        "malformed json",
			input:       []byte(`invalid json`),
			wantErr:     true,
			errContains: "failed to unmarshal",
		},
		{
			name:  "empty input",
			input: []byte(""),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateAdditionalFields(tc.input)
			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAdditionalFieldsValidate(t *testing.T) {
	t.Parallel()

	validAccount := &solana.AccountMeta{
		PublicKey: solana.MustPublicKeyFromBase58("11111111111111111111111111111111"),
		IsSigner:  true,
	}
	tests := []struct {
		name        string
		fields      AdditionalFields
		wantErr     bool
		errContains string
	}{
		{
			name: "valid additional fields",
			fields: AdditionalFields{
				Accounts: []*solana.AccountMeta{validAccount},
				Value:    big.NewInt(123),
			},
			wantErr: false,
		},
		{
			name: "nil accounts",
			fields: AdditionalFields{
				Accounts: nil,
				Value:    big.NewInt(123),
			},
			wantErr: false,
		},
		{
			name: "empty accounts",
			fields: AdditionalFields{
				Accounts: []*solana.AccountMeta{},
				Value:    big.NewInt(123),
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.fields.Validate()
			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
