package internal

import (
	"fmt"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
)

type UnkownChainSelectorFamilyError struct {
	ChainSelector uint64
	ChainFamily   string
}

var SupportedChainSelectorFamilies = []string{
	chain_selectors.FamilyEVM,
	chain_selectors.FamilySolana,
}

func (e UnkownChainSelectorFamilyError) Error() string {
	return fmt.Sprintf("unknown chain selector family: %d with family %s. Supported families are %v", e.ChainSelector, e.ChainFamily, SupportedChainSelectorFamilies)
}
