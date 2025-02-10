package solana

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/google/go-cmp/cmp"
	bindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/timelock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/sdk/solana/mocks"
	"github.com/smartcontractkit/mcms/types"
)

func Test_NewTimelockConverter(t *testing.T) {
	t.Parallel()

	client := &rpc.Client{}

	converter := NewTimelockConverter(client)

	require.NotNil(t, converter)
	require.Equal(t, client, converter.client)
}

func Test_TimelockConverter_ConvertBatchToChainOperations(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	configPDA, err := FindTimelockConfigPDA(testTimelockProgramID, testPDASeed)
	require.NoError(t, err)
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

	tests := []struct {
		name            string
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
		setup           func(t *testing.T, mockJSONRPCClient *mocks.JSONRPCClient)
	}{
		{
			name:            "success: schedule action",
			batchOp:         defaultBatchOperation(),
			timelockAddress: defaultTimelockAddress,
			mcmAddress:      defaultMCMAddress,
			delay:           defaultDelay,
			action:          types.TimelockActionSchedule,
			predecessor:     common.Hash{},
			salt:            defaultSalt,
			wantOperations: []types.Operation{
				{
					// initialize
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
					// append first operation
					ChainSelector: chaintest.Chain4Selector,
					Transaction: types.Transaction{
						To:   testTimelockProgramID.String(),
						Data: base64Decode(t, "OjqJenMzkIZ0ZXN0LW1jbQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAOzNziC5jaIAHmroyBw0o6rhzkqnV4l5BvFaLyVxMtx/AQAAAOy/SwEm9OHsR3Iyhg37tMCACi9OJO1mbYf2EdbL8BdlBgAAADB4MTIzNAIAAABzrkiHQ+JGjN94Ifr0hyI7xbfT2AUTMNwqcY6ldOvyZQEBkKcZjjIczPUeb2jNRtBAfOQA/tDFWu7x9HDKmqHFfGAAAQ=="),
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
					// append second operation
					ChainSelector: chaintest.Chain4Selector,
					Transaction: types.Transaction{
						To:   testTimelockProgramID.String(),
						Data: base64Decode(t, "OjqJenMzkIZ0ZXN0LW1jbQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAOzNziC5jaIAHmroyBw0o6rhzkqnV4l5BvFaLyVxMtx/AQAAAA0THF/kMtia7NQl9/Z+bKF0Ggj7DNa/3WxLGpQ1QAaIBgAAADB4NTY3OAAAAAA="),
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
			setup: func(t *testing.T, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()
				config := &bindings.Config{
					ProposerRoleAccessController: proposerAC.PublicKey(),
				}
				mockGetAccountInfo(t, mockJSONRPCClient, configPDA, config, nil)
			},
		},
		{
			name:            "success: cancel action",
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
							{PublicKey: solana.MPK("11111111111111111111111111111111")},
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
			setup: func(t *testing.T, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()
				config := &bindings.Config{}
				mockGetAccountInfo(t, mockJSONRPCClient, configPDA, config, nil)
			},
		},
		{
			name:            "success: bypass action",
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
					// append first operation
					ChainSelector: chaintest.Chain4Selector,
					Transaction: types.Transaction{
						To:   testTimelockProgramID.String(),
						Data: base64Decode(t, "f0QI0mrVGdd0ZXN0LW1jbQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAOzNziC5jaIAHmroyBw0o6rhzkqnV4l5BvFaLyVxMtx/AQAAAOy/SwEm9OHsR3Iyhg37tMCACi9OJO1mbYf2EdbL8BdlBgAAADB4MTIzNAIAAABzrkiHQ+JGjN94Ifr0hyI7xbfT2AUTMNwqcY6ldOvyZQEBkKcZjjIczPUeb2jNRtBAfOQA/tDFWu7x9HDKmqHFfGAAAQ=="),
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
					// append second operation
					ChainSelector: chaintest.Chain4Selector,
					Transaction: types.Transaction{
						To:   testTimelockProgramID.String(),
						Data: base64Decode(t, "f0QI0mrVGdd0ZXN0LW1jbQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAOzNziC5jaIAHmroyBw0o6rhzkqnV4l5BvFaLyVxMtx/AQAAAA0THF/kMtia7NQl9/Z+bKF0Ggj7DNa/3WxLGpQ1QAaIBgAAADB4NTY3OAAAAAA="),
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
					// schedule batch
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
						}}),
						OperationMetadata: types.OperationMetadata{
							ContractType: "RBACTimelock",
							Tags:         []string{"tag1.1", "tag1.2", "tag2.1", "tag2.2"},
						},
					},
				},
			},
			wantPredecessor: common.HexToHash("0xeccdce20b98da2001e6ae8c81c34a3aae1ce4aa757897906f15a2f257132dc7f"),
			setup: func(t *testing.T, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()
				config := &bindings.Config{
					BypasserRoleAccessController: bypasserAC.PublicKey(),
				}
				mockGetAccountInfo(t, mockJSONRPCClient, configPDA, config, nil)
			},
		},
		{
			name:            "failure: invalid Timelock address",
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
			name:            "failure: GetAccountInfo error",
			batchOp:         defaultBatchOperation(),
			timelockAddress: defaultTimelockAddress,
			mcmAddress:      defaultMCMAddress,
			delay:           defaultDelay,
			action:          types.TimelockActionSchedule,
			predecessor:     common.Hash{},
			salt:            defaultSalt,
			wantErr:         "unable to read timelock config pda: GetAccountInfo error",
			setup: func(t *testing.T, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()
				mockGetAccountInfo(t, mockJSONRPCClient, configPDA, &bindings.Config{}, fmt.Errorf("GetAccountInfo error"))
			},
		},
		{
			name:            "failure: invalid action",
			batchOp:         defaultBatchOperation(),
			timelockAddress: defaultTimelockAddress,
			mcmAddress:      defaultMCMAddress,
			delay:           defaultDelay,
			action:          types.TimelockAction("invalid-action"),
			predecessor:     common.Hash{},
			salt:            defaultSalt,
			wantErr:         "unable to build invalid-action instruction: invalid timelock operation: invalid-action",
			setup: func(t *testing.T, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()
				mockGetAccountInfo(t, mockJSONRPCClient, configPDA, &bindings.Config{}, nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			converter, mockJSONRPCClient := newTestTimelockConverter(t)
			if tt.setup != nil {
				tt.setup(t, mockJSONRPCClient)
			}

			operations, predecessors, err := converter.ConvertBatchToChainOperations(ctx, tt.batchOp, tt.timelockAddress,
				tt.mcmAddress, tt.delay, tt.action, tt.predecessor, tt.salt)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Empty(t, cmp.Diff(tt.wantOperations, operations))
				require.Empty(t, cmp.Diff(tt.wantPredecessor, predecessors))
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

// ----- helpers -----

func newTestTimelockConverter(t *testing.T) (*TimelockConverter, *mocks.JSONRPCClient) {
	t.Helper()

	mockJSONRPCClient := mocks.NewJSONRPCClient(t)
	client := rpc.NewWithCustomRPCClient(mockJSONRPCClient)
	converter := NewTimelockConverter(client)

	return converter, mockJSONRPCClient
}

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
