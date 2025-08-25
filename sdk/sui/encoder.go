package sui

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/aptos-labs/aptos-go-sdk/bcs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var (
	// TODO: Change to SUI-specific domain separators when they are updated in the Move contract
	// Currently using the same as Aptos as per the Move contract TODOs
	mcmDomainSeparatorOp       = crypto.Keccak256([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_OP_APTOS"))
	mcmDomainSeparatorMetadata = crypto.Keccak256([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA_APTOS"))
)

// AdditionalFields represents the additional fields in Sui MCMS operations
type AdditionalFields struct {
	ModuleName string `json:"module_name"`
	Function   string `json:"function"`
	StateObj   string `json:"state_obj,omitempty"` // Needed for calling `mcms_entrypoint`
}

var _ sdk.Encoder = &Encoder{}

type Encoder struct {
	ChainSelector        types.ChainSelector
	TxCount              uint64
	OverridePreviousRoot bool
}

func NewEncoder(
	chainSelector types.ChainSelector,
	txCount uint64,
	overridePreviousRoot bool,
) *Encoder {
	return &Encoder{
		ChainSelector:        chainSelector,
		TxCount:              txCount,
		OverridePreviousRoot: overridePreviousRoot,
	}
}

func (e *Encoder) HashOperation(opCount uint32, metadata types.ChainMetadata, op types.Operation) (common.Hash, error) {
	chainID, err := chain_selectors.SuiChainIdFromSelector(uint64(e.ChainSelector))
	if err != nil {
		return common.Hash{}, err
	}
	chainIDBig := (&big.Int{}).SetUint64(chainID)
	toAddress, err := AddressFromHex(op.Transaction.To)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to parse To address %q: %w", op.Transaction.To, err)
	}
	additionalFields := AdditionalFields{}
	if unmarshalErr := json.Unmarshal(op.Transaction.AdditionalFields, &additionalFields); unmarshalErr != nil {
		return common.Hash{}, fmt.Errorf("failed to unmarshal additional fields: %w", unmarshalErr)
	}
	var additionalFieldsMetadata AdditionalFieldsMetadata
	if len(metadata.AdditionalFields) > 0 {
		if unmarshalErr := json.Unmarshal(metadata.AdditionalFields, &additionalFieldsMetadata); unmarshalErr != nil {
			return common.Hash{}, fmt.Errorf("failed to unmarshal additional fields metadata: %w", unmarshalErr)
		}
	}

	mcmsAddress, err := AddressFromHex(metadata.MCMAddress)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to decode mcms package ID: %w", err)
	}

	ser := bcs.Serializer{}
	ser.FixedBytes(mcmDomainSeparatorOp)
	ser.U8(uint8(additionalFieldsMetadata.Role))
	ser.U256(*chainIDBig)
	ser.FixedBytes(mcmsAddress.Bytes())
	ser.U64(uint64(opCount))
	ser.FixedBytes(toAddress.Bytes())
	ser.WriteString(additionalFields.ModuleName)
	ser.WriteString(additionalFields.Function)
	ser.WriteBytes(op.Transaction.Data)

	return crypto.Keccak256Hash(ser.ToBytes()), nil
}

func (e *Encoder) HashMetadata(metadata types.ChainMetadata) (common.Hash, error) {
	chainID, err := chain_selectors.SuiChainIdFromSelector(uint64(e.ChainSelector))
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get chain ID from selector %d: %w", e.ChainSelector, err)
	}
	chainIDBig := (&big.Int{}).SetUint64(chainID)

	if len(metadata.AdditionalFields) == 0 {
		return common.Hash{}, fmt.Errorf("additional fields metadata is empty")
	}
	var additionalFieldsMetadata AdditionalFieldsMetadata
	if unmarshalErr := json.Unmarshal(metadata.AdditionalFields, &additionalFieldsMetadata); unmarshalErr != nil {
		return common.Hash{}, fmt.Errorf("failed to unmarshal additional fields metadata: %w", unmarshalErr)
	}

	mcmsAddress, err := AddressFromHex(metadata.MCMAddress)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to decode mcms package ID: %w", err)
	}

	ser := bcs.Serializer{}
	ser.FixedBytes(mcmDomainSeparatorMetadata)
	ser.U8(uint8(additionalFieldsMetadata.Role))
	ser.U256(*chainIDBig)
	ser.FixedBytes(mcmsAddress.Bytes())
	ser.U64(metadata.StartingOpCount)
	ser.U64(metadata.StartingOpCount + e.TxCount)
	ser.Bool(e.OverridePreviousRoot)

	return crypto.Keccak256Hash(ser.ToBytes()), nil
}

func SerializeTimelockScheduleBatch(targets [][]byte,
	moduleNames []string,
	functionNames []string,
	datas [][]byte,
	predecessor []byte,
	salt []byte,
	delay uint64) ([]byte, error) {
	return bcs.SerializeSingle(func(ser *bcs.Serializer) {
		// Serialize targets vector
		//nolint:gosec
		ser.Uleb128(uint32(len(targets)))
		for _, target := range targets {
			ser.FixedBytes(target)
		}

		// Write module names
		//nolint:gosec
		ser.Uleb128(uint32(len(moduleNames)))
		for _, moduleName := range moduleNames {
			ser.WriteString(moduleName)
		}

		// Write function names
		//nolint:gosec
		ser.Uleb128(uint32(len(functionNames)))
		for _, functionName := range functionNames {
			ser.WriteString(functionName)
		}

		// Write data
		//nolint:gosec
		ser.Uleb128(uint32(len(datas)))
		for _, data := range datas {
			ser.WriteBytes(data)
		}

		ser.WriteBytes(predecessor)
		ser.WriteBytes(salt)
		ser.U64(delay)
	})
}

func SerializeTimelockBypasserExecuteBatch(stateObjects []string, targets [][]byte,
	moduleNames []string,
	functionNames []string,
	datas [][]byte) ([]byte, error) {
	return bcs.SerializeSingle(func(ser *bcs.Serializer) {
		// Serialize stateObjects vector
		ser.Uleb128(uint32(len(stateObjects)))
		for _, stateObj := range stateObjects {
			ser.WriteString(stateObj)
		}

		// Serialize targets vector
		//nolint:gosec
		ser.Uleb128(uint32(len(targets)))
		for _, target := range targets {
			ser.FixedBytes(target)
		}

		// Write module names
		//nolint:gosec
		ser.Uleb128(uint32(len(moduleNames)))
		for _, moduleName := range moduleNames {
			ser.WriteString(moduleName)
		}

		// Write function names
		//nolint:gosec
		ser.Uleb128(uint32(len(functionNames)))
		for _, functionName := range functionNames {
			ser.WriteString(functionName)
		}

		// Write data
		//nolint:gosec
		ser.Uleb128(uint32(len(datas)))
		for _, data := range datas {
			ser.WriteBytes(data)
		}
	})
}

func RemoveStateObjectsFromBypassData(data []byte) ([]byte, error) {
	calls, err := DeserializeTimelockBypasserExecuteBatch(data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize timelock bypasser execute batch: %w", err)
	}

	targets := make([][]byte, len(calls))
	for i, call := range calls {
		targets[i] = call.Target
	}

	moduleNames := make([]string, len(calls))
	for i, call := range calls {
		moduleNames[i] = call.ModuleName
	}

	functionNames := make([]string, len(calls))
	for i, call := range calls {
		functionNames[i] = call.FunctionName
	}

	datas := make([][]byte, len(calls))
	for i, call := range calls {
		datas[i] = call.Data
	}

	return bcs.SerializeSingle(func(ser *bcs.Serializer) {
		// Serialize targets vector
		//nolint:gosec
		ser.Uleb128(uint32(len(targets)))
		for _, target := range targets {
			ser.FixedBytes(target)
		}

		// Write module names
		//nolint:gosec
		ser.Uleb128(uint32(len(moduleNames)))
		for _, moduleName := range moduleNames {
			ser.WriteString(moduleName)
		}

		// Write function names
		//nolint:gosec
		ser.Uleb128(uint32(len(functionNames)))
		for _, functionName := range functionNames {
			ser.WriteString(functionName)
		}

		// Write data
		//nolint:gosec
		ser.Uleb128(uint32(len(datas)))
		for _, data := range datas {
			ser.WriteBytes(data)
		}
	})
}
