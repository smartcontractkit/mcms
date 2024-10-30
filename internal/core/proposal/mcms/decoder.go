package mcms

// Defined on a per chain level
type Decoder interface {
	// Returns: (MethodName, Args, error)
	Decode(operation ChainOperation, abiStr string) (string, string, error)
}
