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
	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/timelock"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tlbe"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"

	"github.com/smartcontractkit/mcms/sdk/ton"
	"github.com/smartcontractkit/mcms/types"
)

// Map of contract type to TL-B definitions (type -> opcode -> TL-B struct)
var typeToTLBMap = map[string]tvm.TLBMap{
	// MCMS contract types
	"com.chainlink.ton.lib.access.RBAC": rbac.TLBs,
	"com.chainlink.ton.mcms.MCMS":       mcms.TLBs,
	"com.chainlink.ton.mcms.Timelock":   timelock.TLBs,
}

func TestDecoder(t *testing.T) {
	t.Parallel()

	exampleRole := crypto.Keccak256Hash([]byte("EXAMPLE_ROLE"))
	// Notice: need to convert to Uint256 (big.Int)
	exampleRoleBig := tlbe.NewUint256(exampleRole.Big())

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
			name: "success - empty message",
			give: types.Operation{
				ChainSelector: 1,
				Transaction: must(ton.NewTransaction(
					address.MustParseAddr("EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"),
					cell.BeginCell().ToSlice(),
					big.NewInt(0),
					"RBACTimelock",
					[]string{"topUp"},
				)),
			},
			contractInterfaces: "com.chainlink.ton.lib.access.RBAC",
			want: &ton.DecodedOperation{
				ContractType: "com.chainlink.ton.lib.access.RBAC",
				MsgType:      "",
				MsgDecoded:   map[string]any{},
				InputKeys:    []string{},
				InputArgs:    []any{},
			},
			wantErr: "",
		},
		{
			name: "success - message with body",
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
				MsgOpcode:    0x95cd540f,
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

			d := ton.NewDecoder(typeToTLBMap)
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
