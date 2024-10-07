package contract

// DeployFunc defines a function type that deploys a contract, returning the address of the deployed
// contract and the transaction data.
//
// Each supported blockchain family should implement a DeployFunc for every contract required to support
// their MCMS implementation.
type DeployFunc[A any, T any] func() (A, T, error)

// Deploy deploys a contract by calling the provided function and returns the address of the
// deployed contract and the transaction data.
func Deploy[A any, T any](deployFunc DeployFunc[A, T]) (A, T, error) {
	return deployFunc()
}
