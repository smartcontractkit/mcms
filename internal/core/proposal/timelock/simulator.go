package timelock

type Simulator interface {
	Simulate(op BatchChainOperation, predecessor string, salt string) (string, error)
}
