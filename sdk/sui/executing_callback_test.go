package sui

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/block-vision/sui-go-sdk/transaction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-sui/bindings/bind"

	mockbindutils "github.com/smartcontractkit/mcms/sdk/sui/mocks/bindutils"
	mockfeequoter "github.com/smartcontractkit/mcms/sdk/sui/mocks/feequoter"
	mockmcms "github.com/smartcontractkit/mcms/sdk/sui/mocks/mcms"
	mocksui "github.com/smartcontractkit/mcms/sdk/sui/mocks/sui"
)

func TestNewExecutingCallbackParams(t *testing.T) {
	t.Parallel()

	// Create mock objects
	mockClient := mocksui.NewISuiAPI(t)
	mockMcms := mockmcms.NewIMcms(t)
	mockFeeQuoterEncoder := mockfeequoter.NewFeeQuoterEncoder(t)

	// Basic test to ensure the constructor works
	params := NewExecutingCallbackParams(
		mockClient,
		mockMcms,
		"0x123456789abcdef",
		mockFeeQuoterEncoder,
		"0xregistry",
		"0xaccount",
	)

	require.NotNil(t, params)
	assert.Equal(t, "0x123456789abcdef", params.mcmsPackageID)
	assert.Equal(t, "0xregistry", params.registryObj)
	assert.Equal(t, "0xaccount", params.accountObj)
	assert.Equal(t, mockClient, params.client)
	assert.Equal(t, mockMcms, params.mcms)
	assert.Equal(t, mockFeeQuoterEncoder, params.entryPointContractEncoder)
}

func TestExecutingCallbackParams_AppendPTB_WithMCMSPackageTarget(t *testing.T) {
	t.Parallel()

	// Create mock objects
	mockClient := mocksui.NewISuiAPI(t)
	mockMcms := mockmcms.NewIMcms(t)
	mockFeeQuoterEncoder := mockfeequoter.NewFeeQuoterEncoder(t)
	mockMcmsEncoder := mockmcms.NewMcmsEncoder(t)
	mockBound := mockbindutils.NewIBoundContract(t)

	// Setup mock expectations
	mockMcms.EXPECT().Encoder().Return(mockMcmsEncoder)
	mockMcms.EXPECT().Bound().Return(mockBound)

	// Mock the ExecuteDispatchToAccountWithArgs call
	expectedCall := &bind.EncodedCall{
		Module: bind.ModuleInformation{
			PackageID:   "0x123456789abcdef",
			PackageName: "mcms",
			ModuleName:  "mcms",
		},
		Function: "execute_dispatch_to_account",
	}
	mockMcmsEncoder.EXPECT().ExecuteDispatchToAccountWithArgs(
		"0xregistry",
		"0xaccount",
		mock.AnythingOfType("*transaction.Argument"),
	).Return(expectedCall, nil)

	// Mock the AppendPTB call
	mockBound.EXPECT().AppendPTB(
		mock.Anything,
		mock.AnythingOfType("*bind.CallOpts"),
		mock.AnythingOfType("*transaction.Transaction"),
		expectedCall,
	).Return(nil, nil)

	mcmsPackageIDHex := "123456789abcdef0" + strings.Repeat("0", 48)
	mcmsPackageIDBytes, err := hex.DecodeString(mcmsPackageIDHex)
	require.NoError(t, err)
	mcmsPackageID := "0x" + mcmsPackageIDHex

	// Create the ExecutingCallbackParams
	params := NewExecutingCallbackParams(
		mockClient,
		mockMcms,
		mcmsPackageID,
		mockFeeQuoterEncoder,
		"0xregistry",
		"0xaccount",
	)

	// Mock the helper functions
	params.extractExecutingCallbackParams = func(mcmsPackageID string, ptb *transaction.Transaction, vectorExecutingCallback *transaction.Argument) (*transaction.Argument, error) {
		// Return a mock transaction.Argument
		return &transaction.Argument{}, nil
	}
	params.closeExecutingCallbackParams = func(mcmsPackageID string, ptb *transaction.Transaction, vectorExecutingCallback *transaction.Argument) error {
		// Just return success
		return nil
	}

	// Create test data
	ctx := context.Background()
	ptb := &transaction.Transaction{}
	executeCallback := &transaction.Argument{}

	// Create a call that targets the MCMS package (should trigger ExecuteDispatchToAccount)
	calls := []Call{
		{
			Target:       mcmsPackageIDBytes,
			StateObj:     "0xstate",
			ModuleName:   "test_module",
			FunctionName: "test_function",
		},
	}

	// Execute the method
	err = params.AppendPTB(ctx, ptb, executeCallback, calls)
	assert.NoError(t, err)
}

func TestExecutingCallbackParams_AppendPTB_WithNonMCMSTarget(t *testing.T) {
	t.Parallel()

	// Create mock objects
	mockClient := mocksui.NewISuiAPI(t)
	mockMcms := mockmcms.NewIMcms(t)
	mockFeeQuoterEncoder := mockfeequoter.NewFeeQuoterEncoder(t)
	mockBound := mockbindutils.NewIBoundContract(t)

	// Setup mock expectations for the non-MCMS target path
	mockMcms.EXPECT().Bound().Return(mockBound)

	// Mock the McmsApplyFeeTokenUpdatesWithArgs call
	expectedCall := &bind.EncodedCall{
		Module: bind.ModuleInformation{
			PackageID:   "0xdifferent",
			PackageName: "destination_package",
			ModuleName:  "destination_module",
		},
		Function: "mcms_test_function",
	}
	mockFeeQuoterEncoder.EXPECT().McmsApplyFeeTokenUpdatesWithArgs(
		"0xstate",
		"0xregistry",
		mock.AnythingOfType("*transaction.Argument"),
	).Return(expectedCall, nil)

	// Mock the AppendPTB call
	mockBound.EXPECT().AppendPTB(
		mock.Anything,
		mock.AnythingOfType("*bind.CallOpts"),
		mock.AnythingOfType("*transaction.Transaction"),
		expectedCall,
	).Return(nil, nil)

	// Create the ExecutingCallbackParams
	params := NewExecutingCallbackParams(
		mockClient,
		mockMcms,
		"0x123456789abcdef",
		mockFeeQuoterEncoder,
		"0xregistry",
		"0xaccount",
	)

	// Mock the helper functions
	params.extractExecutingCallbackParams = func(mcmsPackageID string, ptb *transaction.Transaction, vectorExecutingCallback *transaction.Argument) (*transaction.Argument, error) {
		// Return a mock transaction.Argument
		return &transaction.Argument{}, nil
	}
	params.closeExecutingCallbackParams = func(mcmsPackageID string, ptb *transaction.Transaction, vectorExecutingCallback *transaction.Argument) error {
		// Just return success
		return nil
	}

	// Create test data
	ctx := context.Background()
	ptb := &transaction.Transaction{}
	executeCallback := &transaction.Argument{}

	// Create a call that targets a different package (should trigger mcms_ entrypoint)
	calls := []Call{
		{
			Target:       []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, // Different target
			StateObj:     "0xstate",
			ModuleName:   "destination_module",
			FunctionName: "test_function",
		},
	}

	// Execute the method
	err := params.AppendPTB(ctx, ptb, executeCallback, calls)

	// Should now succeed with mocked functions
	assert.NoError(t, err)
}

func TestExecutingCallbackParams_AppendPTB_ExtractError(t *testing.T) {
	t.Parallel()

	// Create mock objects
	mockClient := mocksui.NewISuiAPI(t)
	mockMcms := mockmcms.NewIMcms(t)
	mockFeeQuoterEncoder := mockfeequoter.NewFeeQuoterEncoder(t)

	// Create the ExecutingCallbackParams
	params := NewExecutingCallbackParams(
		mockClient,
		mockMcms,
		"0x123456789abcdef",
		mockFeeQuoterEncoder,
		"0xregistry",
		"0xaccount",
	)

	// Mock the helper functions to return errors
	params.extractExecutingCallbackParams = func(mcmsPackageID string, ptb *transaction.Transaction, vectorExecutingCallback *transaction.Argument) (*transaction.Argument, error) {
		return nil, fmt.Errorf("mock extract error")
	}
	params.closeExecutingCallbackParams = func(mcmsPackageID string, ptb *transaction.Transaction, vectorExecutingCallback *transaction.Argument) error {
		return nil
	}

	// Create test data
	ctx := context.Background()
	ptb := &transaction.Transaction{}
	executeCallback := &transaction.Argument{}
	mcmsTargetBytes, _ := hex.DecodeString("123456789abcdef" + strings.Repeat("0", 50)) // Pad to 32 bytes
	calls := []Call{
		{
			Target:       mcmsTargetBytes,
			StateObj:     "0xstate",
			ModuleName:   "test_module",
			FunctionName: "test_function",
		},
	}

	// Execute the method
	err := params.AppendPTB(ctx, ptb, executeCallback, calls)

	// Should fail with our mock error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "extracting ExecutingCallbackParams")
	assert.Contains(t, err.Error(), "mock extract error")
}
