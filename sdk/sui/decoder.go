package sui

import (
	"fmt"

	"github.com/aptos-labs/aptos-go-sdk/bcs"
)

type Call struct {
	Target       []byte
	ModuleName   string
	FunctionName string
	Data         []byte
}

func DeserializeTimelockBypasserExecuteBatch(data []byte) ([]Call, error) {
	deserializer := bcs.NewDeserializer(data)

	// Deserialize targets vector
	targetsLen := deserializer.Uleb128()

	targets := make([][]byte, targetsLen)
	for i := uint32(0); i < targetsLen; i++ {
		target := deserializer.ReadFixedBytes(32) // addresses are 32 bytes in Sui
		targets[i] = target
	}

	// Deserialize module names vector
	moduleNamesLen := deserializer.Uleb128()

	moduleNames := make([]string, moduleNamesLen)
	for i := uint32(0); i < moduleNamesLen; i++ {
		moduleName := deserializer.ReadString()
		moduleNames[i] = moduleName
	}

	// Deserialize function names vector
	functionNamesLen := deserializer.Uleb128()

	functionNames := make([]string, functionNamesLen)
	for i := uint32(0); i < functionNamesLen; i++ {
		functionName := deserializer.ReadString()
		functionNames[i] = functionName
	}

	// Deserialize datas vector
	datasLen := deserializer.Uleb128()

	datas := make([][]byte, datasLen)
	for i := uint32(0); i < datasLen; i++ {
		// ReadBytes() handles the length prefix automatically for vector<u8>
		dataBytes := deserializer.ReadBytes()
		datas[i] = dataBytes
	}

	// Verify all vectors have the same length
	if len(targets) != len(moduleNames) || len(moduleNames) != len(functionNames) || len(functionNames) != len(datas) {
		return nil, fmt.Errorf("vector lengths mismatch: targets=%d, moduleNames=%d, functionNames=%d, datas=%d",
			len(targets), len(moduleNames), len(functionNames), len(datas))
	}

	// Convert to Call structs
	calls := make([]Call, len(targets))
	for i := 0; i < len(targets); i++ {
		calls[i] = Call{
			Target:       targets[i],
			ModuleName:   moduleNames[i],
			FunctionName: functionNames[i],
			Data:         datas[i],
		}
	}

	return calls, nil
}
