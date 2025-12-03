package ton_test

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"

	tonmcms "github.com/smartcontractkit/mcms/sdk/ton"
	ton_mocks "github.com/smartcontractkit/mcms/sdk/ton/mocks"
)

type roleFetchTest struct {
	name            string
	address         string
	roleMemberCount *big.Int
	roleMembers     []*address.Address
	proposerRole    [32]byte
	mockError       error
	want            []string
	wantErr         error
	roleFetchType   string // Specify the role type (proposers, executors, etc.)
}

// Helper to mock contract calls for each role test case
func (tt roleFetchTest) mockRoleContractCalls(t *testing.T, client *ton_mocks.APIClientWrapped) {
	t.Helper()

	// Mock CurrentMasterchainInfo
	client.EXPECT().CurrentMasterchainInfo(mock.Anything).
		Return(&ton.BlockIDExt{}, nil)

	// Mock response for role member count
	encodedRoleMemberCount := tt.roleMemberCount
	r := ton.NewExecutionResult([]any{encodedRoleMemberCount})
	client.EXPECT().RunGetMethod(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(r, nil).Once()

	// Mock response for each role member
	for _, member := range tt.roleMembers {
		encodedMember := cell.BeginCell().MustStoreAddr(member).ToSlice()

		client.EXPECT().RunGetMethod(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(ton.NewExecutionResult([]any{encodedMember}), nil).Once()
	}
}

func TestTimelockInspectorGetRolesTests(t *testing.T) {
	t.Parallel()

	var chainID = chaintest.Chain7TONID
	var client *ton.APIClient
	var wallets = []*wallet.Wallet{
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
	}

	ctx := context.Background()
	tests := []roleFetchTest{
		{
			name:            "GetProposers success",
			address:         "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			roleMemberCount: big.NewInt(3),
			roleMembers: []*address.Address{
				wallets[0].Address(),
				wallets[1].Address(),
				wallets[2].Address(),
			},
			proposerRole: [32]byte{0x01},
			want: []string{
				wallets[0].Address().String(),
				wallets[1].Address().String(),
				wallets[2].Address().String(),
			},
			roleFetchType: "proposers",
		},
		{
			name:          "GetProposers call contract failure error",
			address:       "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			mockError:     errors.New("call to contract failed"),
			want:          nil,
			wantErr:       errors.New("error getting getRoleMemberCount: call to contract failed"),
			roleFetchType: "proposers",
		},
		{
			name:            "GetExecutors success",
			address:         "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			roleMemberCount: big.NewInt(3),
			roleMembers: []*address.Address{
				wallets[0].Address(),
				wallets[1].Address(),
				wallets[2].Address(),
			},
			proposerRole: [32]byte{0x01},
			want: []string{
				wallets[0].Address().String(),
				wallets[1].Address().String(),
				wallets[2].Address().String(),
			},
			roleFetchType: "executors",
		},
		{
			name:          "GetExecutors call contract failure error",
			address:       "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			mockError:     errors.New("call to contract failed"),
			want:          nil,
			wantErr:       errors.New("error getting getRoleMemberCount: call to contract failed"),
			roleFetchType: "executors",
		},
		{
			name:            "GetExecutors success",
			address:         "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			roleMemberCount: big.NewInt(3),
			roleMembers: []*address.Address{
				wallets[0].Address(),
				wallets[1].Address(),
				wallets[2].Address(),
			},
			proposerRole: [32]byte{0x01},
			want: []string{
				wallets[0].Address().String(),
				wallets[1].Address().String(),
				wallets[2].Address().String(),
			},
			roleFetchType: "executors",
		},
		{
			name:          "GetExecutors call contract failure error",
			address:       "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			mockError:     errors.New("call to contract failed"),
			want:          nil,
			wantErr:       errors.New("error getting getRoleMemberCount: call to contract failed"),
			roleFetchType: "executors",
		},
		{
			name:            "GetBypassers success",
			address:         "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			roleMemberCount: big.NewInt(3),
			roleMembers: []*address.Address{
				wallets[0].Address(),
				wallets[1].Address(),
				wallets[2].Address(),
			},
			proposerRole: [32]byte{0x01},
			want: []string{
				wallets[0].Address().String(),
				wallets[1].Address().String(),
				wallets[2].Address().String(),
			},
			roleFetchType: "bypassers",
		},
		{
			name:          "GetBypassers call contract failure error",
			address:       "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			mockError:     errors.New("call to contract failed"),
			want:          nil,
			wantErr:       errors.New("error getting getRoleMemberCount: call to contract failed"),
			roleFetchType: "bypassers",
		},
		{
			name:            "GetCancellers success",
			address:         "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			roleMemberCount: big.NewInt(3),
			roleMembers: []*address.Address{
				wallets[0].Address(),
				wallets[1].Address(),
				wallets[2].Address(),
			},
			proposerRole: [32]byte{0x01},
			want: []string{
				wallets[0].Address().String(),
				wallets[1].Address().String(),
				wallets[2].Address().String(),
			},
			roleFetchType: "bypassers",
		},
		{
			name:          "GetCancellers call contract failure error",
			address:       "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			mockError:     errors.New("call to contract failed"),
			want:          nil,
			wantErr:       errors.New("error getting getRoleMemberCount: call to contract failed"),
			roleFetchType: "cancellers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a new mock client and inspector for each test case
			client := ton_mocks.NewAPIClientWrapped(t)
			inspector := tonmcms.NewTimelockInspector(client)

			// Mock the contract calls based on the test case
			if tt.mockError == nil {
				tt.mockRoleContractCalls(t, client)
			} else {
				// Mock CurrentMasterchainInfo
				client.EXPECT().CurrentMasterchainInfo(mock.Anything).
					Return(&ton.BlockIDExt{}, nil)

				// If there's an error, mock it on the first CallContract call
				client.EXPECT().RunGetMethod(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, tt.mockError).Once()
			}

			// Select and call the appropriate role-fetching function
			var got []string
			var err error
			switch tt.roleFetchType {
			case "proposers":
				got, err = inspector.GetProposers(ctx, tt.address)
			case "executors":
				got, err = inspector.GetExecutors(ctx, tt.address)
			case "cancellers":
				got, err = inspector.GetCancellers(ctx, tt.address)
			case "bypassers":
				got, err = inspector.GetBypassers(ctx, tt.address)
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
			client.AssertExpectations(t)
		})
	}
}

func TestTimelockInspectorIsOperation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name      string
		address   string
		opID      [32]byte
		mockError error
		want      bool
		wantErr   error
	}{
		{
			name:    "IsOperation success",
			address: "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			opID:    [32]byte{0x01},
			want:    true,
		},
		{
			name:      "IsOperation call contract failure error",
			address:   "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			opID:      [32]byte{0x02},
			mockError: errors.New("call to contract failed"),
			want:      false,
			wantErr:   errors.New("error getting isOperation: call to contract failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a new mock client and inspector for each test case
			client := ton_mocks.NewAPIClientWrapped(t)
			inspector := tonmcms.NewTimelockInspector(client)

			// Mock the contract call based on the test case
			// Mock CurrentMasterchainInfo
			client.EXPECT().CurrentMasterchainInfo(mock.Anything).
				Return(&ton.BlockIDExt{}, nil)

			if tt.mockError == nil {
				// Encode the expected `IsOperation` return value for a successful call
				wantInt := 0
				if tt.want {
					wantInt = 1
				}
				r := ton.NewExecutionResult([]any{big.NewInt(int64(wantInt))})
				client.EXPECT().RunGetMethod(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(r, nil).Once()
			} else {
				// If there's an error, mock it on the first CallContract call
				client.EXPECT().RunGetMethod(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, tt.mockError).Once()
			}

			// Call the `IsOperation` method
			got, err := inspector.IsOperation(ctx, tt.address, tt.opID)

			// Assertions for expected error or successful result
			if tt.wantErr != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.wantErr.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}

			// Verify expectations
			client.AssertExpectations(t)
		})
	}
}

// Helper function to test the various "IsOperation" states
func testIsOperationState(
	t *testing.T,
	methodName string,
	addr string,
	opID [32]byte,
	want bool,
	mockError error,
	wantErr error,
) {
	t.Helper()

	ctx := context.Background()

	// Create a new mock client and inspector for each test case
	client := ton_mocks.NewAPIClientWrapped(t)
	inspector := tonmcms.NewTimelockInspector(client)

	// Mock the contract call based on the test case
	// Mock CurrentMasterchainInfo
	client.EXPECT().CurrentMasterchainInfo(mock.Anything).
		Return(&ton.BlockIDExt{}, nil)

	if mockError == nil {
		// Encode the expected `IsOperation` return value for a successful call
		wantInt := 0
		if want {
			wantInt = 1
		}
		r := ton.NewExecutionResult([]any{big.NewInt(int64(wantInt))})
		client.EXPECT().RunGetMethod(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(r, nil).Once()
	} else {
		// If there's an error, mock it on the first CallContract call
		client.EXPECT().RunGetMethod(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, mockError).Once()
	}

	// Call the respective method based on methodName
	var got bool
	var err error
	switch methodName {
	case "isOperationPending":
		got, err = inspector.IsOperationPending(ctx, addr, opID)
	case "isOperationReady":
		got, err = inspector.IsOperationReady(ctx, addr, opID)
	case "isOperationDone":
		got, err = inspector.IsOperationDone(ctx, addr, opID)
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
	client.AssertExpectations(t)
}

// Individual test functions calling the helper function with specific method names
func TestTimelockInspectorIsOperationPending(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		address   string
		opID      [32]byte
		want      bool
		mockError error
		wantErr   error
	}{
		{
			name:    "IsOperationPending success",
			address: "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			opID:    [32]byte{0x01},
			want:    true,
		},
		{
			name:      "IsOperationPending call contract failure error",
			address:   "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			opID:      [32]byte{0x02},
			mockError: errors.New("call to contract failed"),
			want:      false,
			wantErr:   errors.New("error getting isOperationPending: call to contract failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testIsOperationState(t, "isOperationPending", tt.address, tt.opID, tt.want, tt.mockError, tt.wantErr)
		})
	}
}

func TestTimelockInspectorIsOperationReady(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		address   string
		opID      [32]byte
		want      bool
		mockError error
		wantErr   error
	}{
		{
			name:    "IsOperationReady success",
			address: "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			opID:    [32]byte{0x01},
			want:    true,
		},
		{
			name:      "IsOperationReady call contract failure error",
			address:   "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			opID:      [32]byte{0x02},
			mockError: errors.New("call to contract failed"),
			want:      false,
			wantErr:   errors.New("error getting isOperationReady: call to contract failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testIsOperationState(t, "isOperationReady", tt.address, tt.opID, tt.want, tt.mockError, tt.wantErr)
		})
	}
}

func TestTimelockInspectorIsOperationDone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		address   string
		opID      [32]byte
		want      bool
		mockError error
		wantErr   error
	}{
		{
			name:    "IsOperationDone success",
			address: "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			opID:    [32]byte{0x01},
			want:    true,
		},
		{
			name:      "IsOperationDone call contract failure error",
			address:   "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			opID:      [32]byte{0x02},
			mockError: errors.New("call to contract failed"),
			want:      false,
			wantErr:   errors.New("error getting isOperationDone: call to contract failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testIsOperationState(t, "isOperationDone", tt.address, tt.opID, tt.want, tt.mockError, tt.wantErr)
		})
	}
}

func TestTimelockInspectorGetMinDelay(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name      string
		address   string
		minDelay  *big.Int
		mockError error
		want      uint64
		wantErr   error
	}{
		{
			name:     "GetMinDelay success",
			address:  "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			minDelay: big.NewInt(300),
			want:     300,
		},
		{
			name:      "GetMinDelay call contract failure error",
			address:   "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			mockError: errors.New("call to contract failed"),
			want:      0,
			wantErr:   errors.New("error getting getMinDelay: call to contract failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a new mock client and inspector for each test case
			client := ton_mocks.NewAPIClientWrapped(t)
			inspector := tonmcms.NewTimelockInspector(client)

			// Mock CurrentMasterchainInfo
			client.EXPECT().CurrentMasterchainInfo(mock.Anything).
				Return(&ton.BlockIDExt{}, nil)

			if tt.mockError == nil {
				// Encode the expected `getMinDelay` return value for a successful call
				r := ton.NewExecutionResult([]any{tt.minDelay})
				client.EXPECT().RunGetMethod(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(r, nil).Once()
			} else {
				// Simulate a low-level call failure
				client.EXPECT().RunGetMethod(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, tt.mockError).Once()
			}

			// Act
			got, err := inspector.GetMinDelay(ctx, tt.address)

			// Assert
			if tt.wantErr != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.wantErr.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}

			client.AssertExpectations(t)
		})
	}
}
