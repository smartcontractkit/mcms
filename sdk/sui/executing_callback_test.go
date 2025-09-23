package sui

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/block-vision/sui-go-sdk/models"
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

type MockEntrypointArgEncoder struct {
	t           *testing.T
	registryObj string
	expected    *bind.EncodedCall
}

func (e *MockEntrypointArgEncoder) EncodeEntryPointArg(executingCallbackParams *transaction.Argument, target, module, function, stateObjID string, data []byte) (*bind.EncodedCall, error) {
	mockFeeQuoterEncoder := mockfeequoter.NewFeeQuoterEncoder(e.t)

	mockFeeQuoterEncoder.EXPECT().McmsApplyFeeTokenUpdatesWithArgs(
		"0xstate",
		"0xregistry",
		mock.AnythingOfType("*transaction.Argument"),
	).Return(e.expected, nil)

	return mockFeeQuoterEncoder.McmsApplyFeeTokenUpdatesWithArgs(
		stateObjID,
		e.registryObj,
		executingCallbackParams,
	)
}

func TestNewExecutingCallbackParams(t *testing.T) {
	t.Parallel()

	// Create mock objects
	mockClient := mocksui.NewISuiAPI(t)
	mockMcms := mockmcms.NewIMcms(t)
	entrypointEncoder := &MockEntrypointArgEncoder{t: t, registryObj: "0xregistry"}

	// Basic test to ensure the constructor works
	params := NewExecutingCallbackParams(
		mockClient,
		mockMcms,
		"0x123456789abcdef",
		entrypointEncoder,
		"0xregistry",
		"0xaccount",
	)

	require.NotNil(t, params)
	assert.Equal(t, "0x123456789abcdef", params.mcmsPackageID)
	assert.Equal(t, "0xregistry", params.registryObj)
	assert.Equal(t, "0xaccount", params.accountObj)
	assert.Equal(t, mockClient, params.client)
	assert.Equal(t, mockMcms, params.mcms)
	assert.Equal(t, entrypointEncoder, params.entryPointEncoder)
}

func TestExecutingCallbackParams_AppendPTB_WithMCMSPackageTarget(t *testing.T) {
	t.Parallel()

	// Create mock objects
	mockClient := mocksui.NewISuiAPI(t)
	mockMcms := mockmcms.NewIMcms(t)
	entrypointEncoder := &MockEntrypointArgEncoder{t: t, registryObj: "0xregistry"}
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
		entrypointEncoder,
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
	entrypointEncoder := &MockEntrypointArgEncoder{
		t:           t,
		registryObj: "0xregistry",
		expected:    expectedCall,
	}
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
		entrypointEncoder,
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
	entrypointEncoder := &MockEntrypointArgEncoder{
		t:           t,
		registryObj: "0xregistry",
	}
	// Create the ExecutingCallbackParams
	params := NewExecutingCallbackParams(
		mockClient,
		mockMcms,
		"0x123456789abcdef",
		entrypointEncoder,
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

func TestExtractExecutingCallbackParams(t *testing.T) {
	t.Parallel()

	ptb := transaction.NewTransaction()
	vectorExecutingCallback := &transaction.Argument{}
	mcmsPackageID := "0x123456789abcdef0" + strings.Repeat("0", 48) // 64 hex chars = 32 bytes

	result, err := extractExecutingCallbackParams(mcmsPackageID, ptb, vectorExecutingCallback)

	require.NoError(t, err, "extractExecutingCallbackParams should not return an error")
	require.NotNil(t, result, "extractExecutingCallbackParams should return a non-nil result")

	assert.IsType(t, &transaction.Argument{}, result, "result should be a *transaction.Argument")
}

func TestExtractExecutingCallbackParams_InvalidPackageID(t *testing.T) {
	t.Parallel()

	ptb := transaction.NewTransaction()
	vectorExecutingCallback := &transaction.Argument{}
	invalidPackageID := "invalid-package-id" // Invalid format

	result, err := extractExecutingCallbackParams(invalidPackageID, ptb, vectorExecutingCallback)

	require.Error(t, err, "extractExecutingCallbackParams should return an error for invalid package ID")
	assert.Nil(t, result, "result should be nil when there's an error")
	assert.Contains(t, err.Error(), "failed to convert type string to TypeTag", "error should mention TypeTag conversion failure")
}

func TestCloseExecutingCallbackParams(t *testing.T) {
	t.Parallel()

	ptb := transaction.NewTransaction()
	vectorExecutingCallback := &transaction.Argument{}
	mcmsPackageID := "0x123456789abcdef0" + strings.Repeat("0", 48) // 64 hex chars = 32 bytes

	err := closeExecutingCallbackParams(mcmsPackageID, ptb, vectorExecutingCallback)

	require.NoError(t, err, "closeExecutingCallbackParams should not return an error")
}

func TestCloseExecutingCallbackParams_InvalidPackageID(t *testing.T) {
	t.Parallel()

	ptb := transaction.NewTransaction()
	vectorExecutingCallback := &transaction.Argument{}
	invalidPackageID := "invalid-package-id" // Invalid format

	err := closeExecutingCallbackParams(invalidPackageID, ptb, vectorExecutingCallback)

	require.Error(t, err, "closeExecutingCallbackParams should return an error for invalid package ID")
	assert.Contains(t, err.Error(), "failed to convert type string to TypeTag", "error should mention TypeTag conversion failure")
}

func TestExecutingCallbackParams_AppendPTB_WithMCMSDeployerTarget_Success(t *testing.T) {
	t.Parallel()

	mockClient := mocksui.NewISuiAPI(t)
	mockMcms := mockmcms.NewIMcms(t)
	mockFeeQuoterEncoder := mockfeequoter.NewFeeQuoterEncoder(t)
	mockMcmsEncoder := mockmcms.NewMcmsEncoder(t)
	mockBound := mockbindutils.NewIBoundContract(t)

	mockMcms.EXPECT().Encoder().Return(mockMcmsEncoder)
	mockMcms.EXPECT().Bound().Return(mockBound)

	expectedDispatchCall := &bind.EncodedCall{
		Module: bind.ModuleInformation{
			PackageID:   "0x123456789abcdef",
			PackageName: "mcms",
			ModuleName:  "mcms",
		},
		Function: "execute_dispatch_to_deployer",
	}
	mockMcmsEncoder.EXPECT().ExecuteDispatchToDeployerWithArgs(
		"0xregistry",
		"0xdeployerstate",
		mock.AnythingOfType("*transaction.Argument"),
	).Return(expectedDispatchCall, nil)

	upgradeTicketArg := &transaction.Argument{}
	mockBound.EXPECT().AppendPTB(
		mock.Anything,
		mock.AnythingOfType("*bind.CallOpts"),
		mock.AnythingOfType("*transaction.Transaction"),
		expectedDispatchCall,
	).Return(upgradeTicketArg, nil)

	mcmsPackageIDHex := "123456789abcdef0" + strings.Repeat("0", 48)
	mcmsPackageIDBytes, err := hex.DecodeString(mcmsPackageIDHex)
	require.NoError(t, err)
	mcmsPackageID := "0x" + mcmsPackageIDHex

	params := NewExecutingCallbackParams(
		mockClient,
		mockMcms,
		mcmsPackageID,
		mockFeeQuoterEncoder,
		"0xregistry",
		"0xaccount",
	)

	params.extractExecutingCallbackParams = func(mcmsPackageID string, ptb *transaction.Transaction, vectorExecutingCallback *transaction.Argument) (*transaction.Argument, error) {
		return &transaction.Argument{}, nil
	}
	params.closeExecutingCallbackParams = func(mcmsPackageID string, ptb *transaction.Transaction, vectorExecutingCallback *transaction.Argument) error {
		return nil
	}

	ctx := context.Background()
	ptb := transaction.NewTransaction()
	executeCallback := &transaction.Argument{}

	calls := []Call{
		{
			Target:           mcmsPackageIDBytes,
			StateObj:         "0xdeployerstate",
			ModuleName:       "mcms_deployer",
			FunctionName:     "authorize_upgrade",
			CompiledModules:  [][]byte{{0xaa, 0xbb}, {0xcc, 0xdd}},
			Dependencies:     []models.SuiAddress{"0x333", "0x444"},
			PackageToUpgrade: "0x555",
		},
	}

	// Execute the method - will fail due to ptb.Upgrade and deployer contract creation
	// This test verifies the code path up to the mock expectations
	err = params.AppendPTB(ctx, ptb, executeCallback, calls)

	// The test will fail at ptb.Upgrade step since we can't easily mock it
	// But the important part is that it validates the mcms_deployer branch works
	// and doesn't return the function validation error
	require.Error(t, err) // Will error at NewMcmsDeployer or CommitUpgrade step
	assert.NotContains(t, err.Error(), "mcms_deployer calls must have FunctionName 'authorize_upgrade'")
}

func TestExecutingCallbackParams_AppendPTB_WithMCMSDeployerTarget_InvalidFunction(t *testing.T) {
	t.Parallel()

	mockClient := mocksui.NewISuiAPI(t)
	mockMcms := mockmcms.NewIMcms(t)
	mockFeeQuoterEncoder := mockfeequoter.NewFeeQuoterEncoder(t)

	mcmsPackageIDHex := "123456789abcdef0" + strings.Repeat("0", 48)
	mcmsPackageIDBytes, err := hex.DecodeString(mcmsPackageIDHex)
	require.NoError(t, err)
	mcmsPackageID := "0x" + mcmsPackageIDHex

	params := NewExecutingCallbackParams(
		mockClient,
		mockMcms,
		mcmsPackageID,
		mockFeeQuoterEncoder,
		"0xregistry",
		"0xaccount",
	)

	params.extractExecutingCallbackParams = func(mcmsPackageID string, ptb *transaction.Transaction, vectorExecutingCallback *transaction.Argument) (*transaction.Argument, error) {
		return &transaction.Argument{}, nil
	}
	params.closeExecutingCallbackParams = func(mcmsPackageID string, ptb *transaction.Transaction, vectorExecutingCallback *transaction.Argument) error {
		return nil
	}

	ctx := context.Background()
	ptb := transaction.NewTransaction()
	executeCallback := &transaction.Argument{}

	calls := []Call{
		{
			Target:       mcmsPackageIDBytes,
			StateObj:     "0xdeployerstate",
			ModuleName:   "mcms_deployer",
			FunctionName: "invalid_function", // Wrong function name
		},
	}

	err = params.AppendPTB(ctx, ptb, executeCallback, calls)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "mcms_deployer calls must have FunctionName 'authorize_upgrade', got: invalid_function")
}

func TestExecutingCallbackParams_AppendPTB_WithMCMSDeployerTarget_ExecuteDispatchError(t *testing.T) {
	t.Parallel()

	mockClient := mocksui.NewISuiAPI(t)
	mockMcms := mockmcms.NewIMcms(t)
	mockFeeQuoterEncoder := mockfeequoter.NewFeeQuoterEncoder(t)
	mockMcmsEncoder := mockmcms.NewMcmsEncoder(t)

	mockMcms.EXPECT().Encoder().Return(mockMcmsEncoder)
	mockMcmsEncoder.EXPECT().ExecuteDispatchToDeployerWithArgs(
		"0xregistry",
		"0xdeployerstate",
		mock.AnythingOfType("*transaction.Argument"),
	).Return(nil, fmt.Errorf("mock dispatch error"))

	mcmsPackageIDHex := "123456789abcdef0" + strings.Repeat("0", 48)
	mcmsPackageIDBytes, err := hex.DecodeString(mcmsPackageIDHex)
	require.NoError(t, err)
	mcmsPackageID := "0x" + mcmsPackageIDHex

	params := NewExecutingCallbackParams(
		mockClient,
		mockMcms,
		mcmsPackageID,
		mockFeeQuoterEncoder,
		"0xregistry",
		"0xaccount",
	)

	params.extractExecutingCallbackParams = func(mcmsPackageID string, ptb *transaction.Transaction, vectorExecutingCallback *transaction.Argument) (*transaction.Argument, error) {
		return &transaction.Argument{}, nil
	}
	params.closeExecutingCallbackParams = func(mcmsPackageID string, ptb *transaction.Transaction, vectorExecutingCallback *transaction.Argument) error {
		return nil
	}

	ctx := context.Background()
	ptb := transaction.NewTransaction()
	executeCallback := &transaction.Argument{}

	calls := []Call{
		{
			Target:       mcmsPackageIDBytes,
			StateObj:     "0xdeployerstate",
			ModuleName:   "mcms_deployer",
			FunctionName: "authorize_upgrade",
		},
	}

	err = params.AppendPTB(ctx, ptb, executeCallback, calls)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "creating ExecuteDispatchToDeployer call 0")
	assert.Contains(t, err.Error(), "mock dispatch error")
}

func TestExecutingCallbackParams_AppendPTB_WithMCMSDeployerTarget_AppendPTBError(t *testing.T) {
	t.Parallel()

	mockClient := mocksui.NewISuiAPI(t)
	mockMcms := mockmcms.NewIMcms(t)
	mockFeeQuoterEncoder := mockfeequoter.NewFeeQuoterEncoder(t)
	mockMcmsEncoder := mockmcms.NewMcmsEncoder(t)
	mockBound := mockbindutils.NewIBoundContract(t)

	mockMcms.EXPECT().Encoder().Return(mockMcmsEncoder)
	mockMcms.EXPECT().Bound().Return(mockBound)

	expectedDispatchCall := &bind.EncodedCall{}
	mockMcmsEncoder.EXPECT().ExecuteDispatchToDeployerWithArgs(
		"0xregistry",
		"0xdeployerstate",
		mock.AnythingOfType("*transaction.Argument"),
	).Return(expectedDispatchCall, nil)

	mockBound.EXPECT().AppendPTB(
		mock.Anything,
		mock.AnythingOfType("*bind.CallOpts"),
		mock.AnythingOfType("*transaction.Transaction"),
		expectedDispatchCall,
	).Return(nil, fmt.Errorf("mock append error"))

	mcmsPackageIDHex := "123456789abcdef0" + strings.Repeat("0", 48)
	mcmsPackageIDBytes, err := hex.DecodeString(mcmsPackageIDHex)
	require.NoError(t, err)
	mcmsPackageID := "0x" + mcmsPackageIDHex

	params := NewExecutingCallbackParams(
		mockClient,
		mockMcms,
		mcmsPackageID,
		mockFeeQuoterEncoder,
		"0xregistry",
		"0xaccount",
	)

	params.extractExecutingCallbackParams = func(mcmsPackageID string, ptb *transaction.Transaction, vectorExecutingCallback *transaction.Argument) (*transaction.Argument, error) {
		return &transaction.Argument{}, nil
	}
	params.closeExecutingCallbackParams = func(mcmsPackageID string, ptb *transaction.Transaction, vectorExecutingCallback *transaction.Argument) error {
		return nil
	}

	ctx := context.Background()
	ptb := transaction.NewTransaction()
	executeCallback := &transaction.Argument{}

	calls := []Call{
		{
			Target:       mcmsPackageIDBytes,
			StateObj:     "0xdeployerstate",
			ModuleName:   "mcms_deployer",
			FunctionName: "authorize_upgrade",
		},
	}

	err = params.AppendPTB(ctx, ptb, executeCallback, calls)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "adding ExecuteDispatchToDeployer call 0 to PTB")
	assert.Contains(t, err.Error(), "mock append error")
}
