package solana

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/types"
)

func TestTimelockConverter_ConvertBatchToChainOperations(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	defaultTimelockAddress := ContractAddress(testTimelockProgramID, testPDASeed)
	defaultMCMAddress := ContractAddress(testMCMProgramID, testPDASeed)
	defaultDelay := types.NewDuration(10 * time.Second)
	defaultSalt := common.HexToHash("0x0000000000000000000000000000000000000001")
	defaultBatchOperation := func() types.BatchOperation {
		return types.BatchOperation{
			ChainSelector: chaintest.Chain4Selector,
			Transactions: []types.Transaction{
				{
					To:   "GwAQ33PbytKignFmKvyVSLp7pD8tKMaBXXNwFTTkGsME",
					Data: []byte("0x1234"),
					AdditionalFields: []byte(`{
						"accounts": [{
							"PublicKey": "8na2HyqgS15GcjiWmMQvQ87o8kw188QgaVSTa6q94orU",
							"IsWritable": true,
							"IsSigner": true
						}, {
							"PublicKey": "AjfVZUFzzC8nyA37GXBEdB57RfqPYXifYNtP9jRdRtCw",
							"IsWritable": true
						}],
						"value": 123
					}`),
					OperationMetadata: types.OperationMetadata{
						ContractType: "Contract1",
						Tags:         []string{"tag1.1", "tag1.2"},
					},
				},
				{
					To:               "t3ChqFTKHUFdjNPDf8CuhFGwkwzqR47LL7sDbeU99XD",
					Data:             []byte("0x5678"),
					AdditionalFields: []byte{},
					OperationMetadata: types.OperationMetadata{
						ContractType: "Contract2",
						Tags:         []string{"tag2.1", "tag2.2"},
					},
				},
			},
		}
	}

	proposerAC, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	bypasserAC, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	cancellerAC, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	additionalFieldsMeta := AdditionalFieldsMetadata{
		ProposerRoleAccessController:  proposerAC.PublicKey(),
		CancellerRoleAccessController: cancellerAC.PublicKey(),
		BypasserRoleAccessController:  bypasserAC.PublicKey(),
	}
	additionalFieldsMetaBytes, err := json.Marshal(additionalFieldsMeta)
	require.NoError(t, err)
	metadata := types.ChainMetadata{
		MCMAddress:       defaultMCMAddress,
		AdditionalFields: additionalFieldsMetaBytes,
	}
	tests := []struct {
		name            string
		metadata        types.ChainMetadata
		batchOp         types.BatchOperation
		timelockAddress string
		mcmAddress      string
		delay           types.Duration
		action          types.TimelockAction
		predecessor     common.Hash
		salt            common.Hash
		wantOperations  []types.Operation
		wantPredecessor common.Hash
		wantErr         string
	}{
		{
			name:            "success: schedule action",
			metadata:        metadata,
			batchOp:         defaultBatchOperation(),
			timelockAddress: defaultTimelockAddress,
			mcmAddress:      defaultMCMAddress,
			delay:           defaultDelay,
			action:          types.TimelockActionSchedule,
			predecessor:     common.Hash{},
			salt:            defaultSalt,
			wantOperations: []types.Operation{
				{
					// initialize operation
					ChainSelector: chaintest.Chain4Selector,
					Transaction: types.Transaction{
						To:   testTimelockProgramID.String(),
						Data: base64Decode(t, "D2DZq3wEcfN0ZXN0LW1jbQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAOzNziC5jaIAHmroyBw0o6rhzkqnV4l5BvFaLyVxMtx/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQIAAAA="),
						AdditionalFields: toJSON(t, AdditionalFields{Accounts: []*solana.AccountMeta{
							{PublicKey: solana.MPK("3x12f1G4bt9j7rsBfLE7rZQ5hXoHuHdjtUr2UKW8gjQp"), IsWritable: true},
							{PublicKey: solana.MPK("GYWcPzXkdzY9DJLcbFs67phqyYzmJxeEKSTtqEoo8oKz")},
							{PublicKey: proposerAC.PublicKey()},
							{PublicKey: solana.MPK("62gDM6BRLf2w1yXfmpePUTsuvbeBbu4QqdjV32wcc4UG"), IsWritable: true},
							{PublicKey: solana.MPK("11111111111111111111111111111111")},
						}}),
						OperationMetadata: types.OperationMetadata{
							ContractType: "RBACTimelock",
							Tags:         []string{"tag1.1", "tag1.2", "tag2.1", "tag2.2"},
						},
					},
				},
				{
					// initialize first instruction
					ChainSelector: chaintest.Chain4Selector,
					Transaction: types.Transaction{
						To:   testTimelockProgramID.String(),
						Data: base64Decode(t, "w+bVh5CUjlV0ZXN0LW1jbQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAOzNziC5jaIAHmroyBw0o6rhzkqnV4l5BvFaLyVxMtx/7L9LASb04exHcjKGDfu0wIAKL04k7WZth/YR1svwF2UCAAAAc65Ih0PiRozfeCH69IciO8W309gFEzDcKnGOpXTr8mUBAZCnGY4yHMz1Hm9ozUbQQHzkAP7QxVru8fRwypqhxXxgAAE="),
						AdditionalFields: toJSON(t, AdditionalFields{Accounts: []*solana.AccountMeta{
							{PublicKey: solana.MPK("3x12f1G4bt9j7rsBfLE7rZQ5hXoHuHdjtUr2UKW8gjQp"), IsWritable: true},
							{PublicKey: solana.MPK("GYWcPzXkdzY9DJLcbFs67phqyYzmJxeEKSTtqEoo8oKz")},
							{PublicKey: proposerAC.PublicKey()},
							{PublicKey: solana.MPK("62gDM6BRLf2w1yXfmpePUTsuvbeBbu4QqdjV32wcc4UG"), IsWritable: true},
							{PublicKey: solana.MPK("11111111111111111111111111111111")},
						}}),
						OperationMetadata: types.OperationMetadata{
							ContractType: "RBACTimelock",
							Tags:         []string{"tag1.1", "tag1.2", "tag2.1", "tag2.2"},
						},
					},
				},
				{
					// append first instruction data
					ChainSelector: chaintest.Chain4Selector,
					Transaction: types.Transaction{
						To:   testTimelockProgramID.String(),
						Data: base64Decode(t, "TE1mg4gMLQV0ZXN0LW1jbQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAOzNziC5jaIAHmroyBw0o6rhzkqnV4l5BvFaLyVxMtx/AAAAAAYAAAAweDEyMzQ="),
						AdditionalFields: toJSON(t, AdditionalFields{Accounts: []*solana.AccountMeta{
							{PublicKey: solana.MPK("3x12f1G4bt9j7rsBfLE7rZQ5hXoHuHdjtUr2UKW8gjQp"), IsWritable: true},
							{PublicKey: solana.MPK("GYWcPzXkdzY9DJLcbFs67phqyYzmJxeEKSTtqEoo8oKz")},
							{PublicKey: proposerAC.PublicKey()},
							{PublicKey: solana.MPK("62gDM6BRLf2w1yXfmpePUTsuvbeBbu4QqdjV32wcc4UG"), IsWritable: true},
							{PublicKey: solana.MPK("11111111111111111111111111111111")},
						}}),
						OperationMetadata: types.OperationMetadata{
							ContractType: "RBACTimelock",
							Tags:         []string{"tag1.1", "tag1.2", "tag2.1", "tag2.2"},
						},
					},
				},
				{
					// initialize second instruction
					ChainSelector: chaintest.Chain4Selector,
					Transaction: types.Transaction{
						To:   testTimelockProgramID.String(),
						Data: base64Decode(t, "w+bVh5CUjlV0ZXN0LW1jbQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAOzNziC5jaIAHmroyBw0o6rhzkqnV4l5BvFaLyVxMtx/DRMcX+Qy2Jrs1CX39n5soXQaCPsM1r/dbEsalDVABogAAAAA"),
						AdditionalFields: toJSON(t, AdditionalFields{Accounts: []*solana.AccountMeta{
							{PublicKey: solana.MPK("3x12f1G4bt9j7rsBfLE7rZQ5hXoHuHdjtUr2UKW8gjQp"), IsWritable: true},
							{PublicKey: solana.MPK("GYWcPzXkdzY9DJLcbFs67phqyYzmJxeEKSTtqEoo8oKz")},
							{PublicKey: proposerAC.PublicKey()},
							{PublicKey: solana.MPK("62gDM6BRLf2w1yXfmpePUTsuvbeBbu4QqdjV32wcc4UG"), IsWritable: true},
							{PublicKey: solana.MPK("11111111111111111111111111111111")},
						}}),
						OperationMetadata: types.OperationMetadata{
							ContractType: "RBACTimelock",
							Tags:         []string{"tag1.1", "tag1.2", "tag2.1", "tag2.2"},
						},
					},
				},
				{
					// append second instruction data
					ChainSelector: chaintest.Chain4Selector,
					Transaction: types.Transaction{
						To:   testTimelockProgramID.String(),
						Data: base64Decode(t, "TE1mg4gMLQV0ZXN0LW1jbQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAOzNziC5jaIAHmroyBw0o6rhzkqnV4l5BvFaLyVxMtx/AQAAAAYAAAAweDU2Nzg="),
						AdditionalFields: toJSON(t, AdditionalFields{Accounts: []*solana.AccountMeta{
							{PublicKey: solana.MPK("3x12f1G4bt9j7rsBfLE7rZQ5hXoHuHdjtUr2UKW8gjQp"), IsWritable: true},
							{PublicKey: solana.MPK("GYWcPzXkdzY9DJLcbFs67phqyYzmJxeEKSTtqEoo8oKz")},
							{PublicKey: proposerAC.PublicKey()},
							{PublicKey: solana.MPK("62gDM6BRLf2w1yXfmpePUTsuvbeBbu4QqdjV32wcc4UG"), IsWritable: true},
							{PublicKey: solana.MPK("11111111111111111111111111111111")},
						}}),
						OperationMetadata: types.OperationMetadata{
							ContractType: "RBACTimelock",
							Tags:         []string{"tag1.1", "tag1.2", "tag2.1", "tag2.2"},
						},
					},
				},
				{
					// finalize
					ChainSelector: chaintest.Chain4Selector,
					Transaction: types.Transaction{
						To:   testTimelockProgramID.String(),
						Data: base64Decode(t, "P9AgYlW27Ix0ZXN0LW1jbQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAOzNziC5jaIAHmroyBw0o6rhzkqnV4l5BvFaLyVxMtx/"),
						AdditionalFields: toJSON(t, AdditionalFields{Accounts: []*solana.AccountMeta{
							{PublicKey: solana.MPK("3x12f1G4bt9j7rsBfLE7rZQ5hXoHuHdjtUr2UKW8gjQp"), IsWritable: true},
							{PublicKey: solana.MPK("GYWcPzXkdzY9DJLcbFs67phqyYzmJxeEKSTtqEoo8oKz")},
							{PublicKey: proposerAC.PublicKey()},
							{PublicKey: solana.MPK("62gDM6BRLf2w1yXfmpePUTsuvbeBbu4QqdjV32wcc4UG"), IsWritable: true},
						}}),
						OperationMetadata: types.OperationMetadata{
							ContractType: "RBACTimelock",
							Tags:         []string{"tag1.1", "tag1.2", "tag2.1", "tag2.2"},
						},
					},
				},
				{
					// schedule batch
					ChainSelector: chaintest.Chain4Selector,
					Transaction: types.Transaction{
						To:   testTimelockProgramID.String(),
						Data: base64Decode(t, "8oxXakfiViB0ZXN0LW1jbQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAOzNziC5jaIAHmroyBw0o6rhzkqnV4l5BvFaLyVxMtx/CgAAAAAAAAA="),
						AdditionalFields: toJSON(t, AdditionalFields{Accounts: []*solana.AccountMeta{
							{PublicKey: solana.MPK("3x12f1G4bt9j7rsBfLE7rZQ5hXoHuHdjtUr2UKW8gjQp"), IsWritable: true},
							{PublicKey: solana.MPK("GYWcPzXkdzY9DJLcbFs67phqyYzmJxeEKSTtqEoo8oKz")},
							{PublicKey: proposerAC.PublicKey()},
							{PublicKey: solana.MPK("62gDM6BRLf2w1yXfmpePUTsuvbeBbu4QqdjV32wcc4UG"), IsWritable: true},
						}}),
						OperationMetadata: types.OperationMetadata{
							ContractType: "RBACTimelock",
							Tags:         []string{"tag1.1", "tag1.2", "tag2.1", "tag2.2"},
						},
					},
				},
			},
			wantPredecessor: common.HexToHash("0xeccdce20b98da2001e6ae8c81c34a3aae1ce4aa757897906f15a2f257132dc7f"),
		},
		{
			name:            "success: cancel action",
			metadata:        metadata,
			batchOp:         defaultBatchOperation(),
			timelockAddress: defaultTimelockAddress,
			mcmAddress:      defaultMCMAddress,
			delay:           defaultDelay,
			action:          types.TimelockActionCancel,
			predecessor:     common.Hash{},
			salt:            defaultSalt,
			wantOperations: []types.Operation{
				{
					// schedule batch
					ChainSelector: chaintest.Chain4Selector,
					Transaction: types.Transaction{
						To:   testTimelockProgramID.String(),
						Data: base64Decode(t, "6NvfKdvs3L50ZXN0LW1jbQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAOzNziC5jaIAHmroyBw0o6rhzkqnV4l5BvFaLyVxMtx/"),
						AdditionalFields: toJSON(t, AdditionalFields{Accounts: []*solana.AccountMeta{
							{PublicKey: solana.MPK("3x12f1G4bt9j7rsBfLE7rZQ5hXoHuHdjtUr2UKW8gjQp"), IsWritable: true},
							{PublicKey: solana.MPK("GYWcPzXkdzY9DJLcbFs67phqyYzmJxeEKSTtqEoo8oKz")},
							{PublicKey: cancellerAC.PublicKey()},
							{PublicKey: solana.MPK("62gDM6BRLf2w1yXfmpePUTsuvbeBbu4QqdjV32wcc4UG"), IsWritable: true},
						}}),
						OperationMetadata: types.OperationMetadata{
							ContractType: "RBACTimelock",
							Tags:         []string{"tag1.1", "tag1.2", "tag2.1", "tag2.2"},
						},
					},
				},
			},
			wantPredecessor: common.HexToHash("0xeccdce20b98da2001e6ae8c81c34a3aae1ce4aa757897906f15a2f257132dc7f"),
		},
		{
			name:            "success: bypass action",
			metadata:        metadata,
			batchOp:         defaultBatchOperation(),
			timelockAddress: defaultTimelockAddress,
			mcmAddress:      defaultMCMAddress,
			delay:           defaultDelay,
			action:          types.TimelockActionBypass,
			predecessor:     common.Hash{},
			salt:            defaultSalt,
			wantOperations: []types.Operation{
				{
					// initialize
					ChainSelector: chaintest.Chain4Selector,
					Transaction: types.Transaction{
						To:   testTimelockProgramID.String(),
						Data: base64Decode(t, "OhswzBPFPxp0ZXN0LW1jbQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAOzNziC5jaIAHmroyBw0o6rhzkqnV4l5BvFaLyVxMtx/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAECAAAA"),
						AdditionalFields: toJSON(t, AdditionalFields{Accounts: []*solana.AccountMeta{
							{PublicKey: solana.MPK("5kA68ZDhLguQqxoEraVvdSYzR43JddUZXEm8SCdCEQfh"), IsWritable: true},
							{PublicKey: solana.MPK("GYWcPzXkdzY9DJLcbFs67phqyYzmJxeEKSTtqEoo8oKz")},
							{PublicKey: solana.MPK(bypasserAC.PublicKey().String())},
							{PublicKey: solana.MPK("62gDM6BRLf2w1yXfmpePUTsuvbeBbu4QqdjV32wcc4UG"), IsWritable: true},
							{PublicKey: solana.MPK("11111111111111111111111111111111")},
						}}),
						OperationMetadata: types.OperationMetadata{
							ContractType: "RBACTimelock",
							Tags:         []string{"tag1.1", "tag1.2", "tag2.1", "tag2.2"},
						},
					},
				},
				{
					// initialize first instruction
					ChainSelector: chaintest.Chain4Selector,
					Transaction: types.Transaction{
						To:   testTimelockProgramID.String(),
						Data: base64Decode(t, "MhHNrK+Mwyd0ZXN0LW1jbQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAOzNziC5jaIAHmroyBw0o6rhzkqnV4l5BvFaLyVxMtx/7L9LASb04exHcjKGDfu0wIAKL04k7WZth/YR1svwF2UCAAAAc65Ih0PiRozfeCH69IciO8W309gFEzDcKnGOpXTr8mUBAZCnGY4yHMz1Hm9ozUbQQHzkAP7QxVru8fRwypqhxXxgAAE="),
						AdditionalFields: toJSON(t, AdditionalFields{Accounts: []*solana.AccountMeta{
							{PublicKey: solana.MPK("5kA68ZDhLguQqxoEraVvdSYzR43JddUZXEm8SCdCEQfh"), IsWritable: true},
							{PublicKey: solana.MPK("GYWcPzXkdzY9DJLcbFs67phqyYzmJxeEKSTtqEoo8oKz")},
							{PublicKey: solana.MPK(bypasserAC.PublicKey().String())},
							{PublicKey: solana.MPK("62gDM6BRLf2w1yXfmpePUTsuvbeBbu4QqdjV32wcc4UG"), IsWritable: true},
							{PublicKey: solana.MPK("11111111111111111111111111111111")},
						}}),
						OperationMetadata: types.OperationMetadata{
							ContractType: "RBACTimelock",
							Tags:         []string{"tag1.1", "tag1.2", "tag2.1", "tag2.2"},
						},
					},
				},
				{
					// append first instruction
					ChainSelector: chaintest.Chain4Selector,
					Transaction: types.Transaction{
						To:   testTimelockProgramID.String(),
						Data: base64Decode(t, "uOiX3m9118V0ZXN0LW1jbQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAOzNziC5jaIAHmroyBw0o6rhzkqnV4l5BvFaLyVxMtx/AAAAAAYAAAAweDEyMzQ="),
						AdditionalFields: toJSON(t, AdditionalFields{Accounts: []*solana.AccountMeta{
							{PublicKey: solana.MPK("5kA68ZDhLguQqxoEraVvdSYzR43JddUZXEm8SCdCEQfh"), IsWritable: true},
							{PublicKey: solana.MPK("GYWcPzXkdzY9DJLcbFs67phqyYzmJxeEKSTtqEoo8oKz")},
							{PublicKey: solana.MPK(bypasserAC.PublicKey().String())},
							{PublicKey: solana.MPK("62gDM6BRLf2w1yXfmpePUTsuvbeBbu4QqdjV32wcc4UG"), IsWritable: true},
							{PublicKey: solana.MPK("11111111111111111111111111111111")},
						}}),
						OperationMetadata: types.OperationMetadata{
							ContractType: "RBACTimelock",
							Tags:         []string{"tag1.1", "tag1.2", "tag2.1", "tag2.2"},
						},
					},
				},
				{
					// initialize second instruction
					ChainSelector: chaintest.Chain4Selector,
					Transaction: types.Transaction{
						To:   testTimelockProgramID.String(),
						Data: base64Decode(t, "MhHNrK+Mwyd0ZXN0LW1jbQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAOzNziC5jaIAHmroyBw0o6rhzkqnV4l5BvFaLyVxMtx/DRMcX+Qy2Jrs1CX39n5soXQaCPsM1r/dbEsalDVABogAAAAA"),
						AdditionalFields: toJSON(t, AdditionalFields{Accounts: []*solana.AccountMeta{
							{PublicKey: solana.MPK("5kA68ZDhLguQqxoEraVvdSYzR43JddUZXEm8SCdCEQfh"), IsWritable: true},
							{PublicKey: solana.MPK("GYWcPzXkdzY9DJLcbFs67phqyYzmJxeEKSTtqEoo8oKz")},
							{PublicKey: solana.MPK(bypasserAC.PublicKey().String())},
							{PublicKey: solana.MPK("62gDM6BRLf2w1yXfmpePUTsuvbeBbu4QqdjV32wcc4UG"), IsWritable: true},
							{PublicKey: solana.MPK("11111111111111111111111111111111")},
						}}),
						OperationMetadata: types.OperationMetadata{
							ContractType: "RBACTimelock",
							Tags:         []string{"tag1.1", "tag1.2", "tag2.1", "tag2.2"},
						},
					},
				},
				{
					// append second instruction
					ChainSelector: chaintest.Chain4Selector,
					Transaction: types.Transaction{
						To:   testTimelockProgramID.String(),
						Data: base64Decode(t, "uOiX3m9118V0ZXN0LW1jbQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAOzNziC5jaIAHmroyBw0o6rhzkqnV4l5BvFaLyVxMtx/AQAAAAYAAAAweDU2Nzg="),
						AdditionalFields: toJSON(t, AdditionalFields{Accounts: []*solana.AccountMeta{
							{PublicKey: solana.MPK("5kA68ZDhLguQqxoEraVvdSYzR43JddUZXEm8SCdCEQfh"), IsWritable: true},
							{PublicKey: solana.MPK("GYWcPzXkdzY9DJLcbFs67phqyYzmJxeEKSTtqEoo8oKz")},
							{PublicKey: solana.MPK(bypasserAC.PublicKey().String())},
							{PublicKey: solana.MPK("62gDM6BRLf2w1yXfmpePUTsuvbeBbu4QqdjV32wcc4UG"), IsWritable: true},
							{PublicKey: solana.MPK("11111111111111111111111111111111")},
						}}),
						OperationMetadata: types.OperationMetadata{
							ContractType: "RBACTimelock",
							Tags:         []string{"tag1.1", "tag1.2", "tag2.1", "tag2.2"},
						},
					},
				},
				{
					// finalize
					ChainSelector: chaintest.Chain4Selector,
					Transaction: types.Transaction{
						To:   testTimelockProgramID.String(),
						Data: base64Decode(t, "LTfGM3wYqfp0ZXN0LW1jbQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAOzNziC5jaIAHmroyBw0o6rhzkqnV4l5BvFaLyVxMtx/"),
						AdditionalFields: toJSON(t, AdditionalFields{Accounts: []*solana.AccountMeta{
							{PublicKey: solana.MPK("5kA68ZDhLguQqxoEraVvdSYzR43JddUZXEm8SCdCEQfh"), IsWritable: true},
							{PublicKey: solana.MPK("GYWcPzXkdzY9DJLcbFs67phqyYzmJxeEKSTtqEoo8oKz")},
							{PublicKey: solana.MPK(bypasserAC.PublicKey().String())},
							{PublicKey: solana.MPK("62gDM6BRLf2w1yXfmpePUTsuvbeBbu4QqdjV32wcc4UG"), IsWritable: true},
						}}),
						OperationMetadata: types.OperationMetadata{
							ContractType: "RBACTimelock",
							Tags:         []string{"tag1.1", "tag1.2", "tag2.1", "tag2.2"},
						},
					},
				},
				{
					// bypass batch execute
					ChainSelector: chaintest.Chain4Selector,
					Transaction: types.Transaction{
						To:   testTimelockProgramID.String(),
						Data: base64Decode(t, "Wj5CBuOuHsJ0ZXN0LW1jbQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAOzNziC5jaIAHmroyBw0o6rhzkqnV4l5BvFaLyVxMtx/"),
						AdditionalFields: toJSON(t, AdditionalFields{Accounts: []*solana.AccountMeta{
							{PublicKey: solana.MPK("5kA68ZDhLguQqxoEraVvdSYzR43JddUZXEm8SCdCEQfh"), IsWritable: true},
							{PublicKey: solana.MPK("GYWcPzXkdzY9DJLcbFs67phqyYzmJxeEKSTtqEoo8oKz")},
							{PublicKey: solana.MPK("2g4vS5Y9g5FKoBakfNTEQcoyuPxuqgiXhribGxE1Vrsb")},
							{PublicKey: solana.MPK(bypasserAC.PublicKey().String())},
							{PublicKey: solana.MPK("62gDM6BRLf2w1yXfmpePUTsuvbeBbu4QqdjV32wcc4UG"), IsWritable: true},
							{PublicKey: solana.MPK("GwAQ33PbytKignFmKvyVSLp7pD8tKMaBXXNwFTTkGsME"), IsWritable: false, IsSigner: false},
							{PublicKey: solana.MPK("8na2HyqgS15GcjiWmMQvQ87o8kw188QgaVSTa6q94orU"), IsWritable: true, IsSigner: false},
							{PublicKey: solana.MPK("AjfVZUFzzC8nyA37GXBEdB57RfqPYXifYNtP9jRdRtCw"), IsWritable: true, IsSigner: false},
							{PublicKey: solana.MPK("t3ChqFTKHUFdjNPDf8CuhFGwkwzqR47LL7sDbeU99XD"), IsWritable: false, IsSigner: false},
						}}),
						OperationMetadata: types.OperationMetadata{
							ContractType: "RBACTimelock",
							Tags:         []string{"tag1.1", "tag1.2", "tag2.1", "tag2.2"},
						},
					},
				},
			},
			wantPredecessor: common.HexToHash("0xeccdce20b98da2001e6ae8c81c34a3aae1ce4aa757897906f15a2f257132dc7f"),
		},
		{
			name:            "failure: invalid Timelock address",
			metadata:        metadata,
			batchOp:         defaultBatchOperation(),
			timelockAddress: "invalid-address",
			mcmAddress:      defaultMCMAddress,
			delay:           defaultDelay,
			action:          types.TimelockActionSchedule,
			predecessor:     common.Hash{},
			salt:            defaultSalt,
			wantErr:         "unable to parse timelock address: invalid solana contract address format",
		},
		{
			name:            "failure: invalid MCM address",
			metadata:        metadata,
			batchOp:         defaultBatchOperation(),
			timelockAddress: defaultTimelockAddress,
			mcmAddress:      "invalid-address",
			delay:           defaultDelay,
			action:          types.TimelockActionSchedule,
			predecessor:     common.Hash{},
			salt:            defaultSalt,
			wantErr:         "unable to parse mcm address: invalid solana contract address format",
		},
		{
			name:            "failure: invalid To address",
			metadata:        metadata,
			timelockAddress: defaultTimelockAddress,
			mcmAddress:      defaultMCMAddress,
			delay:           defaultDelay,
			action:          types.TimelockActionSchedule,
			predecessor:     common.Hash{},
			salt:            defaultSalt,
			wantErr:         "unable to convert batchop to solana instructions: unable to parse program id from To field: ",
			batchOp: func(bop types.BatchOperation) types.BatchOperation {
				bop.Transactions[0].To = "invalid-address"
				return bop
			}(defaultBatchOperation()),
		},
		{
			name:            "failure: invalid action",
			metadata:        metadata,
			batchOp:         defaultBatchOperation(),
			timelockAddress: defaultTimelockAddress,
			mcmAddress:      defaultMCMAddress,
			delay:           defaultDelay,
			action:          types.TimelockAction("invalid-action"),
			predecessor:     common.Hash{},
			salt:            defaultSalt,
			wantErr:         "unable to build invalid-action instruction: invalid timelock operation: invalid-action",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			converter := TimelockConverter{}

			operations, predecessors, err := converter.ConvertBatchToChainOperations(ctx,
				tt.metadata,
				tt.batchOp,
				tt.timelockAddress,
				tt.mcmAddress,
				tt.delay,
				tt.action,
				tt.predecessor,
				tt.salt,
			)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Empty(t, cmp.Diff(tt.wantPredecessor, predecessors))
				require.Empty(t, cmp.Diff(tt.wantOperations, operations))
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

// ----- helpers -----
func toJSON(t *testing.T, payload any) []byte {
	t.Helper()
	marshalled, err := json.Marshal(&payload)
	require.NoError(t, err)

	return marshalled
}

func base64Decode(t *testing.T, str string) []byte {
	t.Helper()
	decodedBytes, err := base64.StdEncoding.DecodeString(str)
	require.NoError(t, err)

	return decodedBytes
}

func TestAppendIxDataChunkSize(t *testing.T) {
	tests := []struct {
		name      string
		numProofs string
		want      int
	}{
		{name: "numProofs: default (8)", numProofs: "", want: 323},
		{name: "numProofs:          0 ", numProofs: "0", want: 579},
		{name: "numProofs:          1 ", numProofs: "1", want: 547},
		{name: "numProofs:          2 ", numProofs: "1", want: 547},
		{name: "numProofs:          8 ", numProofs: "8", want: 323},
		{name: "numProofs:         12 ", numProofs: "12", want: 195},
		{name: "numProofs:         18 ", numProofs: "18", want: 3},
		{name: "numProofs:         19 ", numProofs: "19", want: -29}, // <<< we max at 2^18 operations in a Solana proposal
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MCMS_SOLANA_NUM_PROOFS", tt.numProofs)
			require.Equal(t, tt.want, AppendIxDataChunkSize())
		})
	}
}
