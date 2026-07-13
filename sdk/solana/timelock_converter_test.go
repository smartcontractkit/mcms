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
							ContractType: rbacTimelockContractType,
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
							ContractType: rbacTimelockContractType,
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
							ContractType: rbacTimelockContractType,
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
							ContractType: rbacTimelockContractType,
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
							ContractType: rbacTimelockContractType,
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
							ContractType: rbacTimelockContractType,
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
							ContractType: rbacTimelockContractType,
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
							ContractType: rbacTimelockContractType,
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
							ContractType: rbacTimelockContractType,
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
							ContractType: rbacTimelockContractType,
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
							ContractType: rbacTimelockContractType,
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
							ContractType: rbacTimelockContractType,
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
							ContractType: rbacTimelockContractType,
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
							ContractType: rbacTimelockContractType,
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
							ContractType: rbacTimelockContractType,
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
			wantErr:         "unable to convert batch operation to solana instructions: unable to parse program id from To field: ",
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

func TestTimelockConverter_ExecutePayerSignerOverride(t *testing.T) {
	t.Parallel()

	timelockAddress := ContractAddress(testTimelockProgramID, testPDASeed)
	mcmAddress := ContractAddress(testMCMProgramID, testPDASeed)

	proposerAC, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)
	cancellerAC, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)
	bypasserAC, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	payer, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	baseMetadata := AdditionalFieldsMetadata{
		ProposerRoleAccessController:  proposerAC.PublicKey(),
		CancellerRoleAccessController: cancellerAC.PublicKey(),
		BypasserRoleAccessController:  bypasserAC.PublicKey(),
	}
	metadataWithoutPayer := types.ChainMetadata{MCMAddress: mcmAddress}
	metadataWithPayer := types.ChainMetadata{MCMAddress: mcmAddress}
	metaBytes, err := json.Marshal(baseMetadata)
	require.NoError(t, err)
	metadataWithoutPayer.AdditionalFields = metaBytes
	metaBytesWithPayer, err := json.Marshal(baseMetadata.WithExecutePayer(payer.PublicKey()))
	require.NoError(t, err)
	metadataWithPayer.AdditionalFields = metaBytesWithPayer

	// A batch op whose remaining accounts include the payer as a writable,
	// non-signer account (mirroring a BPF spill / transfer recipient).
	batchOp := func() types.BatchOperation {
		return types.BatchOperation{
			ChainSelector: chaintest.Chain4Selector,
			Transactions: []types.Transaction{{
				To:   "11111111111111111111111111111111",
				Data: []byte{1, 2, 3, 4},
				AdditionalFields: toJSON(t, AdditionalFields{Accounts: []*solana.AccountMeta{
					{PublicKey: payer.PublicKey(), IsWritable: true},
				}}),
				OperationMetadata: types.OperationMetadata{ContractType: "System", Tags: []string{"t"}},
			}},
		}
	}

	convert := func(t *testing.T, metadata types.ChainMetadata, action types.TimelockAction) []types.Operation {
		t.Helper()
		ops, _, cerr := TimelockConverter{}.ConvertBatchToChainOperations(context.Background(), metadata, batchOp(),
			timelockAddress, mcmAddress, types.NewDuration(time.Second), action, common.Hash{},
			common.HexToHash("0x01"))
		require.NoError(t, cerr)
		require.NotEmpty(t, ops)

		return ops
	}

	// payerIsSigner reports the IsSigner flag of the payer account in the last
	// converted op (BypasserExecuteBatch for a bypass), and whether it was found.
	payerIsSigner := func(t *testing.T, ops []types.Operation) (isSigner, found bool) {
		t.Helper()
		last := ops[len(ops)-1]
		var fields AdditionalFields
		require.NoError(t, json.Unmarshal(last.Transaction.AdditionalFields, &fields))
		for _, acc := range fields.Accounts {
			if acc.PublicKey.Equals(payer.PublicKey()) {
				return acc.IsSigner, true
			}
		}

		return false, false
	}

	t.Run("bypass, no execute payer in metadata: payer stays non-signer", func(t *testing.T) {
		t.Parallel()
		isSigner, found := payerIsSigner(t, convert(t, metadataWithoutPayer, types.TimelockActionBypass))
		require.True(t, found, "payer must appear in bypass remaining accounts")
		require.False(t, isSigner)
	})

	t.Run("bypass, metadata with execute payer: payer becomes signer", func(t *testing.T) {
		t.Parallel()
		isSigner, found := payerIsSigner(t, convert(t, metadataWithPayer, types.TimelockActionBypass))
		require.True(t, found)
		require.True(t, isSigner)
	})

	t.Run("bypass, metadata payer not in accounts: no-op", func(t *testing.T) {
		t.Parallel()
		other, oerr := solana.NewRandomPrivateKey()
		require.NoError(t, oerr)
		metaBytesOtherPayer, merr := json.Marshal(baseMetadata.WithExecutePayer(other.PublicKey()))
		require.NoError(t, merr)
		metadataOtherPayer := types.ChainMetadata{MCMAddress: mcmAddress, AdditionalFields: metaBytesOtherPayer}
		isSigner, found := payerIsSigner(t, convert(t, metadataOtherPayer, types.TimelockActionBypass))
		require.True(t, found)
		require.False(t, isSigner, "override must not touch accounts other than the configured payer")
	})

	t.Run("schedule, metadata with execute payer: unchanged", func(t *testing.T) {
		t.Parallel()
		withPayer := convert(t, metadataWithPayer, types.TimelockActionSchedule)
		withoutPayer := convert(t, metadataWithoutPayer, types.TimelockActionSchedule)
		require.Empty(t, cmp.Diff(withoutPayer, withPayer), "schedule conversion must ignore execute payers")
	})

	t.Run("bypass, zero execute payer pointer: treated as unset", func(t *testing.T) {
		t.Parallel()
		zero := solana.PublicKey{}
		metaBytesZero, zerr := json.Marshal(AdditionalFieldsMetadata{
			ProposerRoleAccessController:  proposerAC.PublicKey(),
			CancellerRoleAccessController: cancellerAC.PublicKey(),
			BypasserRoleAccessController:  bypasserAC.PublicKey(),
			ExecutePayer:                  &zero,
		})
		require.NoError(t, zerr)
		metadataZeroPayer := types.ChainMetadata{MCMAddress: mcmAddress, AdditionalFields: metaBytesZero}
		isSigner, found := payerIsSigner(t, convert(t, metadataZeroPayer, types.TimelockActionBypass))
		require.True(t, found)
		require.False(t, isSigner, "zero ExecutePayer must not trigger the signer override")
	})
}

func TestApplyExecutePayerSignerOverride(t *testing.T) {
	t.Parallel()

	a := solana.NewWallet().PublicKey()
	b := solana.NewWallet().PublicKey()
	accounts := []*solana.AccountMeta{
		{PublicKey: a, IsWritable: true},
		{PublicKey: b, IsWritable: true},
	}

	applyExecutePayerSignerOverride(accounts, b)

	require.False(t, accounts[0].IsSigner, "non-payer account must be untouched")
	require.True(t, accounts[1].IsSigner, "payer account must be marked signer")

	t.Run("no match leaves accounts unchanged", func(t *testing.T) {
		t.Parallel()
		other := solana.NewWallet().PublicKey()
		accts := []*solana.AccountMeta{{PublicKey: a, IsWritable: true}}
		applyExecutePayerSignerOverride(accts, other)
		require.False(t, accts[0].IsSigner)
	})

	t.Run("empty accounts is a no-op", func(t *testing.T) {
		t.Parallel()
		require.NotPanics(t, func() {
			applyExecutePayerSignerOverride(nil, a)
			applyExecutePayerSignerOverride([]*solana.AccountMeta{}, a)
		})
	})
}

func TestBypassRemainingAccounts(t *testing.T) {
	t.Parallel()

	payer := solana.NewWallet().PublicKey()
	baseMeta := AdditionalFieldsMetadata{
		ProposerRoleAccessController:  solana.NewWallet().PublicKey(),
		CancellerRoleAccessController: solana.NewWallet().PublicKey(),
		BypasserRoleAccessController:  solana.NewWallet().PublicKey(),
	}

	validBatch := types.BatchOperation{
		ChainSelector: chaintest.Chain4Selector,
		Transactions: []types.Transaction{{
			To:   "11111111111111111111111111111111",
			Data: []byte{1},
			AdditionalFields: toJSON(t, AdditionalFields{Accounts: []*solana.AccountMeta{
				{PublicKey: payer, IsWritable: true},
			}}),
		}},
	}

	t.Run("marks execute payer as signer", func(t *testing.T) {
		t.Parallel()
		accounts, err := bypassRemainingAccounts(validBatch, baseMeta.WithExecutePayer(payer))
		require.NoError(t, err)
		require.Len(t, accounts, 2) // program id + payer
		found := false
		for _, acc := range accounts {
			if acc.PublicKey.Equals(payer) {
				found = true
				require.True(t, acc.IsSigner)
			}
		}
		require.True(t, found)
	})

	t.Run("without execute payer keeps non-signer", func(t *testing.T) {
		t.Parallel()
		accounts, err := bypassRemainingAccounts(validBatch, baseMeta)
		require.NoError(t, err)
		for _, acc := range accounts {
			if acc.PublicKey.Equals(payer) {
				require.False(t, acc.IsSigner)
			}
		}
	})

	t.Run("returns error for invalid program id", func(t *testing.T) {
		t.Parallel()
		badBatch := types.BatchOperation{
			ChainSelector: chaintest.Chain4Selector,
			Transactions: []types.Transaction{{
				To:               "not-a-valid-solana-program-id",
				Data:             []byte{1},
				AdditionalFields: toJSON(t, AdditionalFields{}),
			}},
		}
		_, err := bypassRemainingAccounts(badBatch, baseMeta)
		require.Error(t, err)
		require.ErrorContains(t, err, "unable to parse program id")
	})

	t.Run("returns error for nil account in additional fields", func(t *testing.T) {
		t.Parallel()
		badBatch := types.BatchOperation{
			ChainSelector: chaintest.Chain4Selector,
			Transactions: []types.Transaction{{
				To:               "11111111111111111111111111111111",
				Data:             []byte{1},
				AdditionalFields: []byte(`{"accounts":[null]}`),
			}},
		}
		_, err := bypassRemainingAccounts(badBatch, baseMeta)
		require.Error(t, err)
		require.ErrorContains(t, err, "nil account in batch operation additional fields")
	})
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

func TestOperationID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		batchOp     types.BatchOperation
		action      types.TimelockAction
		predecessor common.Hash
		salt        common.Hash
		want        common.Hash
		wantErr     string
	}{
		{
			name: "success: schedule proposal",
			batchOp: types.BatchOperation{
				ChainSelector: chaintest.Chain4Selector,
				Transactions: []types.Transaction{{
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
					OperationMetadata: types.OperationMetadata{},
				}, {
					To:                "t3ChqFTKHUFdjNPDf8CuhFGwkwzqR47LL7sDbeU99XD",
					Data:              []byte("0x5678"),
					AdditionalFields:  []byte{},
					OperationMetadata: types.OperationMetadata{},
				}},
			},
			action:      types.TimelockActionSchedule,
			predecessor: common.HexToHash("0x0123"),
			salt:        common.HexToHash("0xabcd"),
			want:        common.HexToHash("0x6b6cdd5d076633e336ced0996555e37bb4895c9f18bdfabfe7f068bfab37c080"),
		},
		{
			name: "success: bypass proposal",
			batchOp: types.BatchOperation{
				ChainSelector: chaintest.Chain4Selector,
				Transactions: []types.Transaction{{
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
					OperationMetadata: types.OperationMetadata{},
				}, {
					To:                "t3ChqFTKHUFdjNPDf8CuhFGwkwzqR47LL7sDbeU99XD",
					Data:              []byte("0x5678"),
					AdditionalFields:  []byte{},
					OperationMetadata: types.OperationMetadata{},
				}},
			},
			action:      types.TimelockActionBypass,
			predecessor: common.HexToHash("0x0123"),
			salt:        common.HexToHash("0xabcd"),
			want:        common.HexToHash("0x342cfab7d5a36183bab6c3e35e7ddfa800e0746d5112857021c5cbe96ace2b4b"),
		},
		{
			name: "failure: bad additional fields",
			batchOp: types.BatchOperation{
				ChainSelector: chaintest.Chain4Selector,
				Transactions: []types.Transaction{{
					To:                "GwAQ33PbytKignFmKvyVSLp7pD8tKMaBXXNwFTTkGsME",
					Data:              []byte("0x1234"),
					AdditionalFields:  []byte(`invalid`),
					OperationMetadata: types.OperationMetadata{},
				}},
			},
			action:      types.TimelockActionSchedule,
			predecessor: common.HexToHash("0x0123"),
			salt:        common.HexToHash("0xabcd"),
			wantErr:     "unable to convert batch operation to solana instructions: unable to unmarshal Solana additional fields: invalid character",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			operationID, err := OperationID(tt.batchOp, tt.action, tt.predecessor, tt.salt)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.want, operationID)
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
