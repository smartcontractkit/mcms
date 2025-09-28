package ton

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/mcms/types"
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

	if err := fields.Validate(); err != nil {
		return err
	}

	return nil
}

type AdditionalFields struct {
	Value *big.Int `json:"value"`
}

// Validate ensures the TON-specific fields are correct
func (f AdditionalFields) Validate() error {
	if f.Value == nil || f.Value.Sign() < 0 {
		return fmt.Errorf("invalid TON value: %v", f.Value)
	}

	return nil
}

func NewTransaction(
	to address.Address,
	body *cell.Slice,
	value *big.Int,
	contractType string,
	tags []string,
) (types.Transaction, error) {
	additionalFields, err := json.Marshal(AdditionalFields{
		Value: value,
	})
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
			ContractType: contractType,
			Tags:         tags,
		},
	}, nil
}
