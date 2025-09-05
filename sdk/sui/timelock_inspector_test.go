package sui

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	mockBindUtils "github.com/smartcontractkit/mcms/sdk/sui/mocks/bindutils"
	mockModuleMcms "github.com/smartcontractkit/mcms/sdk/sui/mocks/mcms"
	mockSui "github.com/smartcontractkit/mcms/sdk/sui/mocks/sui"
)

func TestNewTimelockInspector(t *testing.T) {
	t.Parallel()
	mockClient := mockSui.NewISuiAPI(t)
	mockSigner := mockBindUtils.NewSuiSigner(t)
	mcmsPackageID := "0x123456789abcdef"

	inspector, err := NewTimelockInspector(mockClient, mockSigner, mcmsPackageID)
	require.NoError(t, err)
	assert.NotNil(t, inspector)
	assert.Equal(t, mockClient, inspector.client)
	assert.Equal(t, mockSigner, inspector.signer)
	assert.Equal(t, mcmsPackageID, inspector.mcmsPackageID)
	assert.NotNil(t, inspector.mcms)
}

func TestTimelockInspector_GetProposers(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockClient := mockSui.NewISuiAPI(t)
	mockSigner := mockBindUtils.NewSuiSigner(t)

	inspector, err := NewTimelockInspector(mockClient, mockSigner, "0x123456789abcdef")
	require.NoError(t, err)

	result, err := inspector.GetProposers(ctx, "0x123")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unsupported on Sui")
}

func TestTimelockInspector_GetExecutors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockClient := mockSui.NewISuiAPI(t)
	mockSigner := mockBindUtils.NewSuiSigner(t)

	inspector, err := NewTimelockInspector(mockClient, mockSigner, "0x123456789abcdef")
	require.NoError(t, err)

	result, err := inspector.GetExecutors(ctx, "0x123")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unsupported on Sui")
}

func TestTimelockInspector_GetBypassers(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockClient := mockSui.NewISuiAPI(t)
	mockSigner := mockBindUtils.NewSuiSigner(t)

	inspector, err := NewTimelockInspector(mockClient, mockSigner, "0x123456789abcdef")
	require.NoError(t, err)

	result, err := inspector.GetBypassers(ctx, "0x123")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unsupported on Sui")
}

func TestTimelockInspector_GetCancellers(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockClient := mockSui.NewISuiAPI(t)
	mockSigner := mockBindUtils.NewSuiSigner(t)

	inspector, err := NewTimelockInspector(mockClient, mockSigner, "0x123456789abcdef")
	require.NoError(t, err)

	result, err := inspector.GetCancellers(ctx, "0x123")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unsupported on Sui")
}

func TestTimelockInspector_GetMinDelay(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockClient := mockSui.NewISuiAPI(t)
	mockSigner := mockBindUtils.NewSuiSigner(t)

	// Create a mock MCMS contract
	mockMcms := mockModuleMcms.NewIMcms(t)

	// Create a mock DevInspect
	mockDevInspect := mockModuleMcms.NewIMcmsDevInspect(t)

	// Set up the mock expectations
	mockMcms.EXPECT().DevInspect().Return(mockDevInspect)
	mockDevInspect.EXPECT().TimelockMinDelay(
		mock.Anything, // context
		mock.Anything, // *bind.CallOpts
		mock.Anything, // bind.Object
	).Return(uint64(600), nil)

	// Create the inspector with the mock
	inspector := &TimelockInspector{
		client:        mockClient,
		signer:        mockSigner,
		mcmsPackageID: "0x123456789abcdef",
		mcms:          mockMcms,
	}

	result, err := inspector.GetMinDelay(ctx, "0x123")
	require.NoError(t, err)
	assert.Equal(t, uint64(600), result)
}

func TestTimelockInspector_IsOperation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockClient := mockSui.NewISuiAPI(t)
	mockSigner := mockBindUtils.NewSuiSigner(t)

	// Create a mock MCMS contract
	mockMcms := mockModuleMcms.NewIMcms(t)

	// Create a mock DevInspect
	mockDevInspect := mockModuleMcms.NewIMcmsDevInspect(t)

	// Set up the mock expectations
	mockMcms.EXPECT().DevInspect().Return(mockDevInspect)
	mockDevInspect.EXPECT().TimelockIsOperation(
		mock.Anything, // context
		mock.Anything, // *bind.CallOpts
		mock.Anything, // bind.Object
		mock.Anything, // []byte (opID)
	).Return(true, nil)

	// Create the inspector with the mock
	inspector := &TimelockInspector{
		client:        mockClient,
		signer:        mockSigner,
		mcmsPackageID: "0x123456789abcdef",
		mcms:          mockMcms,
	}

	opID := [32]byte{1, 2, 3, 4, 5}
	result, err := inspector.IsOperation(ctx, "0x123", opID)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestTimelockInspector_IsOperationPending(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockClient := mockSui.NewISuiAPI(t)
	mockSigner := mockBindUtils.NewSuiSigner(t)

	// Create a mock MCMS contract
	mockMcms := mockModuleMcms.NewIMcms(t)

	// Create a mock DevInspect
	mockDevInspect := mockModuleMcms.NewIMcmsDevInspect(t)

	// Set up the mock expectations
	mockMcms.EXPECT().DevInspect().Return(mockDevInspect)
	mockDevInspect.EXPECT().TimelockIsOperationPending(
		mock.Anything, // context
		mock.Anything, // *bind.CallOpts
		mock.Anything, // bind.Object
		mock.Anything, // []byte (opID)
	).Return(false, nil)

	// Create the inspector with the mock
	inspector := &TimelockInspector{
		client:        mockClient,
		signer:        mockSigner,
		mcmsPackageID: "0x123456789abcdef",
		mcms:          mockMcms,
	}

	opID := [32]byte{1, 2, 3, 4, 5}
	result, err := inspector.IsOperationPending(ctx, "0x123", opID)
	require.NoError(t, err)
	assert.False(t, result)
}

func TestTimelockInspector_IsOperationReady(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockClient := mockSui.NewISuiAPI(t)
	mockSigner := mockBindUtils.NewSuiSigner(t)

	// Create a mock MCMS contract
	mockMcms := mockModuleMcms.NewIMcms(t)

	// Create a mock DevInspect
	mockDevInspect := mockModuleMcms.NewIMcmsDevInspect(t)

	// Set up the mock expectations
	mockMcms.EXPECT().DevInspect().Return(mockDevInspect)
	mockDevInspect.EXPECT().TimelockIsOperationReady(
		mock.Anything, // context
		mock.Anything, // *bind.CallOpts
		mock.Anything, // bind.Object (timelock)
		mock.Anything, // bind.Object (clock)
		mock.Anything, // []byte (opID)
	).Return(true, nil)

	// Create the inspector with the mock
	inspector := &TimelockInspector{
		client:        mockClient,
		signer:        mockSigner,
		mcmsPackageID: "0x123456789abcdef",
		mcms:          mockMcms,
	}

	opID := [32]byte{1, 2, 3, 4, 5}
	result, err := inspector.IsOperationReady(ctx, "0x123", opID)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestTimelockInspector_IsOperationDone(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockClient := mockSui.NewISuiAPI(t)
	mockSigner := mockBindUtils.NewSuiSigner(t)

	// Create a mock MCMS contract
	mockMcms := mockModuleMcms.NewIMcms(t)

	// Create a mock DevInspect
	mockDevInspect := mockModuleMcms.NewIMcmsDevInspect(t)

	// Set up the mock expectations
	mockMcms.EXPECT().DevInspect().Return(mockDevInspect)
	mockDevInspect.EXPECT().TimelockIsOperationDone(
		mock.Anything, // context
		mock.Anything, // *bind.CallOpts
		mock.Anything, // bind.Object
		mock.Anything, // []byte (opID)
	).Return(false, nil)

	// Create the inspector with the mock
	inspector := &TimelockInspector{
		client:        mockClient,
		signer:        mockSigner,
		mcmsPackageID: "0x123456789abcdef",
		mcms:          mockMcms,
	}

	opID := [32]byte{1, 2, 3, 4, 5}
	result, err := inspector.IsOperationDone(ctx, "0x123", opID)
	require.NoError(t, err)
	assert.False(t, result)
}
