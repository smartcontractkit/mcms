package types

// ChainMetadata defines the metadata for a chain.
type ChainMetadata struct {
	StartingOpCount uint64 `json:"startingOpCount"`
	MCMAddress      string `json:"mcmAddress"`
	// msigName is a differentiator/seed for supporting
	// multiple multisigs with a single deployed program
	// only applicable to solana
	MSIGName string `json:"msigName"`
}
