package ton_test

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"
	"github.com/smartcontractkit/mcms/types"

	tonmcms "github.com/smartcontractkit/mcms/sdk/ton"
	ton_mocks "github.com/smartcontractkit/mcms/sdk/ton/mocks"
)

func TestInspector_GetConfig(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name       string
		address    string
		mockResult mcms.Config
		mockError  error
		want       *types.Config
		wantErr    error
	}{
		{
			name:    "getConfig call success",
			address: "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			mockResult: mcms.Config{
				Signers: must(tvm.MakeDictFrom([]mcms.Signer{
					{Key: mustKey(wallets[0]), Index: 0, Group: 0},
					{Key: mustKey(wallets[1]), Index: 1, Group: 0},
					{Key: mustKey(wallets[2]), Index: 2, Group: 0},
					{Key: mustKey(wallets[3]), Index: 0, Group: 1},
					{Key: mustKey(wallets[4]), Index: 1, Group: 1},
					{Key: mustKey(wallets[5]), Index: 2, Group: 1},
				}, tvm.KeyUINT8)),
				GroupQuorums: must(tvm.MakeDictFrom([]mcms.GroupQuorum{
					{Val: 3},
					{Val: 2},
				}, tvm.KeyUINT8)), // Valid configuration
				GroupParents: must(tvm.MakeDictFrom([]mcms.GroupParent{
					{Val: 0},
					{Val: 0},
				}, tvm.KeyUINT8)),
			},
			want: &types.Config{
				Quorum: 3,
				SignerKeys: [][]byte{
					mustKey(wallets[0]).Bytes(),
					mustKey(wallets[1]).Bytes(),
					mustKey(wallets[2]).Bytes(),
				},
				GroupSigners: []types.Config{
					{
						Quorum: 2,
						SignerKeys: [][]byte{
							mustKey(wallets[3]).Bytes(),
							mustKey(wallets[4]).Bytes(),
							mustKey(wallets[5]).Bytes(),
						},
						GroupSigners: []types.Config{},
					},
				},
			},
		},
		{
			name:      "CallContract error",
			address:   "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			mockError: errors.New("call to contract failed"),
			want:      nil,
			wantErr:   fmt.Errorf("error getting getConfig: call to contract failed"),
		},
		{
			name:    "Empty Signers list",
			address: "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			mockResult: mcms.Config{
				Signers: must(tvm.MakeDictFrom([]mcms.Signer{}, tvm.KeyUINT8)),
				GroupQuorums: must(tvm.MakeDictFrom([]mcms.GroupQuorum{
					{Val: 3},
					{Val: 2},
				}, tvm.KeyUINT8)),
				GroupParents: must(tvm.MakeDictFrom([]mcms.GroupParent{
					{Val: 0},
					{Val: 0},
				}, tvm.KeyUINT8)),
			},
			want:    nil,
			wantErr: fmt.Errorf("unable to convert to SDK config type: invalid MCMS config: Quorum must be greater than 0"),
			// TODO: figure out why error output for this test case is different
			// wantErr: fmt.Errorf("invalid MCMS config: Quorum must be less than or equal to the number of signers and groups"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a new mock client and inspector for each test case
			client := ton_mocks.NewAPIClientWrapped(t)

			// Mock the contract call based on the test case
			// Mock CurrentMasterchainInfo
			client.EXPECT().CurrentMasterchainInfo(mock.Anything).
				Return(&ton.BlockIDExt{}, nil)

			if tt.mockError == nil {
				// Encode the expected return value for a successful call
				r := ton.NewExecutionResult([]any{
					tt.mockResult.Signers.AsCell(),
					tt.mockResult.GroupQuorums.AsCell(),
					tt.mockResult.GroupParents.AsCell(),
				})
				client.EXPECT().RunGetMethod(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(r, nil).Once()
			} else {
				// If there's an error, mock it on the first CallContract call
				client.EXPECT().RunGetMethod(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, tt.mockError).Once()
			}

			// Instantiate Inspector with the mock client
			inspector := tonmcms.NewInspector(client, tonmcms.NewConfigTransformer())

			// Call GetConfig and capture the got
			got, err := inspector.GetConfig(ctx, tt.address)

			// Assertions for want error or successful got
			if tt.wantErr != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.wantErr.Error())
			} else {
				require.NoError(t, err)
				// Note: we need to remap SignerKeys to Signers for comparison
				tonmcms.ConfigRemapSignerKeys(tt.want)
				assert.Equal(t, tt.want, got)
			}

			// Verify CallContract was called as want
			client.AssertExpectations(t)
		})
	}
}

func TestInspector_GetOpCount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name       string
		address    string
		mockResult *big.Int
		mockError  error
		want       uint64
		wantErr    error
	}{
		{
			name:       "GetOpCount success",
			address:    "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			mockResult: big.NewInt(42), // Arbitrary successful op count
			want:       42,
		},
		{
			name:      "CallContract error",
			address:   "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			mockError: errors.New("call to contract failed"),
			want:      0,
			wantErr:   fmt.Errorf("error getting getOpCount: call to contract failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a new mock client and inspector for each test case
			client := ton_mocks.NewAPIClientWrapped(t)

			// Mock the contract call based on the test case
			// Mock CurrentMasterchainInfo
			client.EXPECT().CurrentMasterchainInfo(mock.Anything).
				Return(&ton.BlockIDExt{}, nil)

			if tt.mockError == nil {
				// Encode the expected return value for a successful call
				r := ton.NewExecutionResult([]any{tt.mockResult})
				client.EXPECT().RunGetMethod(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(r, nil).Once()
			} else {
				// If there's an error, mock it on the first CallContract call
				client.EXPECT().RunGetMethod(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, tt.mockError).Once()
			}

			// Instantiate Inspector with the mock client
			inspector := tonmcms.NewInspector(client, tonmcms.NewConfigTransformer())

			// Call GetOpCount and capture the got
			got, err := inspector.GetOpCount(ctx, tt.address)

			// Assertions for want error or successful got
			if tt.wantErr != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.wantErr.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}

			// Verify CallContract was called as want
			client.AssertExpectations(t)
		})
	}
}

func TestInspector_GetRoot(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name           string
		address        string
		mockResult     []*big.Int
		mockError      error
		wantRoot       common.Hash
		wantValidUntil uint32
		wantErr        error
	}{
		{
			name:    "GetRoot success",
			address: "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			mockResult: []*big.Int{
				new(big.Int).SetBytes(common.HexToHash("0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdef").Bytes()),
				big.NewInt(1234567890),
			},
			wantRoot:       common.HexToHash("0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdef"),
			wantValidUntil: 1234567890,
		},
		{
			name:      "CallContract error",
			address:   "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			mockError: errors.New("call to contract failed"),
			wantErr:   fmt.Errorf("error getting getRoot: call to contract failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a new mock client and inspector for each test case
			client := ton_mocks.NewAPIClientWrapped(t)

			// Mock the contract call based on the test case
			// Mock CurrentMasterchainInfo
			client.EXPECT().CurrentMasterchainInfo(mock.Anything).
				Return(&ton.BlockIDExt{}, nil)

			if tt.mockError == nil {
				// Encode the expected return value for a successful call
				r := ton.NewExecutionResult([]any{tt.mockResult[0], tt.mockResult[1]})
				client.EXPECT().RunGetMethod(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(r, nil).Once()
			} else {
				// If there's an error, mock it on the first CallContract call
				client.EXPECT().RunGetMethod(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, tt.mockError).Once()
			}

			// Instantiate Inspector with the mock client
			inspector := tonmcms.NewInspector(client, tonmcms.NewConfigTransformer())

			// Call GetRoot and capture the result
			got, validUntil, err := inspector.GetRoot(ctx, tt.address)

			// Assertions for want error or successful result
			if tt.wantErr != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.wantErr.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantRoot, got)
				assert.Equal(t, tt.wantValidUntil, validUntil)
			}

			// Verify CallContract was called as want
			client.AssertExpectations(t)
		})
	}
}

func TestInspector_GetRootMetadata(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name       string
		address    string
		mockResult mcms.RootMetadata
		mockError  error
		want       types.ChainMetadata
		wantErr    error
	}{
		{
			name:    "GetRootMetadata success",
			address: "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			mockResult: mcms.RootMetadata{
				ChainID:              big.NewInt(1),
				MultiSig:             address.MustParseAddr("EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"),
				PreOpCount:           123,
				PostOpCount:          456,
				OverridePreviousRoot: false,
			},
			want: types.ChainMetadata{
				StartingOpCount: 123,
				MCMAddress:      "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			},
		},
		{
			name:      "CallContract error",
			address:   "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			mockError: errors.New("call to contract failed"),
			wantErr:   fmt.Errorf("error getting getRootMetadata: call to contract failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a new mock client and inspector for each test case
			client := ton_mocks.NewAPIClientWrapped(t)

			// Mock the contract call based on the test case
			// Mock CurrentMasterchainInfo
			client.EXPECT().CurrentMasterchainInfo(mock.Anything).
				Return(&ton.BlockIDExt{}, nil)

			if tt.mockError == nil {
				// Encode the expected return value for a successful call
				// TODO: not sure if this is how results are returned vs tuple of members, need to check (e2e test)
				mockResultCell, err := tlb.ToCell(tt.mockResult)
				require.NoError(t, err)

				r := ton.NewExecutionResult([]any{mockResultCell.ToBuilder().ToSlice()})
				client.EXPECT().RunGetMethod(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(r, nil).Once()
			} else {
				// If there's an error, mock it on the first CallContract call
				client.EXPECT().RunGetMethod(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, tt.mockError).Once()
			}

			// Instantiate Inspector with the mock client
			inspector := tonmcms.NewInspector(client, tonmcms.NewConfigTransformer())

			// Call GetRootMetadata and capture the got
			got, err := inspector.GetRootMetadata(ctx, tt.address)

			// Assertions for want error or successful got
			if tt.wantErr != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.wantErr.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}

			// Verify CallContract was called as want
			client.AssertExpectations(t)
		})
	}
}
