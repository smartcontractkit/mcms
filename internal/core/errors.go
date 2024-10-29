package core

import (
	"fmt"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
)

type UnknownChainSelectorFamilyError struct {
	ChainSelector uint64
	ChainFamily   string
}

var SupportedChainSelectorFamilies = []string{
	chain_selectors.FamilyEVM,
	chain_selectors.FamilySolana,
}

func (e UnknownChainSelectorFamilyError) Error() string {
	return fmt.Sprintf("unknown chain selector family: %d with family %s. Supported families are %v", e.ChainSelector, e.ChainFamily, SupportedChainSelectorFamilies)
}

func NewUnknownChainSelectorFamilyError(selector uint64, family string) *UnknownChainSelectorFamilyError {
	return &UnknownChainSelectorFamilyError{
		ChainSelector: selector,
		ChainFamily:   family,
	}
}
