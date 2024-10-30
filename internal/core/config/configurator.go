package config

// R in this case is the chain-specific struct thats the set of inputs
// required to make a SetConfig call
type Configurator[R any] interface {
	// Returns a string with the transaction hash
	SetConfigInputs(contract string, config Config) (R, error)
	// Maps to the chain-agnostic config
	ToConfig(onchainConfig R) (*Config, error)
}
