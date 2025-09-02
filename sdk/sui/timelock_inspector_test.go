package sui

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mockBindUtils "github.com/smartcontractkit/mcms/sdk/sui/mocks/bindutils"
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
