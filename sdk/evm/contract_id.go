package evm

import (
	"fmt"

	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/types"
)

var _ types.ContractID = (*EVMContractID)(nil)

type EVMContractID struct {
	Address string
}

func (i EVMContractID) String() string {
	return i.Address
}

func (i EVMContractID) ChainFamily() string {
	return cselectors.FamilyEVM
}

func NewEVMContractID(address string) *EVMContractID {
	return &EVMContractID{Address: address}
}

func FromContractID(cid types.ContractID) (*EVMContractID, error) {
	if cid.ChainFamily() != cselectors.FamilySolana {
		return nil, fmt.Errorf("invalid contract id: %w", types.ErrUnsupportedChainFamily)
	}

	return cid.(*EVMContractID), nil
}

func AddressFromContractID(cid types.ContractID) (string, error) {
	evmContractID, err := FromContractID(cid)
	if err != nil {
		return "", err
	}

	return evmContractID.Address, nil
}
