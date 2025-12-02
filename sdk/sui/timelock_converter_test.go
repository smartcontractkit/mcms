package sui

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
)

func TestNewTimelockConverter(t *testing.T) {
	t.Parallel()

	converter, err := NewTimelockConverter()
	require.NoError(t, err)
	assert.NotNil(t, converter)
}

func TestTimelockConverterConvertBatchToChainOperations(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	tests := []struct {
		name            string
		metadata        types.ChainMetadata
		bop             types.BatchOperation
		timelockAddress string
		mcmAddress      string
		delay           types.Duration
		action          types.TimelockAction
		predecessor     common.Hash
		salt            common.Hash
		wantOpsCount    int
		wantOpID        string
		wantTxDataHex   string
		wantErr         assert.ErrorAssertionFunc
	}{
		{
			name: "success - schedule action with single transaction",
			metadata: types.ChainMetadata{
				StartingOpCount: 1,
				MCMAddress:      "0x123",
			},
			bop: types.BatchOperation{
				ChainSelector: types.ChainSelector(1),
				Transactions: []types.Transaction{
					{
						OperationMetadata: types.OperationMetadata{
							ContractType: "test_contract",
							Tags:         []string{"test_tag"},
						},
						To:   "0x1234567890123456789012345678901234567890123456789012345678901234",
						Data: []byte{0x01, 0x02, 0x03},
						AdditionalFields: MustMarshalJSON(t, AdditionalFields{
							StateObj:   "0xstate123",
							ModuleName: "test_module",
							Function:   "test_function",
						}),
					},
				},
			},
			timelockAddress: "0x456",
			mcmAddress:      "0x789",
			delay:           types.NewDuration(3600 * time.Second),
			action:          types.TimelockActionSchedule,
			predecessor:     common.HexToHash("0xabc"),
			salt:            common.HexToHash("0xdef"),
			wantOpsCount:    1,
			wantErr:         assert.NoError,
			wantTxDataHex:   "011234567890123456789012345678901234567890123456789012345678901234010b746573745f6d6f64756c65010d746573745f66756e6374696f6e0103010203200000000000000000000000000000000000000000000000000000000000000abc200000000000000000000000000000000000000000000000000000000000000def100e000000000000",
			wantOpID:        "0x08f9da5cdabfbd4d838312d342ea28bf922204c82b6bfdf26f057429f8f1d934",
		},
		{
			name: "success - bypass action with multiple transactions",
			metadata: types.ChainMetadata{
				StartingOpCount: 1,
				MCMAddress:      "0x123",
			},
			bop: types.BatchOperation{
				ChainSelector: types.ChainSelector(1),
				Transactions: []types.Transaction{
					{
						OperationMetadata: types.OperationMetadata{
							ContractType: "test_contract1",
							Tags:         []string{"test_tag1"},
						},
						To:   "0x1234567890123456789012345678901234567890123456789012345678901234",
						Data: []byte{0x01, 0x02},
						AdditionalFields: MustMarshalJSON(t, AdditionalFields{
							StateObj:   "0xstate1",
							ModuleName: "module1",
							Function:   "function1",
						}),
					},
					{
						OperationMetadata: types.OperationMetadata{
							ContractType: "test_contract2",
							Tags:         []string{"test_tag2"},
						},
						To:   "0x5678901234567890123456789012345678901234567890123456789012345678",
						Data: []byte{0x03, 0x04},
						AdditionalFields: MustMarshalJSON(t, AdditionalFields{
							StateObj:   "0xstate2",
							ModuleName: "module2",
							Function:   "function2",
						}),
					},
				},
			},
			timelockAddress: "0x456",
			mcmAddress:      "0x789",
			delay:           types.NewDuration(0),
			action:          types.TimelockActionBypass,
			predecessor:     common.Hash{},
			salt:            common.Hash{},
			wantOpsCount:    1,
			wantErr:         assert.NoError,
			wantTxDataHex:   "021234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567802076d6f64756c6531076d6f64756c6532020966756e6374696f6e310966756e6374696f6e3202020102020304",
			wantOpID:        "0xe9ec2c4882b9237ed30523ca69a5d2d4050d7994dea278a428a5879a30b86247",
		},
		{
			name: "success - cancel action",
			metadata: types.ChainMetadata{
				StartingOpCount: 1,
				MCMAddress:      "0x123",
			},
			bop: types.BatchOperation{
				ChainSelector: types.ChainSelector(1),
				Transactions: []types.Transaction{
					{
						OperationMetadata: types.OperationMetadata{
							ContractType: "test_contract",
							Tags:         []string{"test_tag"},
						},
						To:   "0x1234567890123456789012345678901234567890123456789012345678901234",
						Data: []byte{0x01},
						AdditionalFields: MustMarshalJSON(t, AdditionalFields{
							StateObj:   "0xstate123",
							ModuleName: "test_module",
							Function:   "test_function",
						}),
					},
				},
			},
			timelockAddress: "0x456",
			mcmAddress:      "0x789",
			delay:           types.NewDuration(0),
			action:          types.TimelockActionCancel,
			predecessor:     common.Hash{},
			salt:            common.Hash{},
			wantOpsCount:    1,
			wantErr:         assert.NoError,
			wantTxDataHex:   "20b42e07404f27a487b296e607739520026adc536fb1a71f13c96ba1631918c1a5",
			wantOpID:        "0xb42e07404f27a487b296e607739520026adc536fb1a71f13c96ba1631918c1a5",
		},
		{
			name: "failure - invalid additional fields JSON",
			metadata: types.ChainMetadata{
				StartingOpCount: 1,
				MCMAddress:      "0x123",
			},
			bop: types.BatchOperation{
				ChainSelector: types.ChainSelector(1),
				Transactions: []types.Transaction{
					{
						OperationMetadata: types.OperationMetadata{
							ContractType: "test_contract",
							Tags:         []string{"test_tag"},
						},
						To:               "0x1234567890123456789012345678901234567890123456789012345678901234",
						Data:             []byte{0x01},
						AdditionalFields: []byte("{invalid json}"),
					},
				},
			},
			timelockAddress: "0x456",
			mcmAddress:      "0x789",
			delay:           types.NewDuration(3600 * time.Second),
			action:          types.TimelockActionSchedule,
			predecessor:     common.Hash{},
			salt:            common.Hash{},
			wantOpsCount:    0,
			wantErr:         AssertErrorContains("failed to unmarshal additional fields"),
			wantOpID:        "",
		},
		{
			name: "failure - invalid target address",
			metadata: types.ChainMetadata{
				StartingOpCount: 1,
				MCMAddress:      "0x123",
			},
			bop: types.BatchOperation{
				ChainSelector: types.ChainSelector(1),
				Transactions: []types.Transaction{
					{
						OperationMetadata: types.OperationMetadata{
							ContractType: "test_contract",
							Tags:         []string{"test_tag"},
						},
						To:   "invalid_address",
						Data: []byte{0x01},
						AdditionalFields: MustMarshalJSON(t, AdditionalFields{
							StateObj:   "0xstate123",
							ModuleName: "test_module",
							Function:   "test_function",
						}),
					},
				},
			},
			timelockAddress: "0x456",
			mcmAddress:      "0x789",
			delay:           types.NewDuration(3600 * time.Second),
			action:          types.TimelockActionSchedule,
			predecessor:     common.Hash{},
			salt:            common.Hash{},
			wantOpsCount:    0,
			wantErr:         AssertErrorContains("failed to parse target address"),
			wantOpID:        "",
		},
		{
			name: "failure - unsupported timelock action",
			metadata: types.ChainMetadata{
				StartingOpCount: 1,
				MCMAddress:      "0x123",
			},
			bop: types.BatchOperation{
				ChainSelector: types.ChainSelector(1),
				Transactions: []types.Transaction{
					{
						OperationMetadata: types.OperationMetadata{
							ContractType: "test_contract",
							Tags:         []string{"test_tag"},
						},
						To:   "0x1234567890123456789012345678901234567890123456789012345678901234",
						Data: []byte{0x01},
						AdditionalFields: MustMarshalJSON(t, AdditionalFields{
							StateObj:   "0xstate123",
							ModuleName: "test_module",
							Function:   "test_function",
						}),
					},
				},
			},
			timelockAddress: "0x456",
			mcmAddress:      "0x789",
			delay:           types.NewDuration(3600 * time.Second),
			action:          types.TimelockAction("invalid_action"), // Invalid action
			predecessor:     common.Hash{},
			salt:            common.Hash{},
			wantOpsCount:    0,
			wantErr:         AssertErrorContains("unsupported timelock action"),
			wantOpID:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			converter, err := NewTimelockConverter()
			require.NoError(t, err)

			ops, opID, err := converter.ConvertBatchToChainOperations(
				ctx,
				tt.metadata,
				tt.bop,
				tt.timelockAddress,
				tt.mcmAddress,
				tt.delay,
				tt.action,
				tt.predecessor,
				tt.salt,
			)

			if !tt.wantErr(t, err) {
				return
			}

			if err == nil {
				assert.Len(t, ops, tt.wantOpsCount)
				assert.Equal(t, tt.wantOpID, opID.String())
				assert.Equal(t, tt.wantTxDataHex, hex.EncodeToString(ops[0].Transaction.Data[:]))

				if len(ops) > 0 {
					assert.Equal(t, tt.bop.ChainSelector, ops[0].ChainSelector)
					assert.NotNil(t, ops[0].Transaction)
				}
			}
		})
	}
}

func TestHashOperationBatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		targets       [][]byte
		moduleNames   []string
		functionNames []string
		datas         [][]byte
		predecessor   []byte
		salt          []byte
		wantErr       assert.ErrorAssertionFunc
	}{
		{
			name: "success - single operation",
			targets: [][]byte{
				{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20},
			},
			moduleNames:   []string{"test_module"},
			functionNames: []string{"test_function"},
			datas:         [][]byte{{0xaa, 0xbb, 0xcc}},
			predecessor:   []byte{0x11, 0x22, 0x33},
			salt:          []byte{0x44, 0x55, 0x66},
			wantErr:       assert.NoError,
		},
		{
			name: "success - multiple operations",
			targets: [][]byte{
				{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20},
				{0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f, 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f, 0x40},
			},
			moduleNames:   []string{"module1", "module2"},
			functionNames: []string{"function1", "function2"},
			datas:         [][]byte{{0xaa, 0xbb}, {0xcc, 0xdd}},
			predecessor:   []byte{0x11, 0x22},
			salt:          []byte{0x33, 0x44},
			wantErr:       assert.NoError,
		},
		{
			name:          "success - empty operations",
			targets:       [][]byte{},
			moduleNames:   []string{},
			functionNames: []string{},
			datas:         [][]byte{},
			predecessor:   []byte{},
			salt:          []byte{},
			wantErr:       assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hash, err := HashOperationBatch(tt.targets, tt.moduleNames, tt.functionNames, tt.datas, tt.predecessor, tt.salt)

			if !tt.wantErr(t, err) {
				return
			}

			if err == nil {
				assert.NotEqual(t, common.Hash{}, hash)

				// Verify hash is deterministic by running again
				hash2, err2 := HashOperationBatch(tt.targets, tt.moduleNames, tt.functionNames, tt.datas, tt.predecessor, tt.salt)
				require.NoError(t, err2)
				assert.Equal(t, hash, hash2, "Hash should be deterministic")
			}
		})
	}
}

func TestHashOperationBatch_Deterministic(t *testing.T) {
	t.Parallel()

	// Test that different inputs produce different hashes
	targets1 := [][]byte{{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20}}
	targets2 := [][]byte{{0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f, 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f, 0x40}}

	hash1, err1 := HashOperationBatch(targets1, []string{"module1"}, []string{"function1"}, [][]byte{{0xaa}}, []byte{0x11}, []byte{0x22})
	require.NoError(t, err1)

	hash2, err2 := HashOperationBatch(targets2, []string{"module1"}, []string{"function1"}, [][]byte{{0xaa}}, []byte{0x11}, []byte{0x22})
	require.NoError(t, err2)

	assert.NotEqual(t, hash1, hash2, "Different inputs should produce different hashes")
}

func TestTimelockConverterActionConstants(t *testing.T) {
	t.Parallel()

	// Test that the action constants are correctly defined
	assert.Equal(t, "timelock_schedule_batch", TimelockActionSchedule)
	assert.Equal(t, "timelock_cancel", TimelockActionCancel)
	assert.Equal(t, "timelock_bypasser_execute_batch", TimelockActionBypass)
}
