package types

// ChainSelector is a unique identifier for a chain.
//
// These values are defined in the chain-selectors dependency.
// https://github.com/smartcontractkit/chain-selectors
type ChainSelector uint64

// ChainMetadata defines the metadata for a chain.
type ChainMetadata struct {
	StartingOpCount uint64 `json:"startingOpCount"`
	MCMAddress      string `json:"mcmAddress"`
}
