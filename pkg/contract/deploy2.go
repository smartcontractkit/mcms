package contract

type Deployer2[R any] interface {
	Deploy() (R, error)
}

// Deploy deploys a contract by calling the provided function and a result.
func Deploy2[R any](deployer Deployer2[R]) (R, error) {
	return deployer.Deploy()
}
