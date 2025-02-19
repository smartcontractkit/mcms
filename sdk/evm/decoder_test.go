package evm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

func TestDecoder(t *testing.T) {
	t.Parallel()

	// Get ABI
	timelockAbi, err := bindings.RBACTimelockMetaData.GetAbi()
	assert.NoError(t, err)
	exampleRole := crypto.Keccak256Hash([]byte("EXAMPLE_ROLE"))

	// Grant role data
	grantRoleData, err := timelockAbi.Pack("grantRole", [32]byte(exampleRole), common.HexToAddress("0x123"))
	require.NoError(t, err)

	tests := []struct {
		name               string
		give               types.Operation
		contractInterfaces string
		want               *DecodedOperation
		wantErr            string
	}{
		{
			name: "success",
			give: types.Operation{
				ChainSelector: 1,
				Transaction: NewTransaction(
					common.HexToAddress("0xTestTarget"),
					grantRoleData,
					big.NewInt(0),
					"RBACTimelock",
					[]string{"grantRole"},
				),
			},
			contractInterfaces: bindings.RBACTimelockABI,
			want: &DecodedOperation{
				FunctionName: "grantRole",
				InputKeys:    []string{"role", "account"},
				InputArgs:    []any{[32]byte(exampleRole.Bytes()), common.HexToAddress("0x0000000000000000000000000000000000000123")},
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := NewDecoder()
			got, err := d.Decode(tt.give.Transaction, tt.contractInterfaces)
			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
