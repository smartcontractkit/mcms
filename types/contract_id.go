package types

type ContractID interface {
	String() string
	ChainFamily() string
}
