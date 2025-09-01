package sui

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mock_bindutils "github.com/smartcontractkit/mcms/sdk/sui/mocks/bindutils"
	mock_sui "github.com/smartcontractkit/mcms/sdk/sui/mocks/sui"
)

func TestNewTimelockInspector(t *testing.T) {
	t.Parallel()
	mockClient := mock_sui.NewISuiAPI(t)
	mockSigner := mock_bindutils.NewSuiSigner(t)
	mcmsPackageId := "0x123456789abcdef"

	inspector, err := NewTimelockInspector(mockClient, mockSigner, mcmsPackageId)
	require.NoError(t, err)
	assert.NotNil(t, inspector)
	assert.Equal(t, mockClient, inspector.client)
	assert.Equal(t, mockSigner, inspector.signer)
	assert.Equal(t, mcmsPackageId, inspector.mcmsPackageId)
	assert.NotNil(t, inspector.mcms)
}

func TestTimelockInspector_GetProposers(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockClient := mock_sui.NewISuiAPI(t)
	mockSigner := mock_bindutils.NewSuiSigner(t)

	inspector, err := NewTimelockInspector(mockClient, mockSigner, "0x123456789abcdef")
	require.NoError(t, err)

	result, err := inspector.GetProposers(ctx, "0x123")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unsupported on Sui")
}

func TestTimelockInspector_GetExecutors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockClient := mock_sui.NewISuiAPI(t)
	mockSigner := mock_bindutils.NewSuiSigner(t)

	inspector, err := NewTimelockInspector(mockClient, mockSigner, "0x123456789abcdef")
	require.NoError(t, err)

	result, err := inspector.GetExecutors(ctx, "0x123")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unsupported on Sui")
}

func TestTimelockInspector_GetBypassers(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockClient := mock_sui.NewISuiAPI(t)
	mockSigner := mock_bindutils.NewSuiSigner(t)

	inspector, err := NewTimelockInspector(mockClient, mockSigner, "0x123456789abcdef")
	require.NoError(t, err)

	result, err := inspector.GetBypassers(ctx, "0x123")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unsupported on Sui")
}

func TestTimelockInspector_GetCancellers(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockClient := mock_sui.NewISuiAPI(t)
	mockSigner := mock_bindutils.NewSuiSigner(t)

	inspector, err := NewTimelockInspector(mockClient, mockSigner, "0x123456789abcdef")
	require.NoError(t, err)

	result, err := inspector.GetCancellers(ctx, "0x123")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unsupported on Sui")
}

// Note: The following methods require blockchain interaction and are better suited for integration tests:
// - GetMinDelay(ctx, address) (uint64, error)
// - IsOperation(ctx, address, opID) (bool, error)
// - IsOperationPending(ctx, address, opID) (bool, error)
// - IsOperationReady(ctx, address, opID) (bool, error)
// - IsOperationDone(ctx, address, opID) (bool, error)
//
// These methods use concrete types (*module_mcms.McmsContract) that make mocking difficult
// in unit tests. They should be tested via integration tests with a live Sui network.
