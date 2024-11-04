package evm

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	evm_mocks "github.com/smartcontractkit/mcms/sdk/evm/mocks"
)

type roleFetchTest struct {
	name            string
	address         string
	roleMemberCount *big.Int
	roleMembers     []common.Address
	proposerRole    [32]byte
	mockError       error
	want            []common.Address
	wantErr         error
	roleFetchType   string // Specify the role type (proposers, executors, etc.)
}

// Helper to mock contract calls for each role test case
func mockRoleContractCalls(t *testing.T, mockClient *evm_mocks.ContractDeployBackend, parsedABI *abi.ABI, tt roleFetchTest) {
	t.Helper()

	// Mock response for getting the proposer role
	mockClient.EXPECT().CallContract(mock.Anything, mock.IsType(ethereum.CallMsg{}), mock.IsType(&big.Int{})).
		Return(tt.proposerRole[:], nil).Once()

	// Mock response for role member count
	encodedRoleMemberCount, err := parsedABI.Methods["getRoleMemberCount"].Outputs.Pack(tt.roleMemberCount)
	require.NoError(t, err)
	mockClient.EXPECT().CallContract(mock.Anything, mock.IsType(ethereum.CallMsg{}), mock.IsType(&big.Int{})).
		Return(encodedRoleMemberCount, nil).Once()

	// Mock response for each role member
	for _, member := range tt.roleMembers {
		encodedMember, err := parsedABI.Methods["getRoleMember"].Outputs.Pack(member)
		require.NoError(t, err)
		mockClient.EXPECT().CallContract(mock.Anything, mock.Anything, mock.Anything).
			Return(encodedMember, nil).Once()
	}
}

func TestTimelockEVMInspector_GetRolesTests(t *testing.T) {
	t.Parallel()

	tests := []roleFetchTest{
		{
			name:            "GetProposers success",
			address:         "0x1234567890abcdef1234567890abcdef12345678",
			roleMemberCount: big.NewInt(3),
			roleMembers: []common.Address{
				common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdef"),
				common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
				common.HexToAddress("0x1234567890abcdef1234567890abcdef56785678"),
			},
			proposerRole: [32]byte{0x01},
			want: []common.Address{
				common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdef"),
				common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
				common.HexToAddress("0x1234567890abcdef1234567890abcdef56785678"),
			},
			roleFetchType: "proposers",
		},
		{
			name:          "GetProposers call contract failure error",
			address:       "0x1234567890abcdef1234567890abcdef12345678",
			mockError:     errors.New("call to contract failed"),
			want:          nil,
			wantErr:       errors.New("call to contract failed"),
			roleFetchType: "proposers",
		},
		{
			name:            "GetExecutors success",
			address:         "0x1234567890abcdef1234567890abcdef12345678",
			roleMemberCount: big.NewInt(3),
			roleMembers: []common.Address{
				common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdef"),
				common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
				common.HexToAddress("0x1234567890abcdef1234567890abcdef56785678"),
			},
			proposerRole: [32]byte{0x01},
			want: []common.Address{
				common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdef"),
				common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
				common.HexToAddress("0x1234567890abcdef1234567890abcdef56785678"),
			},
			roleFetchType: "executors",
		},
		{
			name:          "GetExecutors call contract failure error",
			address:       "0x1234567890abcdef1234567890abcdef12345678",
			mockError:     errors.New("call to contract failed"),
			want:          nil,
			wantErr:       errors.New("call to contract failed"),
			roleFetchType: "executors",
		},
		{
			name:            "GetExecutors success",
			address:         "0x1234567890abcdef1234567890abcdef12345678",
			roleMemberCount: big.NewInt(3),
			roleMembers: []common.Address{
				common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdef"),
				common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
				common.HexToAddress("0x1234567890abcdef1234567890abcdef56785678"),
			},
			proposerRole: [32]byte{0x01},
			want: []common.Address{
				common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdef"),
				common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
				common.HexToAddress("0x1234567890abcdef1234567890abcdef56785678"),
			},
			roleFetchType: "executors",
		},
		{
			name:          "GetExecutors call contract failure error",
			address:       "0x1234567890abcdef1234567890abcdef12345678",
			mockError:     errors.New("call to contract failed"),
			want:          nil,
			wantErr:       errors.New("call to contract failed"),
			roleFetchType: "executors",
		},
		{
			name:            "GetBypassers success",
			address:         "0x1234567890abcdef1234567890abcdef12345678",
			roleMemberCount: big.NewInt(3),
			roleMembers: []common.Address{
				common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdef"),
				common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
				common.HexToAddress("0x1234567890abcdef1234567890abcdef56785678"),
			},
			proposerRole: [32]byte{0x01},
			want: []common.Address{
				common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdef"),
				common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
				common.HexToAddress("0x1234567890abcdef1234567890abcdef56785678"),
			},
			roleFetchType: "bypassers",
		},
		{
			name:          "GetBypassers call contract failure error",
			address:       "0x1234567890abcdef1234567890abcdef12345678",
			mockError:     errors.New("call to contract failed"),
			want:          nil,
			wantErr:       errors.New("call to contract failed"),
			roleFetchType: "bypassers",
		},
		{
			name:            "GetCancellers success",
			address:         "0x1234567890abcdef1234567890abcdef12345678",
			roleMemberCount: big.NewInt(3),
			roleMembers: []common.Address{
				common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdef"),
				common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
				common.HexToAddress("0x1234567890abcdef1234567890abcdef56785678"),
			},
			proposerRole: [32]byte{0x01},
			want: []common.Address{
				common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdef"),
				common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
				common.HexToAddress("0x1234567890abcdef1234567890abcdef56785678"),
			},
			roleFetchType: "bypassers",
		},
		{
			name:          "GetCancellers call contract failure error",
			address:       "0x1234567890abcdef1234567890abcdef12345678",
			mockError:     errors.New("call to contract failed"),
			want:          nil,
			wantErr:       errors.New("call to contract failed"),
			roleFetchType: "cancellers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a new mock client and inspector for each test case
			mockClient := evm_mocks.NewContractDeployBackend(t)
			inspector := NewTimelockEVMInspector(mockClient)

			// Load the ABI for encoding
			parsedABI, err := bindings.RBACTimelockMetaData.GetAbi()
			require.NoError(t, err)

			// Mock the contract calls based on the test case
			if tt.mockError == nil {
				mockRoleContractCalls(t, mockClient, parsedABI, tt)
			} else {
				// If there's an error, mock it on the first CallContract call
				mockClient.EXPECT().CallContract(mock.Anything, mock.Anything, mock.Anything).
					Return(nil, tt.mockError).Once()
			}

			// Select and call the appropriate role-fetching function
			var got []common.Address
			switch tt.roleFetchType {
			case "proposers":
				got, err = inspector.GetProposers(tt.address)
			case "executors":
				got, err = inspector.GetExecutors(tt.address)
			case "cancellers":
				got, err = inspector.GetCancellers(tt.address)
			case "bypassers":
				got, err = inspector.GetBypassers(tt.address)
			default:
				t.Fatalf("unsupported roleFetchType: %s", tt.roleFetchType)
			}

			// Assertions for expected error or successful result
			if tt.wantErr != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.wantErr.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}

			// Verify expectations
			mockClient.AssertExpectations(t)
		})
	}
}

func TestTimelockEVMInspector_IsOperation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		address   string
		opId      [32]byte
		mockError error
		want      bool
		wantErr   error
	}{
		{
			name:    "IsOperation success",
			address: "0x1234567890abcdef1234567890abcdef12345678",
			opId:    [32]byte{0x01},
			want:    true,
		},
		{
			name:      "IsOperation call contract failure error",
			address:   "0x1234567890abcdef1234567890abcdef12345678",
			opId:      [32]byte{0x02},
			mockError: errors.New("call to contract failed"),
			want:      false,
			wantErr:   errors.New("call to contract failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a new mock client and inspector for each test case
			mockClient := evm_mocks.NewContractDeployBackend(t)
			inspector := NewTimelockEVMInspector(mockClient)

			// Load the ABI for encoding
			parsedABI, err := bindings.RBACTimelockMetaData.GetAbi()
			require.NoError(t, err)

			// Mock the contract call based on the test case
			if tt.mockError == nil {
				// Encode the expected `IsOperation` return value for a successful call
				encodedResult, packErr := parsedABI.Methods["isOperation"].Outputs.Pack(tt.want)
				require.NoError(t, packErr)

				mockClient.EXPECT().CallContract(mock.Anything, mock.IsType(ethereum.CallMsg{}), mock.IsType(&big.Int{})).
					Return(encodedResult, nil).Once()
			} else {
				// Mock a failure for the `IsOperation` call
				mockClient.EXPECT().CallContract(mock.Anything, mock.Anything, mock.Anything).
					Return(nil, tt.mockError).Once()
			}

			// Call the `IsOperation` method
			got, err := inspector.IsOperation(tt.address, tt.opId)

			// Assertions for expected error or successful result
			if tt.wantErr != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.wantErr.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}

			// Verify expectations
			mockClient.AssertExpectations(t)
		})
	}
}

// Helper function to test the various "IsOperation" states
func testIsOperationState(
	t *testing.T,
	methodName string,
	address string,
	opId [32]byte,
	want bool,
	mockError error,
	wantErr error,
) {
	t.Helper()

	// Create a new mock client and inspector for each test case
	mockClient := evm_mocks.NewContractDeployBackend(t)
	inspector := NewTimelockEVMInspector(mockClient)

	// Load the ABI for encoding
	parsedABI, err := bindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err)

	// Mock the contract call based on the test case
	if mockError == nil {
		// Encode the expected return value for a successful call
		encodedResult, packErr := parsedABI.Methods[methodName].Outputs.Pack(want)
		require.NoError(t, packErr)

		mockClient.EXPECT().CallContract(mock.Anything, mock.IsType(ethereum.CallMsg{}), mock.IsType(&big.Int{})).
			Return(encodedResult, nil).Once()
	} else {
		// Mock a failure for the contract call
		mockClient.EXPECT().CallContract(mock.Anything, mock.Anything, mock.Anything).
			Return(nil, mockError).Once()
	}

	// Call the respective method based on methodName
	var got bool
	switch methodName {
	case "isOperationPending":
		got, err = inspector.IsOperationPending(address, opId)
	case "isOperationReady":
		got, err = inspector.IsOperationReady(address, opId)
	case "isOperationDone":
		got, err = inspector.IsOperationDone(address, opId)
	default:
		t.Fatalf("unsupported methodName: %s", methodName)
	}

	// Assertions for expected error or successful result
	if wantErr != nil {
		require.Error(t, err)
		require.EqualError(t, err, wantErr.Error())
	} else {
		require.NoError(t, err)
		require.Equal(t, want, got)
	}

	// Verify expectations
	mockClient.AssertExpectations(t)
}

// Individual test functions calling the helper function with specific method names
func TestTimelockEVMInspector_IsOperationPending(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		address   string
		opId      [32]byte
		want      bool
		mockError error
		wantErr   error
	}{
		{
			name:    "IsOperationPending success",
			address: "0x1234567890abcdef1234567890abcdef12345678",
			opId:    [32]byte{0x01},
			want:    true,
		},
		{
			name:      "IsOperationPending call contract failure error",
			address:   "0x1234567890abcdef1234567890abcdef12345678",
			opId:      [32]byte{0x02},
			mockError: errors.New("call to contract failed"),
			want:      false,
			wantErr:   errors.New("call to contract failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testIsOperationState(t, "isOperationPending", tt.address, tt.opId, tt.want, tt.mockError, tt.wantErr)
		})
	}
}

func TestTimelockEVMInspector_IsOperationReady(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		address   string
		opId      [32]byte
		want      bool
		mockError error
		wantErr   error
	}{
		{
			name:    "IsOperationReady success",
			address: "0x1234567890abcdef1234567890abcdef12345678",
			opId:    [32]byte{0x01},
			want:    true,
		},
		{
			name:      "IsOperationReady call contract failure error",
			address:   "0x1234567890abcdef1234567890abcdef12345678",
			opId:      [32]byte{0x02},
			mockError: errors.New("call to contract failed"),
			want:      false,
			wantErr:   errors.New("call to contract failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testIsOperationState(t, "isOperationReady", tt.address, tt.opId, tt.want, tt.mockError, tt.wantErr)
		})
	}
}

func TestTimelockEVMInspector_IsOperationDone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		address   string
		opId      [32]byte
		want      bool
		mockError error
		wantErr   error
	}{
		{
			name:    "IsOperationDone success",
			address: "0x1234567890abcdef1234567890abcdef12345678",
			opId:    [32]byte{0x01},
			want:    true,
		},
		{
			name:      "IsOperationDone call contract failure error",
			address:   "0x1234567890abcdef1234567890abcdef12345678",
			opId:      [32]byte{0x02},
			mockError: errors.New("call to contract failed"),
			want:      false,
			wantErr:   errors.New("call to contract failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testIsOperationState(t, "isOperationDone", tt.address, tt.opId, tt.want, tt.mockError, tt.wantErr)
		})
	}
}
