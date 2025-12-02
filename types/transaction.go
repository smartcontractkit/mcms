package types //nolint:revive

// TransactionResult represents a generic blockchain transaction.
// It contains the hash of the transaction and the transaction itself.
// Users of this struct should cast the transaction to the appropriate type.
type TransactionResult struct {
	Hash        string `json:"hash"`
	ChainFamily string `json:"chainFamily"`
	RawData     any    `json:"rawData"`
}

func NewTransactionResult(hash string, rawData any, cf string) TransactionResult {
	return TransactionResult{
		Hash:        hash,
		ChainFamily: cf,
		RawData:     rawData,
	}
}
