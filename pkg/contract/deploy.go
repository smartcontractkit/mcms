package contract

// DeployFunc defines a function type that deploys a contract, returning the deployment result.
//
// The result is defined by the type parameter R and is to be implemented by chain sdk. Typically,
// the address of the deployed contract and it's transaction data are returned in the result to
// provide the caller with the necessary information for interacting with the contract.
//
// Each supported blockchain family should implement a DeployFunc for every contract required to support
// their MCMS implementation.
type DeployFunc[R any] func() (R, error)

// Deploy deploys a contract by calling the provided function and a result.
func Deploy[R any](deployFunc DeployFunc[R]) (R, error) {
	return deployFunc()
}
