package aptos

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/smartcontractkit/chainlink-aptos/bindings/mcms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	mock_aptossdk "github.com/smartcontractkit/mcms/sdk/aptos/mocks/aptos"
	mock_mcms "github.com/smartcontractkit/mcms/sdk/aptos/mocks/mcms"
	mock_module_mcms "github.com/smartcontractkit/mcms/sdk/aptos/mocks/mcms/mcms"
)

func TestNewTimelockInspector(t *testing.T) {
	t.Parallel()
	mockClient := mock_aptossdk.NewAptosRpcClient(t)

	inspector := NewTimelockInspector(mockClient)
	assert.NotNil(t, inspector)
	assert.Equal(t, mockClient, inspector.client)
	assert.NotNil(t, inspector.bindingFn)
}

func TestTimelockInspector_GetProposers(t *testing.T) {
	t.Parallel()
	_, err := NewTimelockInspector(nil).GetProposers(t.Context(), "")
	assert.ErrorContains(t, err, "unsupported on Aptos")
}

func TestTimelockInspector_GetExecutors(t *testing.T) {
	t.Parallel()
	_, err := NewTimelockInspector(nil).GetExecutors(t.Context(), "")
	assert.ErrorContains(t, err, "unsupported on Aptos")
}

func TestTimelockInspector_GetBypassers(t *testing.T) {
	t.Parallel()
	_, err := NewTimelockInspector(nil).GetBypassers(t.Context(), "")
	assert.ErrorContains(t, err, "unsupported on Aptos")
}

func TestTimelockInspector_GetCancellers(t *testing.T) {
	t.Parallel()
	_, err := NewTimelockInspector(nil).GetCancellers(t.Context(), "")
	assert.ErrorContains(t, err, "unsupported on Aptos")
}

func TestTimelockInspector_IsOperation(t *testing.T) {
	t.Parallel()
	type args struct {
		mcmsAddr string
		opID     [32]byte
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(m *mock_mcms.MCMS)
		want      bool
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			args: args{
				mcmsAddr: "0x123",
				opID:     [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			},
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMSInterface(t)
				m.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().TimelockIsOperation(
					mock.Anything,
					[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				).Return(true, nil)
			},
			want:    true,
			wantErr: assert.NoError,
		}, {
			name: "failure - invalid MCMS address",
			args: args{
				mcmsAddr: "invalidaddress",
			},
			wantErr: AssertErrorContains("parse MCMS address"),
		}, {
			name: "failure - TimelockIsOperation failed",
			args: args{
				mcmsAddr: "0x123",
				opID:     [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			},
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMSInterface(t)
				m.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().TimelockIsOperation(
					mock.Anything,
					[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				).Return(false, errors.New("error during TimelockIsOperation"))
			},
			want:    false,
			wantErr: AssertErrorContains("error during TimelockIsOperation"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mcmsBinding := mock_mcms.NewMCMS(t)
			inspector := TimelockInspector{
				bindingFn: func(mcmsAddress aptos.AccountAddress, _ aptos.AptosRpcClient) mcms.MCMS {
					require.Equal(t, Must(hexToAddress(tt.args.mcmsAddr)), mcmsAddress)
					return mcmsBinding
				},
			}

			if tt.mockSetup != nil {
				tt.mockSetup(mcmsBinding)
			}

			got, err := inspector.IsOperation(t.Context(), tt.args.mcmsAddr, tt.args.opID)
			if !tt.wantErr(t, err, fmt.Sprintf("IsOperation(%q, %q)", tt.args.mcmsAddr, tt.args.opID)) {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTimelockInspector_IsOperationPending(t *testing.T) {
	t.Parallel()
	type args struct {
		mcmsAddr string
		opID     [32]byte
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(m *mock_mcms.MCMS)
		want      bool
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			args: args{
				mcmsAddr: "0x123",
				opID:     [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			},
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMSInterface(t)
				m.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().TimelockIsOperationPending(
					mock.Anything,
					[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				).Return(true, nil)
			},
			want:    true,
			wantErr: assert.NoError,
		}, {
			name: "failure - invalid MCMS address",
			args: args{
				mcmsAddr: "invalidaddress",
			},
			wantErr: AssertErrorContains("parse MCMS address"),
		}, {
			name: "failure - TimelockIsOperationPending failed",
			args: args{
				mcmsAddr: "0x123",
				opID:     [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			},
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMSInterface(t)
				m.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().TimelockIsOperationPending(
					mock.Anything,
					[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				).Return(false, errors.New("error during TimelockIsOperationPending"))
			},
			want:    false,
			wantErr: AssertErrorContains("error during TimelockIsOperationPending"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mcmsBinding := mock_mcms.NewMCMS(t)
			inspector := TimelockInspector{
				bindingFn: func(mcmsAddress aptos.AccountAddress, _ aptos.AptosRpcClient) mcms.MCMS {
					require.Equal(t, Must(hexToAddress(tt.args.mcmsAddr)), mcmsAddress)
					return mcmsBinding
				},
			}

			if tt.mockSetup != nil {
				tt.mockSetup(mcmsBinding)
			}

			got, err := inspector.IsOperationPending(t.Context(), tt.args.mcmsAddr, tt.args.opID)
			if !tt.wantErr(t, err, fmt.Sprintf("IsOperationPending(%q, %q)", tt.args.mcmsAddr, tt.args.opID)) {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTimelockInspector_IsOperationReady(t *testing.T) {
	t.Parallel()
	type args struct {
		mcmsAddr string
		opID     [32]byte
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(m *mock_mcms.MCMS)
		want      bool
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			args: args{
				mcmsAddr: "0x123",
				opID:     [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			},
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMSInterface(t)
				m.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().TimelockIsOperationReady(
					mock.Anything,
					[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				).Return(true, nil)
			},
			want:    true,
			wantErr: assert.NoError,
		}, {
			name: "failure - invalid MCMS address",
			args: args{
				mcmsAddr: "invalidaddress",
			},
			wantErr: AssertErrorContains("parse MCMS address"),
		}, {
			name: "failure - TimelockIsOperationReady failed",
			args: args{
				mcmsAddr: "0x123",
				opID:     [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			},
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMSInterface(t)
				m.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().TimelockIsOperationReady(
					mock.Anything,
					[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				).Return(false, errors.New("error during TimelockIsOperationReady"))
			},
			want:    false,
			wantErr: AssertErrorContains("error during TimelockIsOperationReady"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mcmsBinding := mock_mcms.NewMCMS(t)
			inspector := TimelockInspector{
				bindingFn: func(mcmsAddress aptos.AccountAddress, _ aptos.AptosRpcClient) mcms.MCMS {
					require.Equal(t, Must(hexToAddress(tt.args.mcmsAddr)), mcmsAddress)
					return mcmsBinding
				},
			}

			if tt.mockSetup != nil {
				tt.mockSetup(mcmsBinding)
			}

			got, err := inspector.IsOperationReady(t.Context(), tt.args.mcmsAddr, tt.args.opID)
			if !tt.wantErr(t, err, fmt.Sprintf("IsOperationReady(%q, %q)", tt.args.mcmsAddr, tt.args.opID)) {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTimelockInspector_IsOperationDone(t *testing.T) {
	t.Parallel()
	type args struct {
		mcmsAddr string
		opID     [32]byte
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(m *mock_mcms.MCMS)
		want      bool
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			args: args{
				mcmsAddr: "0x123",
				opID:     [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			},
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMSInterface(t)
				m.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().TimelockIsOperationDone(
					mock.Anything,
					[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				).Return(true, nil)
			},
			want:    true,
			wantErr: assert.NoError,
		}, {
			name: "failure - invalid MCMS address",
			args: args{
				mcmsAddr: "invalidaddress",
			},
			wantErr: AssertErrorContains("parse MCMS address"),
		}, {
			name: "failure - TimelockIsOperationDone failed",
			args: args{
				mcmsAddr: "0x123",
				opID:     [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			},
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMSInterface(t)
				m.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().TimelockIsOperationDone(
					mock.Anything,
					[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				).Return(false, errors.New("error during TimelockIsOperationDone"))
			},
			want:    false,
			wantErr: AssertErrorContains("error during TimelockIsOperationDone"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mcmsBinding := mock_mcms.NewMCMS(t)
			inspector := TimelockInspector{
				bindingFn: func(mcmsAddress aptos.AccountAddress, _ aptos.AptosRpcClient) mcms.MCMS {
					require.Equal(t, Must(hexToAddress(tt.args.mcmsAddr)), mcmsAddress)
					return mcmsBinding
				},
			}

			if tt.mockSetup != nil {
				tt.mockSetup(mcmsBinding)
			}

			got, err := inspector.IsOperationDone(t.Context(), tt.args.mcmsAddr, tt.args.opID)
			if !tt.wantErr(t, err, fmt.Sprintf("IsOperationDone(%q, %q)", tt.args.mcmsAddr, tt.args.opID)) {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTimelockInspector_GetMinDelay(t *testing.T) {
	t.Parallel()

	type args struct {
		mcmsAddr string
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(m *mock_mcms.MCMS)
		want      uint64
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			args: args{
				mcmsAddr: "0x123",
			},
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMSInterface(t)
				m.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().TimelockMinDelay(
					mock.Anything,
				).Return(uint64(321), nil)
			},
			want:    321,
			wantErr: assert.NoError,
		},
		{
			name: "failure - invalid MCMS address",
			args: args{
				mcmsAddr: "invalidaddress",
			},
			wantErr: AssertErrorContains("parse MCMS address"),
		},
		{
			name: "failure - TimelockMinDelay failed",
			args: args{
				mcmsAddr: "0x123",
			},
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMSInterface(t)
				m.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().TimelockMinDelay(
					mock.Anything,
				).Return(uint64(0), errors.New("error during TimelockMinDelay"))
			},
			want:    0,
			wantErr: AssertErrorContains("error during TimelockMinDelay"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mcmsBinding := mock_mcms.NewMCMS(t)
			inspector := TimelockInspector{
				bindingFn: func(mcmsAddress aptos.AccountAddress, _ aptos.AptosRpcClient) mcms.MCMS {
					// only validate hex address if parsing was supposed to succeed
					if tt.name != "failure - invalid MCMS address" {
						require.Equal(t, Must(hexToAddress(tt.args.mcmsAddr)), mcmsAddress)
					}
					return mcmsBinding
				},
			}

			if tt.mockSetup != nil {
				tt.mockSetup(mcmsBinding)
			}

			got, err := inspector.GetMinDelay(t.Context(), tt.args.mcmsAddr)
			if !tt.wantErr(t, err, fmt.Sprintf("GetMinDelay(%q)", tt.args.mcmsAddr)) {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
