package timelock

type ScheduledExecutor interface {
	Execute(op BatchChainOperation, predecessor string, salt string) (string, error)
}
