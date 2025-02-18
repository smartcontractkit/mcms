package sdk

type DecodedOperation interface {
	MethodName() string // MethodName returns the name of the method.
	Keys() []string     // InputNames returns the names of the input arguments.
	Args() []any        // Args returns the values input arguments.

	// String returns a human readable representation of the decoded operation.
	//
	// The first return value is the method name.
	// The second return value is a string representation of the input arguments.
	// The third return value is an error if there was an issue generating the string.
	String() (string, string, error)
}
