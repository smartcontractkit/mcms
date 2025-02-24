package solana

import (
	"testing"

	cpistub "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/external_program_cpi_stub"
	"github.com/stretchr/testify/require"

	// "github.com/gagliardetto/solana-go"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

func TestDecoder_Decode(t *testing.T) {
	t.Parallel()

	instruction := cpistub.NewEmptyInstruction().Build()
	tx, err := NewTransactionFromInstruction(instruction, "contracType", nil)
	require.NoError(t, err)

	tests := []struct {
		name    string
		tx      types.Transaction
		idl     string
		want    sdk.DecodedOperation
		wantErr string
	}{
		{
			name: "success: transfer",
			tx:   tx,
			idl:  cpistubIDL(),
			want: &DecodedOperation{
				FunctionName: "empty",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewDecoder().Decode(tt.tx, tt.idl)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

var cpistubIDL = func() string {
	return `{"version":"0.0.0-dev","name":"external_program_cpi_stub","instructions":[{"name":"initialize","accounts":[{"name":"u8Value","isMut":true,"isSigner":false},{"name":"stubCaller","isMut":true,"isSigner":true},{"name":"systemProgram","isMut":false,"isSigner":false}],"args":[]},{"name":"empty","accounts":[],"args":[]},{"name":"u8InstructionData","accounts":[],"args":[{"name":"data","type":"u8"}]},{"name":"structInstructionData","accounts":[],"args":[{"name":"data","type":{"defined":"Value"}}]},{"name":"accountRead","accounts":[{"name":"u8Value","isMut":false,"isSigner":false}],"args":[]},{"name":"accountMut","accounts":[{"name":"u8Value","isMut":true,"isSigner":false},{"name":"stubCaller","isMut":false,"isSigner":true},{"name":"systemProgram","isMut":false,"isSigner":false}],"args":[]},{"name":"bigInstructionData","docs":["instruction that accepts arbitrarily large instruction data."],"accounts":[],"args":[{"name":"data","type":"bytes"}]}],"accounts":[{"name":"Value","type":{"kind":"struct","fields":[{"name":"value","type":"u8"}]}}]}`
}
