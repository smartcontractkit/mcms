package timelock

type Inspector interface {
	GetProposers(timelockAddress string)
	GetExecutors(timelockAddress string)
	GetBypassers(timelockAddress string)
	GetCancellers(timelockAddress string)
	isOperation(timelockAddress string)
	isOperationPending(timelockAddress string)
	isOperationReady(timelockAddress string)
	isOperationDone(timelockAddress string)
}
