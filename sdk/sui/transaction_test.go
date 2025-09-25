package sui

import (
	"encoding/json"
	"testing"

	"github.com/block-vision/sui-go-sdk/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
)

func TestValidateAdditionalFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       json.RawMessage
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid additional fields",
			input: json.RawMessage(`{
				"module_name": "test_module",
				"function": "test_function",
				"state_obj": "0x123",
				"internal_state_objects": ["0x456", "0x789"]
			}`),
			expectError: false,
		},
		{
			name: "valid additional fields - minimal",
			input: json.RawMessage(`{
				"module_name": "test",
				"function": "func"
			}`),
			expectError: false,
		},
		{
			name: "valid additional fields - with empty optional fields",
			input: json.RawMessage(`{
				"module_name": "test_module",
				"function": "test_function",
				"state_obj": "",
				"internal_state_objects": []
			}`),
			expectError: false,
		},
		{
			name: "invalid json",
			input: json.RawMessage(`{
				"module_name": "test_module",
				"function": "test_function"
				"invalid": json
			}`),
			expectError: true,
			errorMsg:    "failed to unmarshal Sui additional fields",
		},
		{
			name: "empty module name",
			input: json.RawMessage(`{
				"module_name": "",
				"function": "test_function"
			}`),
			expectError: true,
			errorMsg:    "module name length must be between 1 and 64 characters",
		},
		{
			name: "module name too long",
			input: json.RawMessage(`{
				"module_name": "a_very_long_module_name_that_exceeds_the_maximum_allowed_length_of_64_characters",
				"function": "test_function"
			}`),
			expectError: true,
			errorMsg:    "module name length must be between 1 and 64 characters",
		},
		{
			name: "empty function name",
			input: json.RawMessage(`{
				"module_name": "test_module",
				"function": ""
			}`),
			expectError: true,
			errorMsg:    "function length must be between 1 and 64 characters",
		},
		{
			name: "function name too long",
			input: json.RawMessage(`{
				"module_name": "test_module",
				"function": "a_very_long_function_name_that_exceeds_the_maximum_allowed_length_of_64_characters"
			}`),
			expectError: true,
			errorMsg:    "function length must be between 1 and 64 characters",
		},
		{
			name: "missing module name",
			input: json.RawMessage(`{
				"function": "test_function"
			}`),
			expectError: true,
			errorMsg:    "module name length must be between 1 and 64 characters",
		},
		{
			name: "missing function name",
			input: json.RawMessage(`{
				"module_name": "test_module"
			}`),
			expectError: true,
			errorMsg:    "function length must be between 1 and 64 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateAdditionalFields(tt.input)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAdditionalFields_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		fields      AdditionalFields
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid fields",
			fields: AdditionalFields{
				ModuleName: "test_module",
				Function:   "test_function",
			},
			expectError: false,
		},
		{
			name: "valid fields with state objects",
			fields: AdditionalFields{
				ModuleName:           "test_module",
				Function:             "test_function",
				StateObj:             "0x123",
				InternalStateObjects: []string{"0x456", "0x789"},
			},
			expectError: false,
		},
		{
			name: "minimum valid module name",
			fields: AdditionalFields{
				ModuleName: "a",
				Function:   "f",
			},
			expectError: false,
		},
		{
			name: "maximum valid module name",
			fields: AdditionalFields{
				ModuleName: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ab",
				Function:   "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ab",
			},
			expectError: false,
		},
		{
			name: "empty module name",
			fields: AdditionalFields{
				ModuleName: "",
				Function:   "test_function",
			},
			expectError: true,
			errorMsg:    "module name length must be between 1 and 64 characters",
		},
		{
			name: "module name too long",
			fields: AdditionalFields{
				ModuleName: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abc",
				Function:   "test_function",
			},
			expectError: true,
			errorMsg:    "module name length must be between 1 and 64 characters",
		},
		{
			name: "empty function name",
			fields: AdditionalFields{
				ModuleName: "test_module",
				Function:   "",
			},
			expectError: true,
			errorMsg:    "function length must be between 1 and 64 characters",
		},
		{
			name: "function name too long",
			fields: AdditionalFields{
				ModuleName: "test_module",
				Function:   "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abc",
			},
			expectError: true,
			errorMsg:    "function length must be between 1 and 64 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.fields.Validate()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewTransaction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		moduleName   string
		function     string
		to           string
		data         []byte
		contractType string
		tags         []string
		expected     func(t *testing.T, tx types.Transaction)
	}{
		{
			name:         "basic transaction",
			moduleName:   "test_module",
			function:     "test_function",
			to:           "0x123456789abcdef",
			data:         []byte("test_data"),
			contractType: "TestContract",
			tags:         []string{"tag1", "tag2"},
			expected: func(t *testing.T, tx types.Transaction) {
				t.Helper()
				assert.Equal(t, "0x123456789abcdef", tx.To)
				assert.Equal(t, []byte("test_data"), tx.Data)
				assert.Equal(t, "TestContract", tx.ContractType)
				assert.Equal(t, []string{"tag1", "tag2"}, tx.Tags)

				var additionalFields AdditionalFields
				err := json.Unmarshal(tx.AdditionalFields, &additionalFields)
				require.NoError(t, err)
				assert.Equal(t, "test_module", additionalFields.ModuleName)
				assert.Equal(t, "test_function", additionalFields.Function)
				assert.Empty(t, additionalFields.StateObj)
				assert.Empty(t, additionalFields.InternalStateObjects)
			},
		},
		{
			name:         "transaction with empty data",
			moduleName:   "module",
			function:     "func",
			to:           "0xabc",
			data:         []byte{},
			contractType: "",
			tags:         []string{},
			expected: func(t *testing.T, tx types.Transaction) {
				t.Helper()
				assert.Equal(t, "0xabc", tx.To)
				assert.Equal(t, []byte{}, tx.Data)
				assert.Empty(t, tx.ContractType)
				assert.Equal(t, []string{}, tx.Tags)

				var additionalFields AdditionalFields
				err := json.Unmarshal(tx.AdditionalFields, &additionalFields)
				require.NoError(t, err)
				assert.Equal(t, "module", additionalFields.ModuleName)
				assert.Equal(t, "func", additionalFields.Function)
			},
		},
		{
			name:         "transaction with nil tags",
			moduleName:   "module",
			function:     "func",
			to:           "0xabc",
			data:         []byte("data"),
			contractType: "Contract",
			tags:         nil,
			expected: func(t *testing.T, tx types.Transaction) {
				t.Helper()
				assert.Equal(t, "0xabc", tx.To)
				assert.Equal(t, []byte("data"), tx.Data)
				assert.Equal(t, "Contract", tx.ContractType)
				assert.Nil(t, tx.Tags)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tx, err := NewTransaction(tt.moduleName, tt.function, tt.to, tt.data, tt.contractType, tt.tags)
			require.NoError(t, err)

			tt.expected(t, tx)
		})
	}
}

func TestNewTransactionWithStateObj(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		moduleName   string
		function     string
		to           string
		data         []byte
		contractType string
		tags         []string
		stateObj     string
		expected     func(t *testing.T, tx types.Transaction)
	}{
		{
			name:         "transaction with state object",
			moduleName:   "test_module",
			function:     "test_function",
			to:           "0x123456789abcdef",
			data:         []byte("test_data"),
			contractType: "TestContract",
			tags:         []string{"tag1", "tag2"},
			stateObj:     "0x999",
			expected: func(t *testing.T, tx types.Transaction) {
				t.Helper()
				assert.Equal(t, "0x123456789abcdef", tx.To)
				assert.Equal(t, []byte("test_data"), tx.Data)
				assert.Equal(t, "TestContract", tx.ContractType)
				assert.Equal(t, []string{"tag1", "tag2"}, tx.Tags)

				var additionalFields AdditionalFields
				err := json.Unmarshal(tx.AdditionalFields, &additionalFields)
				require.NoError(t, err)
				assert.Equal(t, "test_module", additionalFields.ModuleName)
				assert.Equal(t, "test_function", additionalFields.Function)
				assert.Equal(t, "0x999", additionalFields.StateObj)
				assert.Nil(t, additionalFields.InternalStateObjects)
			},
		},
		{
			name:         "transaction with empty state object",
			moduleName:   "module",
			function:     "func",
			to:           "0xabc",
			data:         []byte("data"),
			contractType: "Contract",
			tags:         []string{"tag"},
			stateObj:     "",
			expected: func(t *testing.T, tx types.Transaction) {
				t.Helper()
				var additionalFields AdditionalFields
				err := json.Unmarshal(tx.AdditionalFields, &additionalFields)
				require.NoError(t, err)
				assert.Equal(t, "module", additionalFields.ModuleName)
				assert.Equal(t, "func", additionalFields.Function)
				assert.Empty(t, additionalFields.StateObj)
				assert.Nil(t, additionalFields.InternalStateObjects)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tx, err := NewTransactionWithStateObj(tt.moduleName, tt.function, tt.to, tt.data, tt.contractType, tt.tags, tt.stateObj)
			require.NoError(t, err)

			tt.expected(t, tx)
		})
	}
}

func TestNewTransactionWithManyStateObj(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		moduleName           string
		function             string
		to                   string
		data                 []byte
		contractType         string
		tags                 []string
		stateObj             string
		internalStateObjects []string
		expected             func(t *testing.T, tx types.Transaction)
	}{
		{
			name:                 "transaction with multiple state objects",
			moduleName:           "test_module",
			function:             "test_function",
			to:                   "0x123456789abcdef",
			data:                 []byte("test_data"),
			contractType:         "TestContract",
			tags:                 []string{"tag1", "tag2"},
			stateObj:             "0x999",
			internalStateObjects: []string{"0x111", "0x222", "0x333"},
			expected: func(t *testing.T, tx types.Transaction) {
				t.Helper()
				assert.Equal(t, "0x123456789abcdef", tx.To)
				assert.Equal(t, []byte("test_data"), tx.Data)
				assert.Equal(t, "TestContract", tx.ContractType)
				assert.Equal(t, []string{"tag1", "tag2"}, tx.Tags)

				var additionalFields AdditionalFields
				err := json.Unmarshal(tx.AdditionalFields, &additionalFields)
				require.NoError(t, err)
				assert.Equal(t, "test_module", additionalFields.ModuleName)
				assert.Equal(t, "test_function", additionalFields.Function)
				assert.Equal(t, "0x999", additionalFields.StateObj)
				assert.Equal(t, []string{"0x111", "0x222", "0x333"}, additionalFields.InternalStateObjects)
			},
		},
		{
			name:                 "transaction with empty internal state objects",
			moduleName:           "module",
			function:             "func",
			to:                   "0xabc",
			data:                 []byte("data"),
			contractType:         "Contract",
			tags:                 []string{"tag"},
			stateObj:             "0x999",
			internalStateObjects: []string{},
			expected: func(t *testing.T, tx types.Transaction) {
				t.Helper()
				var additionalFields AdditionalFields
				err := json.Unmarshal(tx.AdditionalFields, &additionalFields)
				require.NoError(t, err)
				assert.Equal(t, "module", additionalFields.ModuleName)
				assert.Equal(t, "func", additionalFields.Function)
				assert.Equal(t, "0x999", additionalFields.StateObj)
				// When marshaling/unmarshaling, empty slice becomes nil
				assert.Empty(t, additionalFields.InternalStateObjects)
			},
		},
		{
			name:                 "transaction with nil internal state objects",
			moduleName:           "module",
			function:             "func",
			to:                   "0xabc",
			data:                 []byte("data"),
			contractType:         "Contract",
			tags:                 []string{"tag"},
			stateObj:             "0x999",
			internalStateObjects: nil,
			expected: func(t *testing.T, tx types.Transaction) {
				t.Helper()
				var additionalFields AdditionalFields
				err := json.Unmarshal(tx.AdditionalFields, &additionalFields)
				require.NoError(t, err)
				assert.Equal(t, "module", additionalFields.ModuleName)
				assert.Equal(t, "func", additionalFields.Function)
				assert.Equal(t, "0x999", additionalFields.StateObj)
				assert.Nil(t, additionalFields.InternalStateObjects)
			},
		},
		{
			name:                 "transaction with single internal state object",
			moduleName:           "module",
			function:             "func",
			to:                   "0xabc",
			data:                 []byte("data"),
			contractType:         "Contract",
			tags:                 []string{},
			stateObj:             "",
			internalStateObjects: []string{"0x111"},
			expected: func(t *testing.T, tx types.Transaction) {
				t.Helper()
				var additionalFields AdditionalFields
				err := json.Unmarshal(tx.AdditionalFields, &additionalFields)
				require.NoError(t, err)
				assert.Equal(t, "module", additionalFields.ModuleName)
				assert.Equal(t, "func", additionalFields.Function)
				assert.Empty(t, additionalFields.StateObj)
				assert.Equal(t, []string{"0x111"}, additionalFields.InternalStateObjects)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tx, err := NewTransactionWithManyStateObj(tt.moduleName, tt.function, tt.to, tt.data, tt.contractType, tt.tags, tt.stateObj, tt.internalStateObjects)
			require.NoError(t, err)

			tt.expected(t, tx)
		})
	}
}

func TestAdditionalFieldsJSONMarshaling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fields   AdditionalFields
		expected string
	}{
		{
			name: "all fields populated",
			fields: AdditionalFields{
				ModuleName:           "test_module",
				Function:             "test_function",
				StateObj:             "0x123",
				InternalStateObjects: []string{"0x456", "0x789"},
			},
			expected: `{"module_name":"test_module","function":"test_function","state_obj":"0x123","internal_state_objects":["0x456","0x789"]}`,
		},
		{
			name: "minimal fields",
			fields: AdditionalFields{
				ModuleName: "module",
				Function:   "func",
			},
			expected: `{"module_name":"module","function":"func"}`,
		},
		{
			name: "with empty optional fields",
			fields: AdditionalFields{
				ModuleName:           "module",
				Function:             "func",
				StateObj:             "",
				InternalStateObjects: []string{},
			},
			expected: `{"module_name":"module","function":"func"}`,
		},
		{
			name: "with deployment fields populated",
			fields: AdditionalFields{
				ModuleName:       "deploy_module",
				Function:         "deploy_func",
				CompiledModules:  [][]byte{[]byte("module1"), []byte("module2")},
				Dependencies:     []models.SuiAddress{"0xdep1", "0xdep2"},
				PackageToUpgrade: "0xpackage123",
			},
			expected: `{"module_name":"deploy_module","function":"deploy_func","compiled_modules":["bW9kdWxlMQ==","bW9kdWxlMg=="],"dependencies":["0xdep1","0xdep2"],"package_to_upgrade":"0xpackage123"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test marshaling
			marshaled, err := json.Marshal(tt.fields)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(marshaled))

			// Test unmarshaling - be more lenient with empty vs nil slices
			var unmarshaled AdditionalFields
			err = json.Unmarshal(marshaled, &unmarshaled)
			require.NoError(t, err)

			// For comparison, normalize empty slices to nil for consistency
			expectedFields := tt.fields
			if len(expectedFields.InternalStateObjects) == 0 {
				expectedFields.InternalStateObjects = nil
			}
			if len(unmarshaled.InternalStateObjects) == 0 {
				unmarshaled.InternalStateObjects = nil
			}
			assert.Equal(t, expectedFields, unmarshaled)
		})
	}
}

func TestTransactionIntegration(t *testing.T) {
	t.Parallel()

	// Test the integration between all transaction creation functions
	moduleName := "integration_module"
	function := "integration_function"
	to := "0x123456789abcdef"
	data := []byte("integration_test_data")
	contractType := "IntegrationContract"
	tags := []string{"integration", "test"}
	stateObj := "0x999"
	internalStateObjects := []string{"0x111", "0x222"}
	compiledModules := [][]byte{{0xaa, 0xbb}, {0xcc, 0xdd}}
	dependencies := []models.SuiAddress{"0x333", "0x444"}
	packageToUpgrade := "0x555"

	// Test NewTransaction
	tx1, err := NewTransaction(moduleName, function, to, data, contractType, tags)
	require.NoError(t, err)

	// Test NewTransactionWithStateObj
	tx2, err := NewTransactionWithStateObj(moduleName, function, to, data, contractType, tags, stateObj)
	require.NoError(t, err)

	// Test NewTransactionWithManyStateObj
	tx3, err := NewTransactionWithManyStateObj(moduleName, function, to, data, contractType, tags, stateObj, internalStateObjects)
	require.NoError(t, err)

	// Test NewTransactionWithUpgradeData
	tx4, err := NewTransactionWithUpgradeData(moduleName, function, to, data, contractType, tags, stateObj, internalStateObjects, compiledModules, dependencies, packageToUpgrade)
	require.NoError(t, err)

	// Verify all transactions have the same basic structure
	assert.Equal(t, tx1.To, tx2.To)
	assert.Equal(t, tx1.To, tx3.To)
	assert.Equal(t, tx1.To, tx4.To)
	assert.Equal(t, tx1.Data, tx2.Data)
	assert.Equal(t, tx1.Data, tx3.Data)
	assert.Equal(t, tx1.Data, tx4.Data)
	assert.Equal(t, tx1.ContractType, tx2.ContractType)
	assert.Equal(t, tx1.ContractType, tx3.ContractType)
	assert.Equal(t, tx1.ContractType, tx4.ContractType)
	assert.Equal(t, tx1.Tags, tx2.Tags)
	assert.Equal(t, tx1.Tags, tx3.Tags)
	assert.Equal(t, tx1.Tags, tx4.Tags)

	// Verify additional fields differ appropriately
	var fields1, fields2, fields3, fields4 AdditionalFields
	err = json.Unmarshal(tx1.AdditionalFields, &fields1)
	require.NoError(t, err)
	err = json.Unmarshal(tx2.AdditionalFields, &fields2)
	require.NoError(t, err)
	err = json.Unmarshal(tx3.AdditionalFields, &fields3)
	require.NoError(t, err)
	err = json.Unmarshal(tx4.AdditionalFields, &fields4)
	require.NoError(t, err)

	// Basic fields should be the same
	assert.Equal(t, fields1.ModuleName, fields2.ModuleName)
	assert.Equal(t, fields1.ModuleName, fields3.ModuleName)
	assert.Equal(t, fields1.ModuleName, fields4.ModuleName)
	assert.Equal(t, fields1.Function, fields2.Function)
	assert.Equal(t, fields1.Function, fields3.Function)
	assert.Equal(t, fields1.Function, fields4.Function)

	// State objects should differ
	assert.Empty(t, fields1.StateObj)
	assert.Equal(t, stateObj, fields2.StateObj)
	assert.Equal(t, stateObj, fields3.StateObj)
	assert.Equal(t, stateObj, fields4.StateObj)

	assert.Empty(t, fields1.InternalStateObjects)
	assert.Nil(t, fields2.InternalStateObjects)
	assert.Equal(t, internalStateObjects, fields3.InternalStateObjects)
	assert.Equal(t, internalStateObjects, fields4.InternalStateObjects)

	// Upgrade-specific fields should only be in tx4
	assert.Nil(t, fields1.CompiledModules)
	assert.Nil(t, fields2.CompiledModules)
	assert.Nil(t, fields3.CompiledModules)
	assert.Equal(t, compiledModules, fields4.CompiledModules)

	assert.Nil(t, fields1.Dependencies)
	assert.Nil(t, fields2.Dependencies)
	assert.Nil(t, fields3.Dependencies)
	assert.Equal(t, dependencies, fields4.Dependencies)

	assert.Empty(t, fields1.PackageToUpgrade)
	assert.Empty(t, fields2.PackageToUpgrade)
	assert.Empty(t, fields3.PackageToUpgrade)
	assert.Equal(t, packageToUpgrade, fields4.PackageToUpgrade)

	// Verify all additional fields are valid
	require.NoError(t, ValidateAdditionalFields(tx1.AdditionalFields))
	require.NoError(t, ValidateAdditionalFields(tx2.AdditionalFields))
	require.NoError(t, ValidateAdditionalFields(tx3.AdditionalFields))
	require.NoError(t, ValidateAdditionalFields(tx4.AdditionalFields))
}

func TestNewTransactionWithUpgradeData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		moduleName           string
		function             string
		to                   string
		data                 []byte
		contractType         string
		tags                 []string
		stateObj             string
		internalStateObjects []string
		compiledModules      [][]byte
		dependencies         []models.SuiAddress
		packageToUpgrade     string
		expected             func(t *testing.T, tx types.Transaction)
	}{
		{
			name:                 "complete upgrade transaction",
			moduleName:           "mcms_deployer",
			function:             "authorize_upgrade",
			to:                   "0x123456789abcdef",
			data:                 []byte("upgrade_data"),
			contractType:         "MCMS",
			tags:                 []string{"upgrade", "mcms"},
			stateObj:             "0x999",
			internalStateObjects: []string{"0x111", "0x222"},
			compiledModules:      [][]byte{{0xaa, 0xbb}, {0xcc, 0xdd}},
			dependencies:         []models.SuiAddress{"0x333", "0x444"},
			packageToUpgrade:     "0x555",
			expected: func(t *testing.T, tx types.Transaction) {
				t.Helper()
				assert.Equal(t, "0x123456789abcdef", tx.To)
				assert.Equal(t, []byte("upgrade_data"), tx.Data)
				assert.Equal(t, "MCMS", tx.ContractType)
				assert.Equal(t, []string{"upgrade", "mcms"}, tx.Tags)

				var additionalFields AdditionalFields
				err := json.Unmarshal(tx.AdditionalFields, &additionalFields)
				require.NoError(t, err)
				assert.Equal(t, "mcms_deployer", additionalFields.ModuleName)
				assert.Equal(t, "authorize_upgrade", additionalFields.Function)
				assert.Equal(t, "0x999", additionalFields.StateObj)
				assert.Equal(t, []string{"0x111", "0x222"}, additionalFields.InternalStateObjects)
				assert.Equal(t, [][]byte{{0xaa, 0xbb}, {0xcc, 0xdd}}, additionalFields.CompiledModules)
				assert.Equal(t, []models.SuiAddress{"0x333", "0x444"}, additionalFields.Dependencies)
				assert.Equal(t, "0x555", additionalFields.PackageToUpgrade)
			},
		},
		{
			name:                 "minimal upgrade transaction",
			moduleName:           "module",
			function:             "func",
			to:                   "0xabc",
			data:                 []byte{},
			contractType:         "",
			tags:                 []string{},
			stateObj:             "",
			internalStateObjects: nil,
			compiledModules:      nil,
			dependencies:         nil,
			packageToUpgrade:     "",
			expected: func(t *testing.T, tx types.Transaction) {
				t.Helper()
				assert.Equal(t, "0xabc", tx.To)
				assert.Equal(t, []byte{}, tx.Data)
				assert.Empty(t, tx.ContractType)
				assert.Equal(t, []string{}, tx.Tags)

				var additionalFields AdditionalFields
				err := json.Unmarshal(tx.AdditionalFields, &additionalFields)
				require.NoError(t, err)
				assert.Equal(t, "module", additionalFields.ModuleName)
				assert.Equal(t, "func", additionalFields.Function)
				assert.Empty(t, additionalFields.StateObj)
				assert.Nil(t, additionalFields.InternalStateObjects)
				assert.Nil(t, additionalFields.CompiledModules)
				assert.Nil(t, additionalFields.Dependencies)
				assert.Empty(t, additionalFields.PackageToUpgrade)
			},
		},
		{
			name:                 "empty slices upgrade transaction",
			moduleName:           "test_module",
			function:             "test_func",
			to:                   "0x123",
			data:                 []byte("test"),
			contractType:         "Test",
			tags:                 []string{"test"},
			stateObj:             "0x777",
			internalStateObjects: []string{},
			compiledModules:      [][]byte{},
			dependencies:         []models.SuiAddress{},
			packageToUpgrade:     "0x888",
			expected: func(t *testing.T, tx types.Transaction) {
				t.Helper()
				var additionalFields AdditionalFields
				err := json.Unmarshal(tx.AdditionalFields, &additionalFields)
				require.NoError(t, err)
				assert.Equal(t, "test_module", additionalFields.ModuleName)
				assert.Equal(t, "test_func", additionalFields.Function)
				assert.Equal(t, "0x777", additionalFields.StateObj)
				assert.Empty(t, additionalFields.InternalStateObjects)
				assert.Empty(t, additionalFields.CompiledModules)
				assert.Empty(t, additionalFields.Dependencies)
				assert.Equal(t, "0x888", additionalFields.PackageToUpgrade)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tx, err := NewTransactionWithUpgradeData(tt.moduleName, tt.function, tt.to, tt.data, tt.contractType, tt.tags, tt.stateObj, tt.internalStateObjects, tt.compiledModules, tt.dependencies, tt.packageToUpgrade)
			require.NoError(t, err)

			tt.expected(t, tx)
		})
	}
}

func TestCreateUpgradeTransaction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		compiledPackage   func() bind.PackageArtifact
		mcmsPackageID     string
		depStateObj       string
		registryObj       string
		mcmsUserPackageID string
		expectError       bool
		errorMsg          string
		expected          func(t *testing.T, tx types.Transaction)
	}{
		{
			name: "valid upgrade transaction",
			compiledPackage: func() bind.PackageArtifact {
				return bind.PackageArtifact{
					Digest:       []byte("test_digest_123"),
					Modules:      []string{"bW9kdWxlMQ==", "bW9kdWxlMg=="}, // base64 encoded "module1", "module2"
					Dependencies: []string{"0x1234567890abcdef123456789abcdef0123456789abcdef0123456789abcdef01", "0x234567890abcdef123456789abcdef0123456789abcdef0123456789abcdef01"},
				}
			},
			mcmsPackageID:     "0x123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef01",
			depStateObj:       "0x456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef012345",
			registryObj:       "0x789abcdef0123456789abcdef0123456789abcdef0123456789abcdef01234567",
			mcmsUserPackageID: "0x1111456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef01",
			expectError:       false,
			expected: func(t *testing.T, tx types.Transaction) {
				t.Helper()
				assert.Equal(t, "0x123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef01", tx.To)
				assert.Equal(t, "MCMS", tx.ContractType)
				assert.Equal(t, []string{"upgrade", "mcms"}, tx.Tags)

				var additionalFields AdditionalFields
				err := json.Unmarshal(tx.AdditionalFields, &additionalFields)
				require.NoError(t, err)
				assert.Equal(t, "mcms_deployer", additionalFields.ModuleName)
				assert.Equal(t, "authorize_upgrade", additionalFields.Function)
				assert.Equal(t, "0x456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef012345", additionalFields.StateObj)
				assert.Equal(t, []string{"0x789abcdef0123456789abcdef0123456789abcdef0123456789abcdef01234567"}, additionalFields.InternalStateObjects)
				assert.Equal(t, "0x1111456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef01", additionalFields.PackageToUpgrade)

				assert.Len(t, additionalFields.CompiledModules, 2)
				assert.Equal(t, []byte("module1"), additionalFields.CompiledModules[0])
				assert.Equal(t, []byte("module2"), additionalFields.CompiledModules[1])

				assert.Equal(t, []models.SuiAddress{"0x1234567890abcdef123456789abcdef0123456789abcdef0123456789abcdef01", "0x234567890abcdef123456789abcdef0123456789abcdef0123456789abcdef01"}, additionalFields.Dependencies)
				assert.NotEmpty(t, tx.Data)
			},
		},
		{
			name: "empty modules upgrade transaction",
			compiledPackage: func() bind.PackageArtifact {
				return bind.PackageArtifact{
					Digest:       []byte("empty_digest"),
					Modules:      []string{},
					Dependencies: []string{},
				}
			},
			mcmsPackageID:     "0x456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef012345",
			depStateObj:       "0x789abcdef0123456789abcdef0123456789abcdef0123456789abcdef01234567",
			registryObj:       "0x012345678abcdef0123456789abcdef0123456789abcdef0123456789abcdef01",
			mcmsUserPackageID: "0x2222456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			expectError:       false,
			expected: func(t *testing.T, tx types.Transaction) {
				t.Helper()
				assert.Equal(t, "0x456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef012345", tx.To)
				assert.Equal(t, "MCMS", tx.ContractType)
				assert.Equal(t, []string{"upgrade", "mcms"}, tx.Tags)

				var additionalFields AdditionalFields
				err := json.Unmarshal(tx.AdditionalFields, &additionalFields)
				require.NoError(t, err)

				assert.Equal(t, "mcms_deployer", additionalFields.ModuleName)
				assert.Equal(t, "authorize_upgrade", additionalFields.Function)
				assert.Equal(t, "0x789abcdef0123456789abcdef0123456789abcdef0123456789abcdef01234567", additionalFields.StateObj)
				assert.Equal(t, []string{"0x012345678abcdef0123456789abcdef0123456789abcdef0123456789abcdef01"}, additionalFields.InternalStateObjects)
				assert.Equal(t, "0x2222456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", additionalFields.PackageToUpgrade)
				assert.Empty(t, additionalFields.CompiledModules)
				assert.Empty(t, additionalFields.Dependencies)
			},
		},
		{
			name: "single module upgrade transaction",
			compiledPackage: func() bind.PackageArtifact {
				return bind.PackageArtifact{
					Digest:       []byte("single_module_digest"),
					Modules:      []string{"c2luZ2xlX21vZHVsZQ=="}, // base64 encoded "single_module"
					Dependencies: []string{"0x567890abcdef123456789abcdef0123456789abcdef0123456789abcdef012345"},
				}
			},
			mcmsPackageID:     "0x789abcdef0123456789abcdef0123456789abcdef0123456789abcdef01234567",
			depStateObj:       "0x123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef01",
			registryObj:       "0x456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef012345",
			mcmsUserPackageID: "0x3333456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			expectError:       false,
			expected: func(t *testing.T, tx types.Transaction) {
				t.Helper()
				var additionalFields AdditionalFields
				err := json.Unmarshal(tx.AdditionalFields, &additionalFields)
				require.NoError(t, err)

				assert.Len(t, additionalFields.CompiledModules, 1)
				assert.Equal(t, []byte("single_module"), additionalFields.CompiledModules[0])
				assert.Equal(t, []models.SuiAddress{"0x567890abcdef123456789abcdef0123456789abcdef0123456789abcdef012345"}, additionalFields.Dependencies)
			},
		},
		{
			name: "invalid base64 module",
			compiledPackage: func() bind.PackageArtifact {
				return bind.PackageArtifact{
					Digest:       []byte("invalid_digest"),
					Modules:      []string{"invalid_base64!@#"},
					Dependencies: []string{},
				}
			},
			mcmsPackageID:     "0x999456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef01",
			depStateObj:       "0x999789abcdef0123456789abcdef0123456789abcdef0123456789abcdef01234",
			registryObj:       "0x999012345abcdef0123456789abcdef0123456789abcdef0123456789abcdef01",
			mcmsUserPackageID: "0x999333456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			expectError:       true,
			errorMsg:          "decoding module 0",
		},
		{
			name: "empty package id parameters",
			compiledPackage: func() bind.PackageArtifact {
				return bind.PackageArtifact{
					Digest:       []byte("test_digest"),
					Modules:      []string{"dGVzdA=="}, // base64 "test"
					Dependencies: []string{},
				}
			},
			mcmsPackageID:     "",
			depStateObj:       "",
			registryObj:       "",
			mcmsUserPackageID: "",
			expectError:       false,
			expected: func(t *testing.T, tx types.Transaction) {
				t.Helper()
				assert.Empty(t, tx.To)
				var additionalFields AdditionalFields
				err := json.Unmarshal(tx.AdditionalFields, &additionalFields)
				require.NoError(t, err)

				assert.Empty(t, additionalFields.StateObj)
				assert.Equal(t, []string{""}, additionalFields.InternalStateObjects)
				assert.Empty(t, additionalFields.PackageToUpgrade)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			compiledPackage := tt.compiledPackage()
			tx, err := CreateUpgradeTransaction(compiledPackage, tt.mcmsPackageID, tt.depStateObj, tt.registryObj, tt.mcmsUserPackageID)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				tt.expected(t, tx)
				require.NoError(t, ValidateAdditionalFields(tx.AdditionalFields))
			}
		})
	}
}
