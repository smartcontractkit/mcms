package mcms

// Validator interface used to validate the fields of a chain operation across different chains.
type Validator interface {
	Validate() error
}
