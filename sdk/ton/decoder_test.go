package ton_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/lib/access/rbac"

	"github.com/smartcontractkit/mcms/sdk/ton"
	"github.com/smartcontractkit/mcms/types"
)

func TestDecoder(t *testing.T) {
	t.Parallel()

	exampleRole := crypto.Keccak256Hash([]byte("EXAMPLE_ROLE"))

	// TODO(ton): fix me - what's up with *big.Int decoding as negative num?
	exampleRoleBig, _ := cell.BeginCell().
		MustStoreBigInt(new(big.Int).SetBytes(exampleRole[:]), 257).
		EndCell().
		ToBuilder().
		ToSlice().
		LoadBigInt(256)

	// Grant role data
	grantRoleData, err := tlb.ToCell(rbac.GrantRole{
		QueryID: 0x1,
		Role:    exampleRoleBig,
		Account: address.MustParseAddr("EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"),
	})
	require.NoError(t, err)

	tests := []struct {
		name               string
		give               types.Operation
		contractInterfaces string
		want               *ton.DecodedOperation
		wantErr            string
	}{
		{
			name: "success",
			give: types.Operation{
				ChainSelector: 1,
				Transaction: must(ton.NewTransaction(
					address.MustParseAddr("EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"),
					grantRoleData.ToBuilder().ToSlice(),
					big.NewInt(0),
					"RBACTimelock",
					[]string{"grantRole"},
				)),
			},
			contractInterfaces: "com.chainlink.ton.lib.access.RBAC",
			want: &ton.DecodedOperation{
				ContractType: "com.chainlink.ton.lib.access.RBAC",
				MsgType:      "GrantRole",
				MsgDecoded: map[string]any{
					"QueryID": uint64(0x1),
					"Role":    exampleRoleBig,
					"Account": address.MustParseAddr("EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"),
				},
				InputKeys: []string{"QueryID", "Role", "Account"},
				InputArgs: []any{uint64(0x1), exampleRoleBig, address.MustParseAddr("EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8")},
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := ton.NewDecoder()
			got, err := d.Decode(tt.give.Transaction, tt.contractInterfaces)
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
