package types

// TransactionResult represents a generic blockchain transaction.
// It contains the hash of the transaction and the transaction itself.
// Users of this struct should cast the transaction to the appropriate type.
type TransactionResult struct {
	Hash        string `json:"hash"`
	ChainFamily string `json:"chainFamily"`
	RawData     any    `json:"tx"`
}

func NewTransactionResult(hash string, tx any, cf string) TransactionResult {
	return TransactionResult{
		Hash:        hash,
		ChainFamily: cf,
		RawData:     tx,
	}
}
