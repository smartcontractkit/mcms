package sui

import (
	"fmt"

	"github.com/aptos-labs/aptos-go-sdk/bcs"
)

const (
	// SuiAddressLength represents the byte length of Sui addresses
	SuiAddressLength = 32
	// MinimumResultLength is the minimum expected length for certain results
	MinimumResultLength = 2
)

type Call struct {
	Target       []byte
	ModuleName   string
	FunctionName string
	Data         []byte
	StateObj     string
}

func DeserializeTimelockBypasserExecuteBatch(data []byte) ([]Call, error) {
	deserializer := bcs.NewDeserializer(data)

	// Deserialize stateObjects vector
	stateObjectsLen := deserializer.Uleb128()
	stateObjects := make([]string, stateObjectsLen)
	for i := range stateObjectsLen {
		stateObjects[i] = deserializer.ReadString()
	}

	// Deserialize targets vector
	targetsLen := deserializer.Uleb128()
	targets := make([][]byte, targetsLen)
	for i := range targetsLen {
		target := deserializer.ReadFixedBytes(SuiAddressLength) // addresses are 32 bytes in Sui
		targets[i] = target
	}

	// Deserialize module names vector
	moduleNamesLen := deserializer.Uleb128()
	moduleNames := make([]string, moduleNamesLen)
	for i := range moduleNamesLen {
		moduleName := deserializer.ReadString()
		moduleNames[i] = moduleName
	}

	// Deserialize function names vector
	functionNamesLen := deserializer.Uleb128()
	functionNames := make([]string, functionNamesLen)
	for i := range functionNamesLen {
		functionName := deserializer.ReadString()
		functionNames[i] = functionName
	}

	// Deserialize datas vector
	datasLen := deserializer.Uleb128()
	datas := make([][]byte, datasLen)
	for i := range datasLen {
		// ReadBytes() handles the length prefix automatically for vector<u8>
		dataBytes := deserializer.ReadBytes()
		datas[i] = dataBytes
	}

	// Verify all vectors have the same length
	if len(targets) != len(moduleNames) || len(moduleNames) != len(functionNames) || len(functionNames) != len(datas) {
		return nil, fmt.Errorf("vector lengths mismatch: targets=%d, moduleNames=%d, functionNames=%d, datas=%d",
			len(targets), len(moduleNames), len(functionNames), len(datas))
	}

	// If stateObjects vector is not empty and matches the length, assign each stateObj to the call
	calls := make([]Call, len(targets))
	for i := range targets {
		stateObj := ""
		if len(stateObjects) == len(targets) {
			stateObj = stateObjects[i]
		}
		calls[i] = Call{
			Target:       targets[i],
			ModuleName:   moduleNames[i],
			FunctionName: functionNames[i],
			Data:         datas[i],
			StateObj:     stateObj,
		}
	}

	return calls, nil
}
