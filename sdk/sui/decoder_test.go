package sui

import (
	"testing"

	"github.com/aptos-labs/aptos-go-sdk/bcs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeserializeTimelockBypasserExecuteBatch(t *testing.T) {
	t.Parallel()

	t.Run("Success - valid BCS data with multiple calls", func(t *testing.T) {
		t.Parallel()
		// Create test data that matches the expected structure
		// Targets are 32-byte addresses (SuiAddressLength)
		targets := [][]byte{
			make([]byte, 32), // address 1 (all zeros)
			make([]byte, 32), // address 2 (all zeros)
			make([]byte, 32), // address 3 (all zeros)
		}
		// Make targets distinguishable
		targets[0][31] = 0x01
		targets[1][31] = 0x02
		targets[2][31] = 0x03

		moduleNames := []string{"module1", "module2", "module3"}
		functionNames := []string{"function1", "function2", "function3"}
		datas := [][]byte{
			{0x01, 0x02, 0x03},
			{0x04, 0x05},
			{0x06, 0x07, 0x08, 0x09},
		}

		// Use the same serialization function that the encoder uses
		serializedData, err := SerializeTimelockBypasserExecuteBatch(targets, moduleNames, functionNames, datas)
		require.NoError(t, err)

		// Test deserialization
		resultCalls, err := DeserializeTimelockBypasserExecuteBatch(serializedData)
		require.NoError(t, err)

		// Verify results
		require.Len(t, resultCalls, 3)
		for i, call := range resultCalls {
			assert.Equal(t, targets[i], call.Target)
			assert.Equal(t, moduleNames[i], call.ModuleName)
			assert.Equal(t, functionNames[i], call.FunctionName)
			assert.Equal(t, datas[i], call.Data)
		}
	})

	t.Run("Success - empty vectors", func(t *testing.T) {
		t.Parallel()
		// Test with empty vectors (all should have zero length)
		targets := [][]byte{}
		moduleNames := []string{}
		functionNames := []string{}
		datas := [][]byte{}

		serializedData, err := SerializeTimelockBypasserExecuteBatch(targets, moduleNames, functionNames, datas)
		require.NoError(t, err)

		// Test deserialization
		resultCalls, err := DeserializeTimelockBypasserExecuteBatch(serializedData)
		require.NoError(t, err)

		// Verify empty results
		assert.Equal(t, []Call{}, resultCalls)
	})

	t.Run("Success - single call", func(t *testing.T) {
		t.Parallel()
		// Test with single elements in each vector
		target := make([]byte, 32)
		copy(target[12:], []byte("0xabcdef1234567890"))
		targets := [][]byte{target}
		moduleNames := []string{"my_module"}
		functionNames := []string{"my_function"}
		datas := [][]byte{{0xde, 0xad, 0xbe, 0xef}}

		serializedData, err := SerializeTimelockBypasserExecuteBatch(targets, moduleNames, functionNames, datas)
		require.NoError(t, err)

		// Test deserialization
		resultCalls, err := DeserializeTimelockBypasserExecuteBatch(serializedData)
		require.NoError(t, err)

		// Verify results
		require.Len(t, resultCalls, 1)
		assert.Equal(t, targets[0], resultCalls[0].Target)
		assert.Equal(t, moduleNames[0], resultCalls[0].ModuleName)
		assert.Equal(t, functionNames[0], resultCalls[0].FunctionName)
		assert.Equal(t, datas[0], resultCalls[0].Data)
	})

	t.Run("Error - targets and moduleNames length mismatch", func(t *testing.T) {
		t.Parallel()
		// Create vectors with different lengths to trigger validation error
		targets := [][]byte{make([]byte, 32), make([]byte, 32)} // 2 elements
		moduleNames := []string{"module1"}                      // 1 element - mismatch!
		functionNames := []string{"func1", "func2"}             // 2 elements
		datas := [][]byte{{0x01}, {0x02}}                       // 2 elements

		// We need to manually create the mismatched BCS data since SerializeTimelockBypasserExecuteBatch
		// would validate the lengths before serializing
		serializedData, err := bcs.SerializeSingle(func(ser *bcs.Serializer) {
			// Serialize targets vector (2 elements)
			ser.Uleb128(uint32(len(targets)))
			for _, target := range targets {
				ser.FixedBytes(target)
			}

			// Serialize module names vector (1 element - mismatch!)
			ser.Uleb128(uint32(len(moduleNames)))
			for _, moduleName := range moduleNames {
				ser.WriteString(moduleName)
			}

			// Serialize function names vector (2 elements)
			ser.Uleb128(uint32(len(functionNames)))
			for _, functionName := range functionNames {
				ser.WriteString(functionName)
			}

			// Serialize datas vector (2 elements)
			ser.Uleb128(uint32(len(datas)))
			for _, data := range datas {
				ser.WriteBytes(data)
			}
		})
		require.NoError(t, err)

		// Test deserialization - should fail
		_, err = DeserializeTimelockBypasserExecuteBatch(serializedData)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "vector lengths mismatch")
	})

	t.Run("Error - targets and functionNames length mismatch", func(t *testing.T) {
		t.Parallel()
		targets := [][]byte{make([]byte, 32), make([]byte, 32)} // 2 elements
		moduleNames := []string{"mod1", "mod2"}                 // 2 elements
		functionNames := []string{"func1"}                      // 1 element - mismatch!
		datas := [][]byte{{0x01}, {0x02}}                       // 2 elements

		serializedData, err := bcs.SerializeSingle(func(ser *bcs.Serializer) {
			ser.Uleb128(uint32(len(targets)))
			for _, target := range targets {
				ser.FixedBytes(target)
			}

			ser.Uleb128(uint32(len(moduleNames)))
			for _, moduleName := range moduleNames {
				ser.WriteString(moduleName)
			}

			ser.Uleb128(uint32(len(functionNames)))
			for _, functionName := range functionNames {
				ser.WriteString(functionName)
			}

			ser.Uleb128(uint32(len(datas)))
			for _, data := range datas {
				ser.WriteBytes(data)
			}
		})
		require.NoError(t, err)

		// Test deserialization - should fail
		_, err = DeserializeTimelockBypasserExecuteBatch(serializedData)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "vector lengths mismatch")
	})

	t.Run("Error - targets and datas length mismatch", func(t *testing.T) {
		t.Parallel()
		targets := [][]byte{make([]byte, 32), make([]byte, 32)} // 2 elements
		moduleNames := []string{"mod1", "mod2"}                 // 2 elements
		functionNames := []string{"func1", "func2"}             // 2 elements
		datas := [][]byte{{0x01}}                               // 1 element - mismatch!

		serializedData, err := bcs.SerializeSingle(func(ser *bcs.Serializer) {
			ser.Uleb128(uint32(len(targets)))
			for _, target := range targets {
				ser.FixedBytes(target)
			}

			ser.Uleb128(uint32(len(moduleNames)))
			for _, moduleName := range moduleNames {
				ser.WriteString(moduleName)
			}

			ser.Uleb128(uint32(len(functionNames)))
			for _, functionName := range functionNames {
				ser.WriteString(functionName)
			}

			ser.Uleb128(uint32(len(datas)))
			for _, data := range datas {
				ser.WriteBytes(data)
			}
		})
		require.NoError(t, err)

		// Test deserialization - should fail
		_, err = DeserializeTimelockBypasserExecuteBatch(serializedData)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "vector lengths mismatch")
	})

	t.Run("Error - insufficient data for targets vector", func(t *testing.T) {
		t.Parallel()
		// Create incomplete BCS data - only length byte for targets but no content
		serializedData := []byte{0x02} // ULEB128 encoding for length 2, but no actual address data

		_, err := DeserializeTimelockBypasserExecuteBatch(serializedData)
		require.Error(t, err)
		// The error will come from the BCS unmarshaling trying to read beyond available data
	})

	t.Run("Error - partial vector data", func(t *testing.T) {
		t.Parallel()
		// Create valid data for targets but missing subsequent vectors
		serializedData, err := bcs.SerializeSingle(func(ser *bcs.Serializer) {
			// Only provide targets data, missing moduleNames, functionNames, and datas
			targets := [][]byte{make([]byte, 32)}
			ser.Uleb128(uint32(len(targets)))
			for _, target := range targets {
				ser.FixedBytes(target)
			}
			// Missing moduleNames, functionNames, and datas vectors
		})
		require.NoError(t, err)

		_, err = DeserializeTimelockBypasserExecuteBatch(serializedData)
		require.Error(t, err)
	})

	t.Run("Success - complex data with varied byte arrays", func(t *testing.T) {
		t.Parallel()
		// Test with more complex, realistic data
		target1 := make([]byte, 32)
		target2 := make([]byte, 32)
		// Set some distinguishable bytes in the addresses
		copy(target1[16:], []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef, 0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef})
		copy(target2[16:], []byte{0xab, 0xcd, 0xef, 0xab, 0xcd, 0xef, 0xab, 0xcd, 0xef, 0xab, 0xcd, 0xef, 0xab, 0xcd, 0xef, 0xab})

		targets := [][]byte{target1, target2}
		moduleNames := []string{
			"mcms_timelock_manager",
			"governance_module",
		}
		functionNames := []string{
			"update_configuration",
			"execute_proposal",
		}
		datas := [][]byte{
			{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, // Configuration data
			{0xde, 0xad, 0xbe, 0xef, 0xca, 0xfe, 0xba, 0xbe}, // Proposal data
		}

		serializedData, err := SerializeTimelockBypasserExecuteBatch(targets, moduleNames, functionNames, datas)
		require.NoError(t, err)

		// Test deserialization
		resultCalls, err := DeserializeTimelockBypasserExecuteBatch(serializedData)
		require.NoError(t, err)

		// Verify results
		require.Len(t, resultCalls, 2)
		for i, call := range resultCalls {
			assert.Equal(t, targets[i], call.Target)
			assert.Equal(t, moduleNames[i], call.ModuleName)
			assert.Equal(t, functionNames[i], call.FunctionName)
			assert.Equal(t, datas[i], call.Data)
		}
	})
}
