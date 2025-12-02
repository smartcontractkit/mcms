package sui

import (
	"encoding/json"
	"fmt"

	"github.com/aptos-labs/aptos-go-sdk/bcs"
	"github.com/block-vision/sui-go-sdk/models"

	"github.com/smartcontractkit/chainlink-sui/bindgen/function"
	"github.com/smartcontractkit/chainlink-sui/bindings/bind"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

const (
	// SuiAddressLength represents the byte length of Sui addresses
	SuiAddressLength = 32
	// MinimumResultLength is the minimum expected length for certain results
	MinimumResultLength = 2
)

type Decoder struct{}

var _ sdk.Decoder = &Decoder{}

func NewDecoder() *Decoder {
	return &Decoder{}
}

func (d Decoder) Decode(tx types.Transaction, contractInterfaces string) (sdk.DecodedOperation, error) {
	var fInfos []function.FunctionInfo
	if err := json.Unmarshal([]byte(contractInterfaces), &fInfos); err != nil {
		return nil, err
	}

	var additionalFields AdditionalFields
	if err := json.Unmarshal(tx.AdditionalFields, &additionalFields); err != nil {
		return nil, fmt.Errorf("failed to unmarshal additional fields: %w", err)
	}
	if err := additionalFields.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate additional fields: %w", err)
	}

	// Find information about the function being called
	var functionInfo *function.FunctionInfo
	for _, fInfo := range fInfos {
		if fInfo.Module == additionalFields.ModuleName && fInfo.Name == additionalFields.Function {
			functionInfo = &fInfo
			break
		}
	}
	if functionInfo == nil {
		return nil, fmt.Errorf("could not find function in contractInterfaces for %s::%s", additionalFields.ModuleName, additionalFields.Function)
	}

	// Extract parameters' names and types and deserialize transaction data
	parNames, parTypes := functionInfo.GetParameters()
	parValues, err := bind.DeserializeBCS(tx.Data, parTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize BCS data: %w", err)
	}

	return NewDecodedOperation(additionalFields.ModuleName, additionalFields.Function, parNames, parValues)
}

type Call struct {
	Target           []byte
	ModuleName       string
	FunctionName     string
	Data             []byte
	StateObj         string
	TypeArgs         []string
	CompiledModules  [][]byte            // For upgrade operations
	Dependencies     []models.SuiAddress // For upgrade operations
	PackageToUpgrade string              // For upgrade operations
}

func deserializeTimelockBypasserExecuteBatch(data []byte) ([]Call, error) {
	deserializer := bcs.NewDeserializer(data)

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
		calls[i] = Call{
			Target:       targets[i],
			ModuleName:   moduleNames[i],
			FunctionName: functionNames[i],
			Data:         datas[i],
		}
	}

	return calls, nil
}
