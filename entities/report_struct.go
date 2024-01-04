package entities

type Report struct {
	TasksInQueue             int
	ExecutingRegularWorkers  int
	ExecutingPriorityWorkers int
	IdleRegularWorkers       int
	IdlePriorityWorkers      int
}
