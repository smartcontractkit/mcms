package ton

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/Masterminds/semver/v3"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"
)

func ValidateAdditionalFields(additionalFields json.RawMessage) error {
	fields := AdditionalFields{
		Value: big.NewInt(0),
	}
	if len(additionalFields) != 0 {
		if err := json.Unmarshal(additionalFields, &fields); err != nil {
			return fmt.Errorf("failed to unmarshal TON additional fields: %w", err)
		}
	}

	return fields.Validate()
}

type AdditionalFields struct {
	ContractTypeFull tvm.FullyQualifiedName `json:"contractTypeFull,omitempty"`
	Value            *big.Int               `json:"value"`
}

// Validate ensures the TON-specific fields are correct
func (f AdditionalFields) Validate() error {
	if f.Value == nil || f.Value.Sign() < 0 {
		return fmt.Errorf("invalid TON value: %v", f.Value)
	}

	return nil
}

// TODO: should use a generic type and an interface to define this (method to create generic types.Transaction from a specific type [S])
func NewTransaction(
	to *address.Address,
	body *cell.Slice,
	value *big.Int,
	contractType string,
	contractVersion *semver.Version,
	contractFQN tvm.FullyQualifiedName,
	tags []string,
) (types.Transaction, error) {
	additionalFields, err := json.Marshal(AdditionalFields{Value: value, ContractTypeFull: contractFQN})
	if err != nil {
		return types.Transaction{}, fmt.Errorf("failed to marshal additional fields: %w", err)
	}

	bodyCell, err := body.ToCell()
	if err != nil {
		return types.Transaction{}, fmt.Errorf("failed to convert body to cell: %w", err)
	}
	data := bodyCell.ToBOC()

	return types.Transaction{
		To:               to.String(),
		Data:             data,
		AdditionalFields: additionalFields,
		OperationMetadata: types.OperationMetadata{
			ContractType:    contractType,
			ContractVersion: contractVersion,
			Tags:            tags,
		},
	}, nil
}
