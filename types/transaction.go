package types

// MinedTransaction represents a generic blockchain transaction.
// It contains the hash of the transaction and the transaction itself.
// Users of this struct should cast the transaction to the appropriate type.
type MinedTransaction struct {
	Hash string      `json:"hash"`
	Tx   interface{} `json:"tx"`
}
