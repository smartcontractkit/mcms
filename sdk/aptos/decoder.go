package aptos

import (
	"encoding/json"
	"fmt"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/smartcontractkit/chainlink-aptos/bindings/bind"
	"github.com/smartcontractkit/chainlink-aptos/relayer/txm"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

type Decoder struct{}

var _ sdk.Decoder = &Decoder{}

func NewDecoder() *Decoder {
	return &Decoder{}
}

func (d Decoder) Decode(tx types.Transaction, contractInterfaces string) (sdk.DecodedOperation, error) {
	functionInfo, err := bind.ParseFunctionInfo(contractInterfaces)
	if err != nil {
		return nil, fmt.Errorf("failed to parse function info: %w", err)
	}

	var additionalFields AdditionalFields
	if err = json.Unmarshal(tx.AdditionalFields, &additionalFields); err != nil {
		return nil, fmt.Errorf("failed to unmarshal additional fields: %w", err)
	}
	if err = additionalFields.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate additional fields: %w", err)
	}

	for _, info := range functionInfo {
		if info.Package == additionalFields.PackageName && info.Module == additionalFields.ModuleName && info.Name == additionalFields.Function {
			paramKeys := make([]string, len(info.Parameters))
			typeTags := make([]aptos.TypeTag, len(info.Parameters))
			for i, parameter := range info.Parameters {
				typeTags[i], err = txm.CreateTypeTag(parameter.Type)
				if err != nil {
					return nil, fmt.Errorf("failed to create type tag: %w", err)
				}
				paramKeys[i] = parameter.Name
			}

			data, err := txm.GetBcsValues(tx.Data, typeTags...)
			if err != nil {
				return nil, fmt.Errorf("failed to get bcs values: %w", err)
			}

			return NewDecodedOperation(additionalFields.PackageName, additionalFields.ModuleName, additionalFields.Function, paramKeys, data)
		}
	}

	return nil, fmt.Errorf("could not find function info for %s::%s::%s", additionalFields.PackageName, additionalFields.ModuleName, additionalFields.Function)
}
