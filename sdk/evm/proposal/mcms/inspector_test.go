package evm

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/smartcontractkit/mcms/internal/core/config"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	evm_mocks "github.com/smartcontractkit/mcms/sdk/evm/mocks"
	"github.com/smartcontractkit/mcms/types"
)

func TestEVMInspector_GetConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		address    string
		mockResult bindings.ManyChainMultiSigConfig
		mockError  error
		want       *config.Config
		wantErr    error
	}{
		{
			name:    "getConfig call success",
			address: "0x1234567890abcdef1234567890abcdef12345678",
			mockResult: bindings.ManyChainMultiSigConfig{
				Signers: []bindings.ManyChainMultiSigSigner{
					{Addr: common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdef"), Index: 0, Group: 0},
					{Addr: common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"), Index: 1, Group: 0},
					{Addr: common.HexToAddress("0x1234567890abcdef1234567890abcdef56785678"), Index: 2, Group: 0},
					{Addr: common.HexToAddress("0x1111111111111111111111111111111111111111"), Index: 0, Group: 1},
					{Addr: common.HexToAddress("0x2222222222222222222222222222222222222222"), Index: 1, Group: 1},
					{Addr: common.HexToAddress("0x3333333333333333333333333333333333333333"), Index: 2, Group: 1},
				},
				GroupQuorums: [32]uint8{3, 2}, // Valid configuration
				GroupParents: [32]uint8{0, 0},
			},
			want: &config.Config{
				Quorum: 3,
				Signers: []common.Address{
					common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdef"),
					common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
					common.HexToAddress("0x1234567890abcdef1234567890abcdef56785678"),
				},
				GroupSigners: []config.Config{
					{
						Quorum: 2,
						Signers: []common.Address{
							common.HexToAddress("0x1111111111111111111111111111111111111111"),
							common.HexToAddress("0x2222222222222222222222222222222222222222"),
							common.HexToAddress("0x3333333333333333333333333333333333333333"),
						},
						GroupSigners: []config.Config{},
					},
				},
			},
		},
		{
			name:      "CallContract error",
			address:   "0x1234567890abcdef1234567890abcdef12345678",
			mockError: errors.New("CallContract failed"),
			want:      nil,
			wantErr:   fmt.Errorf("CallContract failed"),
		},
		{
			name:       "Empty Signers list",
			address:    "0x1234567890abcdef1234567890abcdef12345678",
			mockResult: bindings.ManyChainMultiSigConfig{Signers: []bindings.ManyChainMultiSigSigner{}, GroupQuorums: [32]uint8{3, 2}, GroupParents: [32]uint8{0, 0}},
			want:       nil,
			wantErr:    fmt.Errorf("invalid MCMS config: Quorum must be less than or equal to the number of signers and groups"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := new(evm_mocks.ContractDeployBackend)

			// Encode mock result if there's no CallContract error
			var encodedConfig []byte
			if tt.mockResult.Signers != nil {
				var err error
				parsedABI, err := bindings.ManyChainMultiSigMetaData.GetAbi()
				require.NoError(t, err)

				// Locate the `getConfig` method's output argument definitions
				method, exists := parsedABI.Methods["getConfig"]
				assert.True(t, exists, "getConfig method should exist in ABI")

				// Use method.Outputs to pack the return values
				encodedConfig, err = method.Outputs.Pack(tt.mockResult)
				require.NoError(t, err)
			}

			// Mock CallContract to return either encodedConfig or mockError
			mockClient.EXPECT().CallContract(mock.Anything, mock.IsType(ethereum.CallMsg{}), mock.IsType(&big.Int{})).
				Return(encodedConfig, tt.mockError)

			// Instantiate EVMInspector with the mock client
			inspector := NewEVMInspector(mockClient)

			// Call GetConfig and capture the result
			result, err := inspector.GetConfig(tt.address)

			// Assertions for want error or successful result
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}

			// Verify CallContract was called as want
			mockClient.AssertExpectations(t)
		})
	}
}

func TestEVMInspector_GetOpCount(t *testing.T) {
	t.Parallel()

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
			address:    "0x1234567890abcdef1234567890abcdef12345678",
			mockResult: big.NewInt(42), // Arbitrary successful op count
			want:       42,
		},
		{
			name:      "CallContract error",
			address:   "0x1234567890abcdef1234567890abcdef12345678",
			mockError: errors.New("CallContract failed"),
			want:      0,
			wantErr:   fmt.Errorf("CallContract failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := new(evm_mocks.ContractDeployBackend)

			// Encode the mock result for CallContract if no error is specified
			var encodedOpCount []byte
			if tt.mockResult != nil {
				parsedABI, err := bindings.ManyChainMultiSigMetaData.GetAbi()
				require.NoError(t, err)

				// Retrieve the method's output type and pack the mock result as output
				method, exists := parsedABI.Methods["getOpCount"]
				assert.True(t, exists, "getOpCount method should exist in ABI")

				encodedOpCount, err = method.Outputs.Pack(tt.mockResult)
				require.NoError(t, err)
			}

			// Mock CallContract to return either the encoded OpCount or an error
			mockClient.EXPECT().CallContract(mock.Anything, mock.IsType(ethereum.CallMsg{}), mock.IsType(&big.Int{})).
				Return(encodedOpCount, tt.mockError)

			// Instantiate EVMInspector with the mock client
			inspector := NewEVMInspector(mockClient)

			// Call GetOpCount and capture the result
			result, err := inspector.GetOpCount(tt.address)

			// Assertions for want error or successful result
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}

			// Verify CallContract was called as want
			mockClient.AssertExpectations(t)
		})
	}
}

func TestEVMInspector_GetRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		address        string
		mockResult     bindings.GetRoot
		mockError      error
		wantRoot       common.Hash
		wantValidUntil uint32
		wantErr        error
	}{
		{
			name:           "GetRoot success",
			address:        "0x1234567890abcdef1234567890abcdef12345678",
			mockResult:     bindings.GetRoot{Root: common.HexToHash("0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdef"), ValidUntil: 1234567890},
			wantRoot:       common.HexToHash("0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdef"),
			wantValidUntil: 1234567890,
		},
		{
			name:      "CallContract error",
			address:   "0x1234567890abcdef1234567890abcdef12345678",
			mockError: errors.New("CallContract failed"),
			wantErr:   fmt.Errorf("CallContract failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := new(evm_mocks.ContractDeployBackend)

			// Encode mock result for CallContract if no error is specified
			var encodedRoot []byte
			if tt.mockError == nil {
				parsedABI, err := bindings.ManyChainMultiSigMetaData.GetAbi()
				require.NoError(t, err)

				// Retrieve the method's output type and pack the mock result as output
				method, exists := parsedABI.Methods["getRoot"]
				assert.True(t, exists, "getRoot method should exist in ABI")

				encodedRoot, err = method.Outputs.Pack(tt.mockResult.Root, tt.mockResult.ValidUntil)
				require.NoError(t, err)
			}

			// Mock CallContract to return the encoded root or an error
			mockClient.EXPECT().CallContract(mock.Anything, mock.IsType(ethereum.CallMsg{}), mock.IsType(&big.Int{})).
				Return(encodedRoot, tt.mockError)

			// Instantiate EVMInspector with the mock client
			inspector := NewEVMInspector(mockClient)

			// Call GetRoot and capture the result
			root, validUntil, err := inspector.GetRoot(tt.address)

			// Assertions for want error or successful result
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErr.Error())

			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantRoot, root)
				assert.Equal(t, tt.wantValidUntil, validUntil)
			}

			// Verify CallContract was called as want
			mockClient.AssertExpectations(t)
		})
	}
}

func TestEVMInspector_GetRootMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		address    string
		mockResult bindings.ManyChainMultiSigRootMetadata
		mockError  error
		want       types.ChainMetadata
		wantErr    error
	}{
		{
			name:    "GetRootMetadata success",
			address: "0x1234567890abcdef1234567890abcdef12345678",
			mockResult: bindings.ManyChainMultiSigRootMetadata{
				ChainId:              big.NewInt(1),
				MultiSig:             common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
				PreOpCount:           big.NewInt(123),
				PostOpCount:          big.NewInt(456),
				OverridePreviousRoot: false,
			},
			want: types.ChainMetadata{
				StartingOpCount: 123,
				MCMAddress:      "0x1234567890abcdef1234567890abcdef12345678",
			},
		},
		{
			name:      "CallContract error",
			address:   "0x1234567890abcdef1234567890abcdef12345678",
			mockError: errors.New("CallContract failed"),
			wantErr:   fmt.Errorf("CallContract failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := new(evm_mocks.ContractDeployBackend)

			// Encode the mock result for CallContract if no error is specified
			var encodedMetadata []byte
			if tt.mockError == nil {
				parsedABI, err := bindings.ManyChainMultiSigMetaData.GetAbi()
				require.NoError(t, err)

				// Retrieve the method's output type and pack the mock result as output
				method, exists := parsedABI.Methods["getRootMetadata"]
				assert.True(t, exists, "getRootMetadata method should exist in ABI")

				encodedMetadata, err = method.Outputs.Pack(tt.mockResult)
				require.NoError(t, err)
			}

			// Mock CallContract to return either the encoded metadata or an error
			mockClient.EXPECT().CallContract(mock.Anything, mock.IsType(ethereum.CallMsg{}), mock.IsType(&big.Int{})).
				Return(encodedMetadata, tt.mockError)

			// Instantiate EVMInspector with the mock client
			inspector := NewEVMInspector(mockClient)

			// Call GetRootMetadata and capture the result
			result, err := inspector.GetRootMetadata(tt.address)

			// Assertions for want error or successful result
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}

			// Verify CallContract was called as want
			mockClient.AssertExpectations(t)
		})
	}
}
