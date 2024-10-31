package sdk

// Validator used to validate the fields of a chain operation across different chains.
//
// Implement this to provide chain-specific validation for additional fields.
type Validator interface {
	Validate() error
}
