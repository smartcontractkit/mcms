package sui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	mockbindutils "github.com/smartcontractkit/mcms/sdk/sui/mocks/bindutils"
	mockmodulemcms "github.com/smartcontractkit/mcms/sdk/sui/mocks/mcms"
	mocksui "github.com/smartcontractkit/mcms/sdk/sui/mocks/sui"
)

func TestNewTimelockInspector(t *testing.T) {
	t.Parallel()
	mockClient := mocksui.NewISuiAPI(t)
	mockSigner := mockbindutils.NewSuiSigner(t)
	mcmsPackageID := "0x123456789abcdef"

	inspector, err := NewTimelockInspector(mockClient, mockSigner, mcmsPackageID)
	require.NoError(t, err)
	assert.NotNil(t, inspector)
	assert.Equal(t, mockClient, inspector.client)
	assert.Equal(t, mockSigner, inspector.signer)
	assert.Equal(t, mcmsPackageID, inspector.mcmsPackageID)
	assert.NotNil(t, inspector.mcms)
}

func TestTimelockInspectorGetProposers(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	mockClient := mocksui.NewISuiAPI(t)
	mockSigner := mockbindutils.NewSuiSigner(t)

	inspector, err := NewTimelockInspector(mockClient, mockSigner, "0x123456789abcdef")
	require.NoError(t, err)

	result, err := inspector.GetProposers(ctx, "0x123")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unsupported on Sui")
}

func TestTimelockInspectorGetExecutors(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	mockClient := mocksui.NewISuiAPI(t)
	mockSigner := mockbindutils.NewSuiSigner(t)

	inspector, err := NewTimelockInspector(mockClient, mockSigner, "0x123456789abcdef")
	require.NoError(t, err)

	result, err := inspector.GetExecutors(ctx, "0x123")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unsupported on Sui")
}

func TestTimelockInspectorGetBypassers(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	mockClient := mocksui.NewISuiAPI(t)
	mockSigner := mockbindutils.NewSuiSigner(t)

	inspector, err := NewTimelockInspector(mockClient, mockSigner, "0x123456789abcdef")
	require.NoError(t, err)

	result, err := inspector.GetBypassers(ctx, "0x123")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unsupported on Sui")
}

func TestTimelockInspectorGetCancellers(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	mockClient := mocksui.NewISuiAPI(t)
	mockSigner := mockbindutils.NewSuiSigner(t)

	inspector, err := NewTimelockInspector(mockClient, mockSigner, "0x123456789abcdef")
	require.NoError(t, err)

	result, err := inspector.GetCancellers(ctx, "0x123")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unsupported on Sui")
}

func TestTimelockInspectorGetMinDelay(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	mockClient := mocksui.NewISuiAPI(t)
	mockSigner := mockbindutils.NewSuiSigner(t)

	// Create a mock MCMS contract
	mockmcms := mockmodulemcms.NewIMcms(t)

	// Create a mock DevInspect
	mockDevInspect := mockmodulemcms.NewIMcmsDevInspect(t)

	// Set up the mock expectations
	mockmcms.EXPECT().DevInspect().Return(mockDevInspect)
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
		mcms:          mockmcms,
	}

	result, err := inspector.GetMinDelay(ctx, "0x123")
	require.NoError(t, err)
	assert.Equal(t, uint64(600), result)
}

func TestTimelockInspectorIsOperation(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	mockClient := mocksui.NewISuiAPI(t)
	mockSigner := mockbindutils.NewSuiSigner(t)

	// Create a mock MCMS contract
	mockmcms := mockmodulemcms.NewIMcms(t)

	// Create a mock DevInspect
	mockDevInspect := mockmodulemcms.NewIMcmsDevInspect(t)

	// Set up the mock expectations
	mockmcms.EXPECT().DevInspect().Return(mockDevInspect)
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
		mcms:          mockmcms,
	}

	opID := [32]byte{1, 2, 3, 4, 5}
	result, err := inspector.IsOperation(ctx, "0x123", opID)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestTimelockInspectorIsOperationPending(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	mockClient := mocksui.NewISuiAPI(t)
	mockSigner := mockbindutils.NewSuiSigner(t)

	// Create a mock MCMS contract
	mockmcms := mockmodulemcms.NewIMcms(t)

	// Create a mock DevInspect
	mockDevInspect := mockmodulemcms.NewIMcmsDevInspect(t)

	// Set up the mock expectations
	mockmcms.EXPECT().DevInspect().Return(mockDevInspect)
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
		mcms:          mockmcms,
	}

	opID := [32]byte{1, 2, 3, 4, 5}
	result, err := inspector.IsOperationPending(ctx, "0x123", opID)
	require.NoError(t, err)
	assert.False(t, result)
}

func TestTimelockInspectorIsOperationReady(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	mockClient := mocksui.NewISuiAPI(t)
	mockSigner := mockbindutils.NewSuiSigner(t)

	// Create a mock MCMS contract
	mockmcms := mockmodulemcms.NewIMcms(t)

	// Create a mock DevInspect
	mockDevInspect := mockmodulemcms.NewIMcmsDevInspect(t)

	// Set up the mock expectations
	mockmcms.EXPECT().DevInspect().Return(mockDevInspect)
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
		mcms:          mockmcms,
	}

	opID := [32]byte{1, 2, 3, 4, 5}
	result, err := inspector.IsOperationReady(ctx, "0x123", opID)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestTimelockInspectorIsOperationDone(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	mockClient := mocksui.NewISuiAPI(t)
	mockSigner := mockbindutils.NewSuiSigner(t)

	// Create a mock MCMS contract
	mockmcms := mockmodulemcms.NewIMcms(t)

	// Create a mock DevInspect
	mockDevInspect := mockmodulemcms.NewIMcmsDevInspect(t)

	// Set up the mock expectations
	mockmcms.EXPECT().DevInspect().Return(mockDevInspect)
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
		mcms:          mockmcms,
	}

	opID := [32]byte{1, 2, 3, 4, 5}
	result, err := inspector.IsOperationDone(ctx, "0x123", opID)
	require.NoError(t, err)
	assert.False(t, result)
}
