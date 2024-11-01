package types

// ChainMetadata defines the metadata for a chain.
type ChainMetadata struct {
	StartingOpCount uint64 `json:"startingOpCount"`
	MCMAddress      string `json:"mcmAddress"`
}
