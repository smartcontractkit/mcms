package chaintest

import (
	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/types"
)

var (
	Chain1RawSelector = cselectors.GETH_TESTNET.Selector       // 3379446385462418246
	Chain1Selector    = types.ChainSelector(Chain1RawSelector) // 3379446385462418246
	Chain1EVMID       = cselectors.GETH_TESTNET.EvmChainID     // 1337

	Chain2RawSelector = cselectors.ETHEREUM_TESTNET_SEPOLIA.Selector   // 16015286601757825753
	Chain2Selector    = types.ChainSelector(Chain2RawSelector)         // 16015286601757825753
	Chain2EVMID       = cselectors.ETHEREUM_TESTNET_SEPOLIA.EvmChainID // 11155111

	Chain3RawSelector = cselectors.ETHEREUM_TESTNET_SEPOLIA_BASE_1.Selector   // 10344971235874465080
	Chain3Selector    = types.ChainSelector(Chain3RawSelector)                // 10344971235874465080
	Chain3EVMID       = cselectors.ETHEREUM_TESTNET_SEPOLIA_BASE_1.EvmChainID // 84532

	// TestInvalidChainSelector is a chain selector that doesn't exist.
	TestInvalidChainSelector = types.ChainSelector(0)
)
