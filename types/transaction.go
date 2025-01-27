package types

// NativeTransaction represents a generic blockchain transaction.
// It contains the hash of the transaction and the transaction itself.
// Users of this struct should cast the transaction to the appropriate type.
type NativeTransaction struct {
	Hash string `json:"hash"`
	Tx   any    `json:"tx"`
}
