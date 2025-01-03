package mcms

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/mocks"
	"github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"
)

func Test_NewTimelockExecutable(t *testing.T) {
	t.Parallel()

	var (
		executor = mocks.NewTimelockExecutor(t)

		validChainMetadata = map[types.ChainSelector]types.ChainMetadata{
			chaintest.Chain1Selector: {
				StartingOpCount: 1,
				MCMAddress:      "0x123",
			},
		}

		validTimelockAddresses = map[types.ChainSelector]string{
			chaintest.Chain1Selector: "0x123",
		}

		validTx = types.Transaction{
			To:               "0x123",
			AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
			Data:             common.Hex2Bytes("0x"),
			OperationMetadata: types.OperationMetadata{
				ContractType: "Sample contract",
				Tags:         []string{"tag1", "tag2"},
			},
		}

		validBatchOps = []types.BatchOperation{
			{
				ChainSelector: chaintest.Chain1Selector,
				Transactions: []types.Transaction{
					validTx,
				},
			},
		}
	)

	tests := []struct {
		name          string
		giveProposal  *TimelockProposal
		giveExecutors map[types.ChainSelector]sdk.TimelockExecutor
		wantErr       bool
		wantErrMsg    string
	}{
		{
			name: "success",
			giveProposal: &TimelockProposal{
				BaseProposal: BaseProposal{
					Version:              "v1",
					Kind:                 types.KindTimelockProposal,
					Description:          "description",
					ValidUntil:           2004259681,
					OverridePreviousRoot: false,
					Signatures:           []types.Signature{},
					ChainMetadata:        validChainMetadata,
				},
				Action:            types.TimelockActionSchedule,
				Delay:             types.MustParseDuration("1h"),
				TimelockAddresses: validTimelockAddresses,
				Operations:        validBatchOps,
			},
			giveExecutors: map[types.ChainSelector]sdk.TimelockExecutor{
				types.ChainSelector(1): executor,
			},
			wantErr:    false,
			wantErrMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewTimelockExecutable(tt.giveProposal, tt.giveExecutors)

			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
